package bridgeapp

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/keweenaw-endurance/backend/internal/rfid"
	"github.com/keweenaw-endurance/backend/internal/services"
)

// Config drives the device bridge / reader GUI.
type Config struct {
	HostedAPIURL    string        `json:"hosted_api_url"`
	BridgeToken     string        `json:"bridge_token"`
	OrganizerPIN    string        `json:"organizer_pin"`
	DeviceID        string        `json:"device_id"`
	EventID         string        `json:"event_id"`
	RaceID          string        `json:"race_id"`
	CheckpointID    string        `json:"checkpoint_id"`
	DataDir         string        `json:"data_dir"`
	LocalAddr       string        `json:"local_addr"`
	PartitionSignal string        `json:"partition_signal"`
	ProxmarkCLI     string        `json:"proxmark3_cli"`
	ProxmarkPort    string        `json:"proxmark3_port"`
	RFIDHardware    bool          `json:"rfid_hardware"`
	BridgeMock      bool          `json:"bridge_mock"`
	WriteOnly       bool          `json:"write_only"`
	HFGain          int           `json:"hf_gain"`
	PollInterval    time.Duration `json:"-"`
	PollMS          int           `json:"poll_ms"`
}

type configFile struct {
	HostedAPIURL    string `json:"hosted_api_url"`
	BridgeToken     string `json:"bridge_token"`
	OrganizerPIN    string `json:"organizer_pin"`
	DeviceID        string `json:"device_id"`
	EventID         string `json:"event_id"`
	RaceID          string `json:"race_id"`
	CheckpointID    string `json:"checkpoint_id"`
	DataDir         string `json:"data_dir"`
	LocalAddr       string `json:"local_addr"`
	PartitionSignal string `json:"partition_signal"`
	ProxmarkCLI     string `json:"proxmark3_cli"`
	ProxmarkPort    string `json:"proxmark3_port"`
	RFIDHardware    bool   `json:"rfid_hardware"`
	BridgeMock      bool   `json:"bridge_mock"`
	WriteOnly       bool   `json:"write_only"`
	HFGain          int    `json:"hf_gain"`
	PollMS          int    `json:"poll_ms"`
}

// DefaultConfig returns race-day defaults (no secrets).
// Event/race/checkpoint default to All You Can East Bluffet 12 Hour finish.
func DefaultConfig() Config {
	seed := SeedBluffetDetails()
	// Default All races (empty RaceID) — one finish reader for the whole event.
	return Config{
		HostedAPIURL:    "https://www.keweenawendurance.com",
		OrganizerPIN:    DefaultOrganizerPIN,
		DeviceID:        services.DefaultBridgeDeviceID,
		EventID:         seed.EventID,
		DataDir:         "./bridge-data",
		LocalAddr:       "127.0.0.1:8091",
		PartitionSignal: filepath.Join(os.TempDir(), "keweenaw-bridge-partition.signal"),
		ProxmarkCLI:     "pm3",
		ProxmarkPort:    "COM3",
		RFIDHardware:    true,
		HFGain:          rfid.HFGainDefault,
		PollInterval:    500 * time.Millisecond,
		PollMS:          500,
	}
}

// PreferredGUIDataDir is the portable per-user data directory for the GUI.
func PreferredGUIDataDir() string {
	if base := os.Getenv("LOCALAPPDATA"); base != "" {
		return filepath.Join(base, "KeweenawEndurance", "bridge-data")
	}
	return "./bridge-data"
}

// ConfigPath returns the default config JSON path under dataDir.
func ConfigPath(dataDir string) string {
	if strings.TrimSpace(dataDir) == "" {
		dataDir = PreferredGUIDataDir()
	}
	return filepath.Join(dataDir, "reader-gui-config.json")
}

// LoadConfigFile reads JSON without applying env overrides.
func LoadConfigFile(path string) (Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}
	var file configFile
	if err := json.Unmarshal(raw, &file); err != nil {
		return Config{}, err
	}
	cfg := configFromFile(file)
	normalizeConfig(&cfg)
	return cfg, nil
}

// LoadConfig loads file (if present), applies defaults, then env overrides.
func LoadConfig(path string) (Config, error) {
	cfg := DefaultConfig()
	if path == "" {
		path = ConfigPath(cfg.DataDir)
	}
	if raw, err := os.ReadFile(path); err == nil {
		var file configFile
		if err := json.Unmarshal(raw, &file); err != nil {
			return Config{}, err
		}
		cfg = mergeConfig(cfg, configFromFile(file))
	} else if !os.IsNotExist(err) {
		return Config{}, err
	}
	ApplyEnv(&cfg)
	normalizeConfig(&cfg)
	return cfg, nil
}

