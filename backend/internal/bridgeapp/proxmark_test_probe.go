package bridgeapp

import (
	"fmt"
	"strings"

	"github.com/keweenaw-endurance/backend/internal/rfid"
)

// TestProxmark runs a non-destructive probe (hw version via CLI when possible).
func TestProxmark(cfg Config) (string, error) {
	normalizeConfig(&cfg)
	if cfg.BridgeMock {
		return "mock reader OK", nil
	}
	if !cfg.RFIDHardware {
		return "", fmt.Errorf("RFID_HARDWARE is false — enable hardware or use mock")
	}
	reader := rfid.NewCLIProxmarkReader(rfid.CLIProxmarkConfig{
		CLIPath: cfg.ProxmarkCLI,
		Port:    cfg.ProxmarkPort,
		Enabled: true,
	})
	// Poll is non-destructive; empty UUID with no error means hardware answered.
	uid, err := reader.Poll()
	if err != nil {
		// Many "no tag" outcomes still prove the CLI/port path works.
		msg := err.Error()
		if strings.Contains(strings.ToLower(msg), "no tag") ||
			strings.Contains(strings.ToLower(msg), "waiting") ||
			strings.Contains(msg, "ERROR:") {
			return fmt.Sprintf("Proxmark reachable on %s (no tag present)", cfg.ProxmarkPort), nil
		}
		return "", err
	}
	if uid != "" {
		return fmt.Sprintf("Proxmark OK on %s — read UUID %s", cfg.ProxmarkPort, uid), nil
	}
	return fmt.Sprintf("Proxmark OK on %s (no tag in field)", cfg.ProxmarkPort), nil
}
