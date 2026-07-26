package bridgeapp

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApplyBluffetDefaults(t *testing.T) {
	cfg := Config{}
	ApplyBluffetDefaults(&cfg)
	assert.Equal(t, "https://www.keweenawendurance.com", cfg.HostedAPIURL)
	assert.Equal(t, DefaultOrganizerPIN, cfg.OrganizerPIN)
	assert.Equal(t, "laptop-finish-1", cfg.DeviceID)
	assert.Equal(t, BluffetEventIDFull, cfg.EventID)
	assert.Empty(t, cfg.RaceID, "default is All races")
	assert.Empty(t, cfg.CheckpointID)
	assert.True(t, cfg.RFIDHardware)
}

func TestApplyBluffetDefaults_ForcesEventKeepsBluffetRace(t *testing.T) {
	cfg := Config{
		EventID:      "old-event",
		DeviceID:     "other-device",
		RaceID:       "209769a1-f723-4f70-ae90-466a46338684", // 6 Hour
		CheckpointID: "stale",
	}
	ApplyBluffetDefaults(&cfg)
	assert.Equal(t, BluffetEventIDFull, cfg.EventID)
	assert.Equal(t, "laptop-finish-1", cfg.DeviceID)
	assert.Equal(t, "209769a1-f723-4f70-ae90-466a46338684", cfg.RaceID)
	assert.Equal(t, "5b7e8d76-8cc4-5e17-9147-9ed99a8df6fa", cfg.CheckpointID)
}

func TestFetchBluffetDetails_FromHosted(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/events":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": []map[string]string{{"id": "b117f5", "name": "All You Can East Bluffet"}},
			})
		case "/api/events/1441674d-a011-471a-a601-722b88b117f5", "/api/events/b117f5":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id": "b117f5",
				"races": []map[string]string{
					{"id": "6a394e", "name": "12 Hour"},
					{"id": "338684", "name": "6 Hour"},
				},
			})
		case "/api/races/6a394e/checkpoints":
			_ = json.NewEncoder(w).Encode([]map[string]string{
				{"id": "6f8b78", "name": "Start Line", "checkpoint_type": "start"},
				{"id": "cb63f5", "name": "Lap Check", "checkpoint_type": "finish"},
			})
		case "/api/races/338684/checkpoints":
			_ = json.NewEncoder(w).Encode([]map[string]string{
				{"id": "8df6fa", "name": "Lap Check", "checkpoint_type": "finish"},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	details, err := FetchBluffetDetails(srv.URL, nil)
	require.NoError(t, err)
	assert.Equal(t, BluffetEventIDFull, details.EventID)
	require.Len(t, details.Races, 2)
	assert.Equal(t, "12 Hour", details.Races[0].Name)
	assert.Equal(t, "6a394e", details.Races[0].RaceID)
	assert.Equal(t, "cb63f5", details.Races[0].FinishCheckpointID)
}

func TestAutofillConfig_FillsEmptyOnly(t *testing.T) {
	t.Setenv("BRIDGE_TOKEN", "from-env-token")
	cfg := Config{HostedAPIURL: "http://127.0.0.1:1"} // force seed fallback
	details, _ := AutofillConfig(&cfg, false)
	assert.Equal(t, "from-env-token", cfg.BridgeToken)
	assert.Equal(t, BluffetEventIDFull, cfg.EventID)
	assert.Empty(t, cfg.RaceID, "All races remains empty — do not autofill a distance")
	assert.NotEmpty(t, details.Races)
}
