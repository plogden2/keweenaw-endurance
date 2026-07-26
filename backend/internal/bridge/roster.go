package bridge

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

// RosterEntry is one racer chip binding in the event-wide cache.
type RosterEntry struct {
	Bib         string
	Name        string
	RaceID      string
	RaceName    string
	LogicalUUID string
	CheckpointID string // finish checkpoint when known
}

// RosterCache maps bib → one or more race entries (duplicate bibs across races).
type RosterCache struct {
	mu      sync.RWMutex
	byBib   map[string][]RosterEntry
	byUUID  map[string]RosterEntry
}

// NewRosterCache returns an empty roster cache.
func NewRosterCache() *RosterCache {
	return &RosterCache{
		byBib:  map[string][]RosterEntry{},
		byUUID: map[string]RosterEntry{},
	}
}

type rosterParticipant struct {
	ID         string   `json:"id"`
	BibNumber  string   `json:"bib_number"`
	FirstName  string   `json:"first_name"`
	LastName   string   `json:"last_name"`
	RFIDTagUID string   `json:"rfid_tag_uid"`
	TagUIDs    []string `json:"tag_uids"`
}

type apiRaceListItem struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Refresh loads participants for a single race (paginated, limit 200).
// Replaces prior cache contents with this race only.
func (c *RosterCache) Refresh(auth *HostedAuth, client *http.Client, raceID string) error {
	return c.RefreshRace(auth, client, raceID, "", "")
}

// RefreshRace loads one race into the cache (replacing all prior entries).
func (c *RosterCache) RefreshRace(auth *HostedAuth, client *http.Client, raceID, raceName, finishCheckpointID string) error {
	entries, err := fetchRaceParticipants(auth, client, raceID, raceName, finishCheckpointID)
	if err != nil {
		return err
	}
	c.replace(entries)
	return nil
}

// RefreshEvent loads participants for every race in the event (paginated).
func (c *RosterCache) RefreshEvent(auth *HostedAuth, client *http.Client, eventID string) error {
	if c == nil {
		return fmt.Errorf("roster cache is nil")
	}
	if auth == nil {
		return fmt.Errorf("auth not configured")
	}
	eventID = strings.TrimSpace(eventID)
	if eventID == "" {
		return fmt.Errorf("event_id is required")
	}
	if client == nil {
		client = &http.Client{Timeout: 20 * time.Second}
	}

	races, err := fetchRacesForEvent(auth, client, eventID)
	if err != nil {
		return err
	}
	var all []RosterEntry
	for _, r := range races {
		finishID, _ := fetchFinishCheckpointID(auth, client, r.ID)
		entries, err := fetchRaceParticipants(auth, client, r.ID, r.Name, finishID)
		if err != nil {
			return err
		}
		all = append(all, entries...)
	}
	c.replace(all)
	return nil
}

func (c *RosterCache) replace(entries []RosterEntry) {
	nextBib := map[string][]RosterEntry{}
	nextUUID := map[string]RosterEntry{}
	for _, e := range entries {
		bib := strings.TrimSpace(e.Bib)
		uid := strings.ToLower(strings.TrimSpace(e.LogicalUUID))
		if bib == "" || uid == "" {
			continue
		}
		e.Bib = bib
		e.LogicalUUID = uid
		nextBib[bib] = append(nextBib[bib], e)
		nextUUID[uid] = e
	}
	c.mu.Lock()
	c.byBib = nextBib
	c.byUUID = nextUUID
	c.mu.Unlock()
}

func fetchRacesForEvent(auth *HostedAuth, client *http.Client, eventID string) ([]apiRaceListItem, error) {
	var out []apiRaceListItem
	page := 1
	for {
		url := fmt.Sprintf("%s/api/races?event_id=%s&page=%d&limit=200", auth.BaseURL, eventID, page)
		raw, err := hostedGET(auth, client, url)
		if err != nil {
			return nil, err
		}
		var wrapped struct {
			Data  []apiRaceListItem `json:"data"`
			Total int               `json:"total"`
		}
		if err := json.Unmarshal(raw, &wrapped); err != nil {
			return nil, err
		}
		out = append(out, wrapped.Data...)
		if page*200 >= wrapped.Total || len(wrapped.Data) == 0 {
			break
		}
		page++
	}
	return out, nil
}

