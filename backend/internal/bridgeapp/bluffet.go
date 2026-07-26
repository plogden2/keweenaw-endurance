package bridgeapp

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
)

// Canonical All You Can East Bluffet 2026 IDs (seed + production short forms).
const (
	BluffetEventIDFull = "1441674d-a011-471a-a601-722b88b117f5"
	BluffetEventIDShort = "b117f5"
	BluffetEventName   = "All You Can East Bluffet"
	DefaultOrganizerPIN = "1738"
)

// BluffetRace is one Bluffet distance with its finish (Lap Check) checkpoint.
type BluffetRace struct {
	Name           string
	RaceID         string
	FinishCheckpointID string
}

// BluffetDetails holds event + races discovered from hosted (or seed fallback).
type BluffetDetails struct {
	EventID string
	Races   []BluffetRace
}

// SeedBluffetDetails returns offline-safe Bluffet IDs from the hardware seed.
func SeedBluffetDetails() BluffetDetails {
	return BluffetDetails{
		EventID: BluffetEventIDFull,
		Races: []BluffetRace{
			{Name: "12 Hour", RaceID: "17da3ba1-2e09-4eb1-aeb3-d9dd5b6a394e", FinishCheckpointID: "81ca12c0-dfec-512e-b605-7e1dfbcb63f5"},
			{Name: "6 Hour", RaceID: "209769a1-f723-4f70-ae90-466a46338684", FinishCheckpointID: "5b7e8d76-8cc4-5e17-9147-9ed99a8df6fa"},
			{Name: "90-Minute Kids", RaceID: "0e45ee85-800c-4e1f-a95b-4b92462e790a", FinishCheckpointID: "31b14fd1-a863-57e1-b90f-45f1593cdd49"},
		},
	}
}

// ApplyBluffetDefaults sets All You Can East Bluffet race-day fields.
// Default is All races (empty RaceID/CheckpointID) so one finish reader scores
// every distance. A saved single-race selection is preserved. Secrets are never overwritten.
func ApplyBluffetDefaults(cfg *Config) {
	if cfg == nil {
		return
	}
	seed := SeedBluffetDetails()
	firstLaunch := strings.TrimSpace(cfg.EventID) == ""
	cfg.HostedAPIURL = "https://www.keweenawendurance.com"
	if cfg.OrganizerPIN == "" {
		cfg.OrganizerPIN = DefaultOrganizerPIN
	}
	cfg.DeviceID = "laptop-finish-1"
	cfg.EventID = seed.EventID
	if strings.TrimSpace(cfg.RaceID) != "" && len(seed.Races) > 0 {
		// Preserve a previously chosen Bluffet distance; refresh finish checkpoint.
		for _, r := range seed.Races {
			if r.RaceID == cfg.RaceID || strings.HasSuffix(r.RaceID, cfg.RaceID) || strings.HasSuffix(cfg.RaceID, r.RaceID) {
				cfg.RaceID = r.RaceID
				cfg.CheckpointID = r.FinishCheckpointID
				break
			}
		}
	} else {
		// All races (event finish) — intentional empty race/checkpoint.
		cfg.RaceID = ""
		cfg.CheckpointID = ""
	}
	if cfg.ProxmarkPort == "" {
		cfg.ProxmarkPort = "COM3"
	}
	// First launch only: enable Proxmark hardware by default.
	if firstLaunch && !cfg.BridgeMock {
		cfg.RFIDHardware = true
	}
	normalizeConfig(cfg)
}

// ResolveBridgeToken returns BRIDGE_TOKEN env, else gcloud Secret Manager value.
func ResolveBridgeToken() (string, error) {
	if t := strings.TrimSpace(os.Getenv("BRIDGE_TOKEN")); t != "" {
		return t, nil
	}
	cmd := exec.Command("gcloud", "secrets", "versions", "access", "latest", "--secret=keweenaw-bridge-token")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("gcloud bridge token: %w: %s", err, strings.TrimSpace(string(out)))
	}
	tok := strings.TrimSpace(string(out))
	if tok == "" {
		return "", fmt.Errorf("bridge token secret was empty")
	}
	return tok, nil
}

