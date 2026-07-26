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

// RosterCache maps bib → logical RFID UUID for offline manual entry.
type RosterCache struct {
	mu   sync.RWMutex
	byBib map[string]string
}

// NewRosterCache returns an empty roster cache.
func NewRosterCache() *RosterCache {
	return &RosterCache{byBib: map[string]string{}}
}

type rosterParticipant struct {
	ID         string   `json:"id"`
	BibNumber  string   `json:"bib_number"`
	RFIDTagUID string   `json:"rfid_tag_uid"`
	TagUIDs    []string `json:"tag_uids"`
}

// Refresh loads participants for a race from hosted (paginated, limit 200).
func (c *RosterCache) Refresh(auth *HostedAuth, client *http.Client, raceID string) error {
	if c == nil {
		return fmt.Errorf("roster cache is nil")
	}
	if auth == nil {
		return fmt.Errorf("auth not configured")
	}
	raceID = strings.TrimSpace(raceID)
	if raceID == "" {
		return fmt.Errorf("race_id is required")
	}
	if client == nil {
		client = &http.Client{Timeout: 20 * time.Second}
	}

	next := map[string]string{}
	page := 1
	for {
		url := fmt.Sprintf("%s/api/races/%s/participants?page=%d&limit=200", auth.BaseURL, raceID, page)
		req, err := http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			return err
		}
		if auth.BearerToken != "" {
			req.Header.Set("Authorization", "Bearer "+auth.BearerToken)
		}
		if auth.BridgeToken != "" {
			req.Header.Set("X-Bridge-Token", auth.BridgeToken)
		}

		resp, err := client.Do(req)
		if err != nil {
			return err
		}
		raw, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("roster refresh failed: %s", strings.TrimSpace(string(raw)))
		}

		var wrapped struct {
			Data  []rosterParticipant `json:"data"`
			Total int                 `json:"total"`
		}
		if err := json.Unmarshal(raw, &wrapped); err != nil {
			return err
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
			if bib != "" && uid != "" {
				next[bib] = uid
			}
		}
		if page*200 >= wrapped.Total || len(wrapped.Data) == 0 {
			break
		}
		page++
	}

	c.mu.Lock()
	c.byBib = next
	c.mu.Unlock()
	return nil
}

// LogicalUUIDForBib returns the cached logical UUID for a bib.
func (c *RosterCache) LogicalUUIDForBib(bib string) (string, bool) {
	if c == nil {
		return "", false
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	uid, ok := c.byBib[strings.TrimSpace(bib)]
	return uid, ok && uid != ""
}

// Len returns cached bib count.
func (c *RosterCache) Len() int {
	if c == nil {
		return 0
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.byBib)
}