// SaveConfig writes config JSON (creates parent dirs).
func SaveConfig(path string, cfg Config) error {
	normalizeConfig(&cfg)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	file := configFile{
		HostedAPIURL:    cfg.HostedAPIURL,
		BridgeToken:     cfg.BridgeToken,
		OrganizerPIN:    cfg.OrganizerPIN,
		DeviceID:        cfg.DeviceID,
		EventID:         cfg.EventID,
		RaceID:          cfg.RaceID,
		CheckpointID:    cfg.CheckpointID,
		DataDir:         cfg.DataDir,
		LocalAddr:       cfg.LocalAddr,
		PartitionSignal: cfg.PartitionSignal,
		ProxmarkCLI:     cfg.ProxmarkCLI,
		ProxmarkPort:    cfg.ProxmarkPort,
		RFIDHardware:    cfg.RFIDHardware,
		BridgeMock:      cfg.BridgeMock,
		WriteOnly:       cfg.WriteOnly,
		HFGain:          cfg.HFGain,
		PollMS:          cfg.PollMS,
	}
	raw, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(raw, '\n'), 0o600)
}

// ApplyEnv overlays standard bridge env vars onto cfg.
func ApplyEnv(cfg *Config) {
	if cfg == nil {
		return
	}
	if v := strings.TrimSpace(os.Getenv("HOSTED_API_URL")); v != "" {
		cfg.HostedAPIURL = strings.TrimRight(v, "/")
	}
	if v := strings.TrimSpace(os.Getenv("BRIDGE_TOKEN")); v != "" {
		cfg.BridgeToken = v
	}
	if v := strings.TrimSpace(os.Getenv("ORGANIZER_PIN")); v != "" {
		cfg.OrganizerPIN = v
	}
	if v := strings.TrimSpace(os.Getenv("DEVICE_ID")); v != "" {
		cfg.DeviceID = v
	}
	if v := strings.TrimSpace(os.Getenv("EVENT_ID")); v != "" {
		cfg.EventID = v
	}
	if v := strings.TrimSpace(os.Getenv("RACE_ID")); v != "" {
		cfg.RaceID = v
	}
	if v := strings.TrimSpace(os.Getenv("CHECKPOINT_ID")); v != "" {
		cfg.CheckpointID = v
	}
	if v := strings.TrimSpace(os.Getenv("BRIDGE_DATA_DIR")); v != "" {
		cfg.DataDir = v
	}
	if v := strings.TrimSpace(os.Getenv("BRIDGE_LOCAL_ADDR")); v != "" {
		cfg.LocalAddr = v
	}
	if v := strings.TrimSpace(os.Getenv("BRIDGE_PARTITION_SIGNAL")); v != "" {
		cfg.PartitionSignal = v
	}
	if v := strings.TrimSpace(os.Getenv("PROXMARK3_CLI")); v != "" {
		cfg.ProxmarkCLI = v
	}
	if v := strings.TrimSpace(os.Getenv("PROXMARK3_PORT")); v != "" {
		cfg.ProxmarkPort = v
	}
	if v := strings.TrimSpace(os.Getenv("RFID_HARDWARE")); v != "" {
		cfg.RFIDHardware = strings.EqualFold(v, "true")
	}
	if v := strings.TrimSpace(os.Getenv("BRIDGE_MOCK")); v != "" {
		cfg.BridgeMock = strings.EqualFold(v, "true")
	}
	if v := strings.TrimSpace(os.Getenv("BRIDGE_WRITE_ONLY")); v != "" {
		cfg.WriteOnly = strings.EqualFold(v, "true")
	}
	if v := strings.TrimSpace(os.Getenv("BRIDGE_POLL_MS")); v != "" {
		if ms, err := time.ParseDuration(v + "ms"); err == nil && ms > 0 {
			cfg.PollInterval = ms
			cfg.PollMS = int(ms / time.Millisecond)
		}
	}
	if v := strings.TrimSpace(os.Getenv("PROXMARK3_HF_GAIN")); v != "" {
		if gain, err := strconv.Atoi(v); err == nil {
			cfg.HFGain = gain
		}
	}
}

