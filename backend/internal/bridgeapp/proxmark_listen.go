package bridgeapp

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/keweenaw-endurance/backend/internal/rfid"
)

// TapCallback is invoked for each accepted Proxmark test tap (not recorded).
type TapCallback func(logicalUUID string)

// ListenProxmark runs a continuous arm session for hardware diagnostics.
// Each decoded UUID beeps and invokes onTap. Nothing is recorded or synced.
// Returns nil when ctx is cancelled; setup/config errors otherwise.
func ListenProxmark(ctx context.Context, cfg Config, onTap TapCallback) error {
	normalizeConfig(&cfg)
	if cfg.BridgeMock {
		return errors.New("disable mock reader to test Proxmark hardware")
	}
	if !cfg.RFIDHardware {
		return fmt.Errorf("RFID_HARDWARE is false — enable hardware or use mock")
	}

	rfid.PrewarmTapBeep()
	reader := rfid.NewCLIProxmarkReader(rfid.CLIProxmarkConfig{
		CLIPath: cfg.ProxmarkCLI,
		Port:    cfg.ProxmarkPort,
		Enabled: true,
		HFGain:  cfg.HFGain,
	})
	// Beep once in the listen loop after a full UUID decode (avoids Lua double-beep).
	reader.SetBeepEnabled(false)
	reader.SetHFGain(cfg.HFGain)
	pm3 := rfid.NewProxmark3(reader)

	return listenProxmarkLoop(ctx, pm3.ArmScan, rfid.PlayTapBeep, onTap, ReadSuccessCooldown)
}

func listenProxmarkLoop(
	ctx context.Context,
	armScan func(context.Context) (string, error),
	beep func(),
	onTap TapCallback,
	debounce time.Duration,
) error {
	if armScan == nil {
		return errors.New("arm scan not configured")
	}
	var last string
	var lastAt time.Time
	for {
		if err := ctx.Err(); err != nil {
			return nil
		}
		uid, err := armScan(ctx)
		if ctx.Err() != nil {
			return nil
		}
		if err != nil {
			select {
			case <-ctx.Done():
				return nil
			case <-time.After(500 * time.Millisecond):
			}
			continue
		}
		uid = strings.ToLower(strings.TrimSpace(uid))
		if uid == "" {
			continue
		}
		if debounce > 0 && uid == last && time.Since(lastAt) < debounce {
			continue
		}
		last = uid
		lastAt = time.Now()
		if beep != nil {
			beep()
		}
		if onTap != nil {
			onTap(uid)
		}
	}
}
