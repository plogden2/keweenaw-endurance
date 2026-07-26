package bridgeapp

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfig_SaveLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "reader-gui-config.json")

	cfg := Config{
		HostedAPIURL:   "https://www.keweenawendurance.com",
		BridgeToken:    "tok",
		OrganizerPIN:   "1738",
		DeviceID:       "laptop-finish-1",
		EventID:        "evt-1",
		RaceID:         "race-1",
		CheckpointID:   "cp-1",
		DataDir:        dir,
		LocalAddr:      "127.0.0.1:8091",
		ProxmarkCLI:    `C:\repo\scripts\pm3.cmd`,
		ProxmarkPort:   "COM3",
		RFIDHardware:   true,
		BridgeMock:     false,
		PollInterval:   500 * time.Millisecond,
	}
	require.NoError(t, SaveConfig(path, cfg))

	loaded, err := LoadConfigFile(path)
	require.NoError(t, err)
	assert.Equal(t, cfg.HostedAPIURL, loaded.HostedAPIURL)
	assert.Equal(t, cfg.BridgeToken, loaded.BridgeToken)
	assert.Equal(t, cfg.OrganizerPIN, loaded.OrganizerPIN)
	assert.Equal(t, cfg.DeviceID, loaded.DeviceID)
	assert.Equal(t, cfg.EventID, loaded.EventID)
	assert.Equal(t, cfg.RaceID, loaded.RaceID)
	assert.Equal(t, cfg.CheckpointID, loaded.CheckpointID)
	assert.Equal(t, cfg.ProxmarkPort, loaded.ProxmarkPort)
	assert.True(t, loaded.RFIDHardware)
	assert.Equal(t, 500*time.Millisecond, loaded.PollInterval)
}

func TestConfig_ApplyEnvOverridesFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "reader-gui-config.json")
	require.NoError(t, SaveConfig(path, Config{
		HostedAPIURL: "http://file.example",
		DeviceID:     "from-file",
		EventID:      "evt-file",
		DataDir:      dir,
	}))

	t.Setenv("HOSTED_API_URL", "https://env.example")
	t.Setenv("DEVICE_ID", "from-env")
	t.Setenv("EVENT_ID", "evt-env")
	t.Setenv("RFID_HARDWARE", "true")
	t.Setenv("PROXMARK3_PORT", "COM7")

	cfg, err := LoadConfig(path)
	require.NoError(t, err)
	assert.Equal(t, "https://env.example", cfg.HostedAPIURL)
	assert.Equal(t, "from-env", cfg.DeviceID)
	assert.Equal(t, "evt-env", cfg.EventID)
	assert.True(t, cfg.RFIDHardware)
	assert.Equal(t, "COM7", cfg.ProxmarkPort)
}

func TestConfig_LoadMissingFileUsesDefaults(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing.json")
	cfg, err := LoadConfig(path)
	require.NoError(t, err)
	assert.Equal(t, "https://www.keweenawendurance.com", cfg.HostedAPIURL)
	assert.Equal(t, "laptop-finish-1", cfg.DeviceID)
	assert.Equal(t, "127.0.0.1:8091", cfg.LocalAddr)
	assert.Equal(t, 500*time.Millisecond, cfg.PollInterval)
	assert.NotEmpty(t, cfg.DataDir)
}