func fetchFinishCheckpointID(auth *HostedAuth, client *http.Client, raceID string) (string, error) {
	url := fmt.Sprintf("%s/api/races/%s/checkpoints", auth.BaseURL, raceID)
	raw, err := hostedGET(auth, client, url)
	if err != nil {
		return "", err
	}
	var list []struct {
		ID             string `json:"id"`
		CheckpointType string `json:"checkpoint_type"`
	}
	if err := json.Unmarshal(raw, &list); err != nil {
		var wrapped struct {
			Data []struct {
				ID             string `json:"id"`
				CheckpointType string `json:"checkpoint_type"`
			} `json:"data"`
		}
		if err2 := json.Unmarshal(raw, &wrapped); err2 != nil {
			return "", err
		}
		list = wrapped.Data
	}
	for _, cp := range list {
		if strings.EqualFold(cp.CheckpointType, "finish") {
			return cp.ID, nil
		}
	}
	if len(list) > 0 {
		return list[0].ID, nil
	}
	return "", nil
}

func fetchRaceParticipants(auth *HostedAuth, client *http.Client, raceID, raceName, finishCheckpointID string) ([]RosterEntry, error) {
	if auth == nil {
		return nil, fmt.Errorf("auth not configured")
	}
	raceID = strings.TrimSpace(raceID)
	if raceID == "" {
		return nil, fmt.Errorf("race_id is required")
	}
	if client == nil {
		client = &http.Client{Timeout: 20 * time.Second}
	}

	var out []RosterEntry
	page := 1
	for {
		url := fmt.Sprintf("%s/api/races/%s/participants?page=%d&limit=200", auth.BaseURL, raceID, page)
		raw, err := hostedGET(auth, client, url)
		if err != nil {
			return nil, err
		}
		var wrapped struct {
			Data  []rosterParticipant `json:"data"`
			Total int                 `json:"total"`
		}
		if err := json.Unmarshal(raw, &wrapped); err != nil {
			return nil, err
		}
		for _, p := range wrapped.Data {
			bib := strings.TrimSpace(p.BibNumber)
			uid := strings.TrimSpace(strings.ToLower(p.RFIDTagUID))
			if uid == "" {
				for _, t := range p.TagUIDs {
					if u := strings.TrimSpace(strings.ToLower(t)); u != "" {
						uid = u
						break
					}
				}
			}
			if bib == "" || uid == "" {
				continue
			}
			name := strings.TrimSpace(p.FirstName + " " + p.LastName)
			out = append(out, RosterEntry{
				Bib:          bib,
				Name:         name,
				RaceID:       raceID,
				RaceName:     raceName,
				LogicalUUID:  uid,
				CheckpointID: finishCheckpointID,
			})
		}
		if page*200 >= wrapped.Total || len(wrapped.Data) == 0 {
			break
		}
		page++
	}
	return out, nil
}

func hostedGET(auth *HostedAuth, client *http.Client, url string) ([]byte, error) {
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
		return nil, fmt.Errorf("roster request failed: %s", strings.TrimSpace(string(raw)))
	}
	return raw, nil
}

// EntriesForBib returns all cached entries for a bib (may be multiple races).
func (c *RosterCache) EntriesForBib(bib string) []RosterEntry {
	if c == nil {
		return nil
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	src := c.byBib[strings.TrimSpace(bib)]
	out := make([]RosterEntry, len(src))
	copy(out, src)
	return out
}

// LogicalUUIDForBib returns the UUID when exactly one entry exists for the bib.
func (c *RosterCache) LogicalUUIDForBib(bib string) (string, bool) {
	entries := c.EntriesForBib(bib)
	if len(entries) != 1 {
		return "", false
	}
	return entries[0].LogicalUUID, entries[0].LogicalUUID != ""
}

// EntryForUUID returns the roster row for a logical UUID when cached.
func (c *RosterCache) EntryForUUID(logicalUUID string) (RosterEntry, bool) {
	if c == nil {
		return RosterEntry{}, false
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	e, ok := c.byUUID[strings.ToLower(strings.TrimSpace(logicalUUID))]
	return e, ok
}

// Len returns distinct bib count.
func (c *RosterCache) Len() int {
	if c == nil {
		return 0
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.byBib)
}