func configFromFile(file configFile) Config {
	cfg := Config{
		HostedAPIURL:    file.HostedAPIURL,
		BridgeToken:     file.BridgeToken,
		OrganizerPIN:    file.OrganizerPIN,
		DeviceID:        file.DeviceID,
		EventID:         file.EventID,
		RaceID:          file.RaceID,
		CheckpointID:    file.CheckpointID,
		DataDir:         file.DataDir,
		LocalAddr:       file.LocalAddr,
		PartitionSignal: file.PartitionSignal,
		ProxmarkCLI:     file.ProxmarkCLI,
		ProxmarkPort:    file.ProxmarkPort,
		RFIDHardware:    file.RFIDHardware,
		BridgeMock:      file.BridgeMock,
		WriteOnly:       file.WriteOnly,
		HFGain:          file.HFGain,
		PollMS:          file.PollMS,
	}
	return cfg
}

func mergeConfig(base, overlay Config) Config {
	out := base
	if overlay.HostedAPIURL != "" {
		out.HostedAPIURL = overlay.HostedAPIURL
	}
	if overlay.BridgeToken != "" {
		out.BridgeToken = overlay.BridgeToken
	}
	if overlay.OrganizerPIN != "" {
		out.OrganizerPIN = overlay.OrganizerPIN
	}
	if overlay.DeviceID != "" {
		out.DeviceID = overlay.DeviceID
	}
	if overlay.EventID != "" {
		out.EventID = overlay.EventID
	}
	if overlay.RaceID != "" {
		out.RaceID = overlay.RaceID
	}
	if overlay.CheckpointID != "" {
		out.CheckpointID = overlay.CheckpointID
	}
	if overlay.DataDir != "" {
		out.DataDir = overlay.DataDir
	}
	if overlay.LocalAddr != "" {
		out.LocalAddr = overlay.LocalAddr
	}
	if overlay.PartitionSignal != "" {
		out.PartitionSignal = overlay.PartitionSignal
	}
	if overlay.ProxmarkCLI != "" {
		out.ProxmarkCLI = overlay.ProxmarkCLI
	}
	if overlay.ProxmarkPort != "" {
		out.ProxmarkPort = overlay.ProxmarkPort
	}
	out.RFIDHardware = overlay.RFIDHardware
	out.BridgeMock = overlay.BridgeMock
	out.WriteOnly = overlay.WriteOnly
	out.HFGain = overlay.HFGain
	if overlay.PollMS > 0 {
		out.PollMS = overlay.PollMS
	}
	return out
}

func normalizeConfig(cfg *Config) {
	cfg.HostedAPIURL = strings.TrimRight(strings.TrimSpace(cfg.HostedAPIURL), "/")
	cfg.BridgeToken = strings.TrimSpace(cfg.BridgeToken)
	cfg.OrganizerPIN = strings.TrimSpace(cfg.OrganizerPIN)
	cfg.DeviceID = strings.TrimSpace(cfg.DeviceID)
	cfg.EventID = CanonicalEventID(cfg.EventID)
	cfg.RaceID = strings.TrimSpace(cfg.RaceID)
	cfg.CheckpointID = strings.TrimSpace(cfg.CheckpointID)
	cfg.DataDir = strings.TrimSpace(cfg.DataDir)
	cfg.LocalAddr = strings.TrimSpace(cfg.LocalAddr)
	cfg.PartitionSignal = strings.TrimSpace(cfg.PartitionSignal)
	cfg.ProxmarkCLI = strings.TrimSpace(cfg.ProxmarkCLI)
	cfg.ProxmarkPort = strings.TrimSpace(cfg.ProxmarkPort)

	if cfg.HostedAPIURL == "" {
		cfg.HostedAPIURL = DefaultConfig().HostedAPIURL
	}
	if cfg.DeviceID == "" {
		cfg.DeviceID = services.DefaultBridgeDeviceID
	}
	if cfg.DataDir == "" {
		cfg.DataDir = DefaultConfig().DataDir
	}
	if cfg.LocalAddr == "" {
		cfg.LocalAddr = "127.0.0.1:8091"
	}
	if cfg.PartitionSignal == "" {
		cfg.PartitionSignal = filepath.Join(os.TempDir(), "keweenaw-bridge-partition.signal")
	}
	if cfg.ProxmarkCLI == "" {
		cfg.ProxmarkCLI = "pm3"
	}
	if cfg.PollMS <= 0 {
		if cfg.PollInterval > 0 {
			cfg.PollMS = int(cfg.PollInterval / time.Millisecond)
		} else {
			cfg.PollMS = 500
		}
	}
	cfg.PollInterval = time.Duration(cfg.PollMS) * time.Millisecond
	cfg.HFGain = rfid.ClampHFGain(cfg.HFGain)
}
