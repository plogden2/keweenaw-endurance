package bridgeapp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/keweenaw-endurance/backend/internal/bridge"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApp_ManualEntryOfflineQueuesPending(t *testing.T) {
	dir := t.TempDir()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/auth/pin":
			_ = json.NewEncoder(w).Encode(map[string]string{"token": "jwt"})
		case r.URL.Path == "/api/races/race-1/participants":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": []map[string]any{
					{"bib_number": "12", "rfid_tag_uid": "11111111-2222-3333-4444-555555555555"},
				},
				"total": 1,
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	cfg := Config{
		HostedAPIURL: srv.URL,
		OrganizerPIN: "1738",
		DeviceID:     "laptop-finish-1",
		EventID:      "11111111-1111-1111-1111-111111111111",
		RaceID:       "race-1",
		CheckpointID: "cp-1",
		DataDir:      dir,
		LocalAddr:    "127.0.0.1:0",
		BridgeMock:   true,
		PollMS:       500,
	}
	normalizeConfig(&cfg)

	app, err := New(cfg)
	require.NoError(t, err)
	require.NoError(t, app.RefreshRoster())

	// Force offline path.
	app.mu.Lock()
	app.online = false
	app.mu.Unlock()

	require.NoError(t, app.ManualEntry("12", time.Date(2026, 7, 25, 12, 0, 0, 0, time.UTC)))
	assert.Equal(t, 1, app.store.PendingCount())
	assert.True(t, filepath.IsAbs(app.store.PendingPath()) || app.store.PendingPath() != "")
}

func TestApp_StartStopMock(t *testing.T) {
	dir := t.TempDir()
	// Unreachable hosted — Start still runs local loops; WS will error in background.
	cfg := Config{
		HostedAPIURL: "http://127.0.0.1:1",
		OrganizerPIN: "1738",
		DeviceID:     "laptop-finish-1",
		EventID:      "11111111-1111-1111-1111-111111111111",
		DataDir:      dir,
		LocalAddr:    "127.0.0.1:18091",
		BridgeMock:   true,
		PollMS:       200,
	}
	// New calls ResolveHostedAuth which needs PIN exchange — stub with bridge token instead.
	cfg.OrganizerPIN = ""
	cfg.BridgeToken = "test-token"
	normalizeConfig(&cfg)

	app, err := New(cfg)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	require.NoError(t, app.Start(ctx))
	assert.True(t, app.Running())
	st := app.StatusSnapshot()
	assert.Equal(t, "laptop-finish-1", st.DeviceID)
	app.Stop()
	assert.False(t, app.Running())
	_ = bridge.ModeOffline
}
