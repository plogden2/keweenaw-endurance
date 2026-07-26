package bridgeapp

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/keweenaw-endurance/backend/internal/bridge"
)

// CatalogEvent is a selectable event for the reader GUI.
type CatalogEvent struct {
	ID   string
	Name string
}

// CatalogRace is a selectable race (plus All-races sentinel handled in UI).
type CatalogRace struct {
	ID                 string
	Name               string
	FinishCheckpointID string
}

// CatalogCheckpoint is a selectable checkpoint for a race.
type CatalogCheckpoint struct {
	ID   string
	Name string
	Type string
}

// FetchCatalogEvents lists hosted events (paginated).
func FetchCatalogEvents(auth *bridge.HostedAuth, client *http.Client) ([]CatalogEvent, error) {
	if auth == nil {
		return nil, fmt.Errorf("auth not configured")
	}
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	var out []CatalogEvent
	page := 1
	for {
		url := fmt.Sprintf("%s/api/events?page=%d&limit=200", auth.BaseURL, page)
		raw, err := catalogGET(auth, client, url)
		if err != nil {
			return nil, err
		}
		var wrapped struct {
			Data  []apiEvent `json:"data"`
			Total int        `json:"total"`
		}
		if err := json.Unmarshal(raw, &wrapped); err != nil {
			return nil, err
		}
		for _, e := range wrapped.Data {
			name := strings.TrimSpace(e.Name)
			if name == "" {
				name = e.ID
			}
			out = append(out, CatalogEvent{ID: e.ID, Name: name})
		}
		if page*200 >= wrapped.Total || len(wrapped.Data) == 0 {
			break
		}
		page++
	}
	return out, nil
}

// FetchCatalogRaces lists races for an event with finish checkpoint IDs when found.
func FetchCatalogRaces(auth *bridge.HostedAuth, client *http.Client, eventID string) ([]CatalogRace, error) {
	if auth == nil {
		return nil, fmt.Errorf("auth not configured")
	}
	eventID = strings.TrimSpace(eventID)
	if eventID == "" {
		return nil, fmt.Errorf("event_id is required")
	}
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	var out []CatalogRace
	page := 1
	for {
		url := fmt.Sprintf("%s/api/races?event_id=%s&page=%d&limit=200", auth.BaseURL, eventID, page)
		raw, err := catalogGET(auth, client, url)
		if err != nil {
			return nil, err
		}
		var wrapped struct {
			Data  []apiRace `json:"data"`
			Total int       `json:"total"`
		}
		if err := json.Unmarshal(raw, &wrapped); err != nil {
			return nil, err
		}
		for _, r := range wrapped.Data {
			finishID := ""
			if cps, err := fetchCheckpoints(client, auth.BaseURL, r.ID); err == nil {
				for _, cp := range cps {
					if strings.EqualFold(cp.CheckpointType, "finish") {
						finishID = cp.ID
						break
					}
				}
			}
			name := strings.TrimSpace(r.Name)
			if name == "" {
				name = r.ID
			}
			out = append(out, CatalogRace{ID: r.ID, Name: name, FinishCheckpointID: finishID})
		}
		if page*200 >= wrapped.Total || len(wrapped.Data) == 0 {
			break
		}
		page++
	}
	return out, nil
}

// FetchCatalogCheckpoints lists checkpoints for a race.
func FetchCatalogCheckpoints(auth *bridge.HostedAuth, client *http.Client, raceID string) ([]CatalogCheckpoint, error) {
	if auth == nil {
		return nil, fmt.Errorf("auth not configured")
	}
	raceID = strings.TrimSpace(raceID)
	if raceID == "" {
		return nil, fmt.Errorf("race_id is required")
	}
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	cps, err := fetchCheckpoints(client, auth.BaseURL, raceID)
	if err != nil {
		return nil, err
	}
	out := make([]CatalogCheckpoint, 0, len(cps))
	for _, cp := range cps {
		name := strings.TrimSpace(cp.Name)
		if name == "" {
			name = cp.CheckpointType
		}
		if name == "" {
			name = cp.ID
		}
		out = append(out, CatalogCheckpoint{ID: cp.ID, Name: name, Type: cp.CheckpointType})
	}
	return out, nil
}

func catalogGET(auth *bridge.HostedAuth, client *http.Client, url string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	if auth.BearerToken != "" {
		req.Header.Set("Authorization", "Bearer "+auth.BearerToken)
	}
	if auth.BridgeToken != "" {
		req.Header.Set("X-Bridge-Token", auth.BridgeToken)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	raw, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("catalog request failed: %s", strings.TrimSpace(string(raw)))
	}
	return raw, nil
}

// SeedCatalogFromBluffet builds catalog races from offline seed.
func SeedCatalogFromBluffet() (CatalogEvent, []CatalogRace) {
	seed := SeedBluffetDetails()
	ev := CatalogEvent{ID: seed.EventID, Name: BluffetEventName}
	races := make([]CatalogRace, 0, len(seed.Races))
	for _, r := range seed.Races {
		races = append(races, CatalogRace{
			ID:                 r.RaceID,
			Name:               r.Name,
			FinishCheckpointID: r.FinishCheckpointID,
		})
	}
	return ev, races
}