// FetchBluffetDetails loads Bluffet event/races/finish checkpoints from hosted.
// Falls back to seed IDs when the network or parse fails.
func FetchBluffetDetails(baseURL string, client *http.Client) (BluffetDetails, error) {
	seed := SeedBluffetDetails()
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		return seed, fmt.Errorf("hosted API URL is empty")
	}
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}

	events, err := fetchEvents(client, baseURL)
	if err != nil {
		return seed, err
	}
	var eventID string
	for _, e := range events {
		if strings.EqualFold(strings.TrimSpace(e.Name), BluffetEventName) ||
			e.ID == BluffetEventIDShort || e.ID == BluffetEventIDFull ||
			strings.HasSuffix(BluffetEventIDFull, e.ID) {
			eventID = e.ID
			// Prefer full seed UUID for local CSV paths when API returns short id.
			if e.ID == BluffetEventIDShort || strings.HasSuffix(BluffetEventIDFull, e.ID) {
				eventID = BluffetEventIDFull
			}
			break
		}
	}
	if eventID == "" {
		return seed, fmt.Errorf("bluffet event not found on hosted")
	}

	detail, err := fetchEventDetail(client, baseURL, eventID)
	if err != nil {
		// Short id from list if full detail fails with full UUID.
		detail, err = fetchEventDetail(client, baseURL, BluffetEventIDShort)
		if err != nil {
			return seed, err
		}
	}

	out := BluffetDetails{EventID: BluffetEventIDFull, Races: nil}
	for _, r := range detail.Races {
		cps, err := fetchCheckpoints(client, baseURL, r.ID)
		if err != nil {
			continue
		}
		finishID := ""
		for _, cp := range cps {
			if strings.EqualFold(cp.CheckpointType, "finish") {
				finishID = cp.ID
				break
			}
		}
		if finishID == "" {
			continue
		}
		out.Races = append(out.Races, BluffetRace{
			Name:               r.Name,
			RaceID:             r.ID,
			FinishCheckpointID: finishID,
		})
	}
	if len(out.Races) == 0 {
		return seed, fmt.Errorf("no bluffet races with finish checkpoints")
	}
	// Prefer 12 Hour first for default selection.
	out.Races = preferRaceOrder(out.Races)
	return out, nil
}

func preferRaceOrder(races []BluffetRace) []BluffetRace {
	var twelve, rest []BluffetRace
	for _, r := range races {
		if strings.Contains(strings.ToLower(r.Name), "12") {
			twelve = append(twelve, r)
		} else {
			rest = append(rest, r)
		}
	}
	return append(twelve, rest...)
}

type apiEvent struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type apiRace struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type apiCheckpoint struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	CheckpointType string `json:"checkpoint_type"`
}

func fetchEvents(client *http.Client, baseURL string) ([]apiEvent, error) {
	resp, err := client.Get(baseURL + "/api/events")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("events: %s", strings.TrimSpace(string(raw)))
	}
	var wrapped struct {
		Data []apiEvent `json:"data"`
	}
	if err := json.Unmarshal(raw, &wrapped); err != nil {
		return nil, err
	}
	return wrapped.Data, nil
}

func fetchEventDetail(client *http.Client, baseURL, eventID string) (struct {
	ID    string    `json:"id"`
	Races []apiRace `json:"races"`
}, error) {
	var out struct {
		ID    string    `json:"id"`
		Races []apiRace `json:"races"`
	}
	resp, err := client.Get(baseURL + "/api/events/" + eventID)
	if err != nil {
		return out, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return out, fmt.Errorf("event %s: %s", eventID, strings.TrimSpace(string(raw)))
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return out, err
	}
	return out, nil
}

func fetchCheckpoints(client *http.Client, baseURL, raceID string) ([]apiCheckpoint, error) {
	resp, err := client.Get(baseURL + "/api/races/" + raceID + "/checkpoints")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("checkpoints: %s", strings.TrimSpace(string(raw)))
	}
	var list []apiCheckpoint
	if err := json.Unmarshal(raw, &list); err == nil && len(list) > 0 {
		return list, nil
	}
	var wrapped struct {
		Data []apiCheckpoint `json:"data"`
	}
	if err := json.Unmarshal(raw, &wrapped); err != nil {
		return nil, err
	}
	return wrapped.Data, nil
}

// AutofillConfig merges Bluffet seed + optional live hosted details + bridge token.
// Empty fields only — does not overwrite saved non-empty values unless force is true.
func AutofillConfig(cfg *Config, force bool) (BluffetDetails, error) {
	ApplyBluffetDefaults(cfg)
	details := SeedBluffetDetails()

	if tok, err := ResolveBridgeToken(); err == nil && (force || cfg.BridgeToken == "") {
		cfg.BridgeToken = tok
	}

	live, err := FetchBluffetDetails(cfg.HostedAPIURL, nil)
	if err == nil && len(live.Races) > 0 {
		details = live
		if force || cfg.EventID == "" || cfg.EventID == BluffetEventIDFull || cfg.EventID == BluffetEventIDShort {
			cfg.EventID = details.EventID
		}
		if cfg.RaceID != "" {
			// Keep race id; refresh checkpoint if empty.
			for _, r := range details.Races {
				if r.RaceID == cfg.RaceID {
					if cfg.CheckpointID == "" {
						cfg.CheckpointID = r.FinishCheckpointID
					}
					break
				}
			}
		}
		// Empty RaceID means All races — do not autofill a single distance.
	} else if force && cfg.EventID == "" {
		cfg.EventID = details.EventID
	}
	normalizeConfig(cfg)
	return details, err
}
