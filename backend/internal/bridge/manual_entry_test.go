package bridge

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPostManualEntry_Success(t *testing.T) {
	var gotAuth string
	var gotBody map[string]string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/rfid/manual-entry", r.URL.Path)
		gotAuth = r.Header.Get("Authorization")
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	auth := &HostedAuth{BaseURL: srv.URL, BearerToken: "jwt-here"}
	err := PostManualEntry(auth, nil, ManualEntryRequest{
		RaceID:       "race-1",
		CheckpointID: "cp-1",
		BibNumber:    "42",
		Timestamp:    time.Now().UTC(),
		DeviceID:     "laptop-finish-1",
	})
	require.NoError(t, err)
	assert.Equal(t, "Bearer jwt-here", gotAuth)
	assert.Equal(t, "42", gotBody["bib_number"])
	assert.Equal(t, "race-1", gotBody["race_id"])
	assert.Equal(t, "cp-1", gotBody["checkpoint_id"])
}

func TestEnsureBearer_ExchangesPIN(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/auth/pin", r.URL.Path)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"token":"fresh-jwt"}`))
	}))
	defer srv.Close()

	auth := &HostedAuth{BaseURL: srv.URL, BridgeToken: "bridge"}
	require.NoError(t, EnsureBearer(auth, nil, "1738"))
	assert.Equal(t, "fresh-jwt", auth.BearerToken)
}

func TestRosterCache_LookupBib(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.Path, "/api/races/race-1/participants")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{
				{"id": "p1", "bib_number": "7", "first_name": "Ada", "last_name": "Lovelace", "rfid_tag_uid": "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee", "tag_uids": []string{"aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"}},
				{"id": "p2", "bib_number": "8", "rfid_tag_uid": "", "tag_uids": []string{}},
			},
			"total": 2,
		})
	}))
	defer srv.Close()

	auth := &HostedAuth{BaseURL: srv.URL, BearerToken: "jwt"}
	cache := NewRosterCache()
	require.NoError(t, cache.Refresh(auth, nil, "race-1"))
	uid, ok := cache.LogicalUUIDForBib("7")
	require.True(t, ok)
	assert.Equal(t, "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee", uid)
	entries := cache.EntriesForBib("7")
	require.Len(t, entries, 1)
	assert.Equal(t, "Ada Lovelace", entries[0].Name)
	_, ok = cache.LogicalUUIDForBib("8")
	assert.False(t, ok)
	_, ok = cache.LogicalUUIDForBib("99")
	assert.False(t, ok)
}

func TestRosterCache_AmbiguousBibAcrossRaces(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.Path, "/api/races") && r.URL.Query().Get("event_id") != "":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": []map[string]any{
					{"id": "race-a", "name": "12 Hour"},
					{"id": "race-b", "name": "6 Hour"},
				},
				"total": 2,
			})
		case strings.HasSuffix(r.URL.Path, "/checkpoints"):
			_ = json.NewEncoder(w).Encode([]map[string]any{
				{"id": "cp-finish", "checkpoint_type": "finish"},
			})
		case strings.Contains(r.URL.Path, "/participants"):
			raceID := "race-a"
			if strings.Contains(r.URL.Path, "race-b") {
				raceID = "race-b"
			}
			uid := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
			if raceID == "race-b" {
				uid = "bbbbbbbb-bbbb-cccc-dddd-eeeeeeeeeeee"
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": []map[string]any{
					{"id": "p1", "bib_number": "7", "first_name": "Pat", "last_name": "Runner", "rfid_tag_uid": uid},
				},
				"total": 1,
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	auth := &HostedAuth{BaseURL: srv.URL, BearerToken: "jwt"}
	cache := NewRosterCache()
	require.NoError(t, cache.RefreshEvent(auth, nil, "event-1"))
	entries := cache.EntriesForBib("7")
	require.Len(t, entries, 2)
	_, ok := cache.LogicalUUIDForBib("7")
	assert.False(t, ok, "ambiguous bib must not resolve to a single UUID")
	e, ok := cache.EntryForUUID("bbbbbbbb-bbbb-cccc-dddd-eeeeeeeeeeee")
	require.True(t, ok)
	assert.Equal(t, "Pat Runner", e.Name)
	assert.Equal(t, "6 Hour", e.RaceName)
}
