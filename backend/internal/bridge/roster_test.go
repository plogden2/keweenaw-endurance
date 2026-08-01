package bridge

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMergeRosterPreferNamed(t *testing.T) {
	primary := []RosterEntry{{
		Bib: "6", Name: "Ada", LogicalUUID: "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee",
	}}
	fallback := []RosterEntry{
		{Bib: "6", Name: "", LogicalUUID: "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"},
		{Bib: "7", Name: "", LogicalUUID: "bbbbbbbb-bbbb-cccc-dddd-eeeeeeeeeeee"},
	}
	got := mergeRosterPreferNamed(primary, fallback)
	require.Len(t, got, 2)
	assert.Equal(t, "Ada", got[0].Name)
	assert.Equal(t, "7", got[1].Bib)
}

func TestRefreshEvent_IncludesUnassignedBibs(t *testing.T) {
	const (
		eventID = "b117f5"
		uid     = "40cb5f0d-be59-42e9-97f0-4d422b06d8fa"
	)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/races":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data":  []map[string]string{},
				"total": 0,
			})
		case r.URL.Path == "/api/events/"+eventID+"/bibs":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": []map[string]any{
					{
						"bib_number":       "6",
						"logical_uuid":     uid,
						"tag_uids":         []string{uid},
						"participant_name": nil,
					},
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	auth := &HostedAuth{BaseURL: srv.URL, BridgeToken: "tok"}
	cache := NewRosterCache()
	require.NoError(t, cache.RefreshEvent(auth, srv.Client(), eventID))

	e, ok := cache.EntryForUUID(uid)
	require.True(t, ok)
	assert.Equal(t, "6", e.Bib)
	assert.Equal(t, "", e.Name)
}
