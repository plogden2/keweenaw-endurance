package rfid

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	_ "embed"
)

//go:embed pm3_continuous_arm.lua
var continuousArmLuaTemplate string

const (
	keweenawArmReady   = "KEWEENAW_ARM_READY"
	keweenawTapEnd     = "KEWEENAW_TAP_END"
	luaArmStartTimeout = 20 * time.Second
	// Matches bridgeapp.ReadSuccessCooldown — same chip won't re-beep for 1s.
	immediateTapBeepCooldown = time.Second
)

// luaArmProcess is a long-lived `proxmark3 -l` script that arms wait-for-card
// and emits tap transcripts ending with KEWEENAW_TAP_END.
type luaArmProcess struct {
	cmd      *exec.Cmd
	cancel   context.CancelFunc
	taps     chan string
	errc     chan error
	ready    chan struct{}
	onTapBeep func()
	mu       sync.Mutex
	closed   bool
}

// immediateTapBeep plays the tap tone after a UUID decodes successfully.
// Debounced so a resting tag does not machine-gun the speaker while re-arming.
func (r *CLIProxmarkReader) immediateTapBeep() {
	if r == nil || !r.beepEnabled.Load() {
		return
	}
	r.mu.Lock()
	if time.Since(r.lastTapBeep) < immediateTapBeepCooldown {
		r.mu.Unlock()
		return
	}
	r.lastTapBeep = time.Now()
	r.mu.Unlock()
	PlayTapBeep()
}

func (r *CLIProxmarkReader) preferLuaArm() bool {
	if r == nil || r.injectedRunner || r.useSession {
		return false
	}
	if v := strings.TrimSpace(os.Getenv("PROXMARK3_LUA_ARM")); v != "" {
		return v == "1" || strings.EqualFold(v, "true")
	}
	// Windows interactive sessions cannot see return prompts over pipes; Lua
	// keeps one client process alive without needing the prompt.
	return true
}

func (r *CLIProxmarkReader) armScanLua(ctx context.Context) (string, error) {
	for {
		if err := ctx.Err(); err != nil {
			return "", err
		}
		if err := r.ensureLuaArm(ctx); err != nil {
			if ctx.Err() != nil {
				return "", ctx.Err()
			}
			if strings.Contains(err.Error(), "write in progress") {
				select {
				case <-ctx.Done():
					return "", ctx.Err()
				case <-time.After(50 * time.Millisecond):
					continue
				}
			}
			return "", err
		}
		r.mu.Lock()
		arm := r.luaArm
		r.mu.Unlock()
		if arm == nil {
			return "", fmt.Errorf("lua arm not started")
		}
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case err := <-arm.errc:
			// Only tear down the arm that failed — a replaced successor must keep running.
			r.stopLuaArmIf(arm)
			if err == nil {
				err = io.EOF
			}
			return "", err
		case chunk := <-arm.taps:
			uid, err := parsePollUUID(chunk)
			if err != nil {
				return "", err
			}
			if uid == "" {
				// Empty/partial dump (collision, no tag memory) — keep listening.
				continue
			}
			r.immediateTapBeep()
			return uid, nil
		}
	}
}

func (r *CLIProxmarkReader) ensureLuaArm(ctx context.Context) error {
	r.mu.Lock()
	for r.luaArmStarting {
		r.mu.Unlock()
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(20 * time.Millisecond):
		}
		r.mu.Lock()
	}
	if r.writing {
		r.mu.Unlock()
		return fmt.Errorf("lua arm blocked: write in progress")
	}
	if r.luaArm != nil && !r.luaArm.closed {
		r.mu.Unlock()
		return nil
	}
	r.luaArmStarting = true
	gain := r.hfGain
	cliPath := r.cliPath
	port := r.port
	stale := r.luaArm
	r.luaArm = nil
	r.mu.Unlock()

	if stale != nil {
		_ = stale.Close()
	}
	// Dual arms on one COM port produce coin beeps with empty parses and no site taps.
	if killAll := killAllProxmarkHook; killAll != nil {
		_ = killAll()
	}

	scriptPath, err := writeContinuousArmScript(HFThreshFromGain(gain))
	if err != nil {
		r.mu.Lock()
		r.luaArmStarting = false
		r.mu.Unlock()
		return err
	}
	arm, err := startLuaArm(ctx, cliPath, port, scriptPath, nil)
	if err != nil {
		r.mu.Lock()
		r.luaArmStarting = false
		r.mu.Unlock()
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.luaArmStarting = false
	if r.writing {
		_ = arm.Close()
		return fmt.Errorf("lua arm blocked: write in progress")
	}
	if r.luaArm != nil {
		_ = r.luaArm.Close()
	}
	r.luaArm = arm
	return nil
}

func (r *CLIProxmarkReader) stopLuaArm() {
	r.stopLuaArmIf(nil)
}

// stopLuaArmIf closes the active lua arm. When want is non-nil, only closes if
// it is still the active arm (avoids killing a successor after a restart race).
func (r *CLIProxmarkReader) stopLuaArmIf(want *luaArmProcess) {
	r.mu.Lock()
	arm := r.luaArm
	if want != nil && arm != want {
		r.mu.Unlock()
		return
	}
	r.luaArm = nil
	r.mu.Unlock()
	if arm != nil {
		_ = arm.Close()
	}
}

func writeContinuousArmScript(thresh int) (string, error) {
	if thresh < 1 {
		thresh = 3
	}
	body := strings.ReplaceAll(continuousArmLuaTemplate, "{{THRESH}}", fmt.Sprintf("%d", thresh))
	dir := filepath.Join(os.TempDir(), "keweenaw-endurance")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	path := filepath.Join(dir, "pm3_continuous_arm.lua")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		return "", err
	}
	return path, nil
}

func startLuaArm(ctx context.Context, cliPath, port, scriptPath string, onTapBeep func()) (*luaArmProcess, error) {
	runCtx, cancel := context.WithCancel(context.Background())
	exe := resolveProxmarkExecutable(cliPath)
	args := []string{}
	if port != "" {
		args = append(args, "-p", port)
	}
	args = append(args, "-f", "--incognito", "-l", scriptPath)
	cmd := exec.CommandContext(runCtx, exe, args...)
	cmd.Env = proxmarkCommandEnv(cliPath)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		cancel()
		return nil, err
	}
	cmd.Stderr = cmd.Stdout
	if err := cmd.Start(); err != nil {
		cancel()
		return nil, err
	}
	arm := &luaArmProcess{
		cmd:       cmd,
		cancel:    cancel,
		taps:      make(chan string, 4),
		errc:      make(chan error, 1),
		ready:     make(chan struct{}),
		onTapBeep: onTapBeep,
	}
	go arm.readLoop(stdout)
	go func() {
		err := cmd.Wait()
		arm.mu.Lock()
		arm.closed = true
		arm.mu.Unlock()
		if err != nil {
			select {
			case arm.errc <- err:
			default:
			}
		} else {
			select {
			case arm.errc <- io.EOF:
			default:
			}
		}
	}()

	startCtx, startCancel := context.WithTimeout(ctx, luaArmStartTimeout)
	defer startCancel()
	select {
	case <-startCtx.Done():
		_ = arm.Close()
		return nil, fmt.Errorf("lua arm startup: %w", startCtx.Err())
	case err := <-arm.errc:
		_ = arm.Close()
		return nil, fmt.Errorf("lua arm startup: %w", err)
	case <-arm.ready:
		return arm, nil
	}
}

func (a *luaArmProcess) readLoop(r io.Reader) {
	sc := bufio.NewScanner(r)
	// Proxmark pages can be large; default 64K is enough but be safe.
	buf := make([]byte, 0, 64*1024)
	sc.Buffer(buf, 1024*1024)
	var chunk strings.Builder
	readyClosed := false
	for sc.Scan() {
		line := sc.Text()
		chunk.WriteString(line)
		chunk.WriteByte('\n')
		if strings.Contains(line, keweenawArmReady) && !readyClosed {
			readyClosed = true
			close(a.ready)
			chunk.Reset()
			continue
		}
		if strings.Contains(line, keweenawTapEnd) {
			// Beep is owned by armScanLua after parsePollUUID succeeds — raw dump
			// lines can match under COM contention without a scorable UUID.
			select {
			case a.taps <- chunk.String():
			default:
				// Drop if consumer is slow; next tap will follow.
			}
			chunk.Reset()
		}
	}
	if !readyClosed {
		close(a.ready)
	}
	if err := sc.Err(); err != nil {
		select {
		case a.errc <- err:
		default:
		}
	}
}

func (a *luaArmProcess) Close() error {
	a.mu.Lock()
	if a.closed {
		a.mu.Unlock()
		return nil
	}
	a.closed = true
	a.mu.Unlock()
	if a.cancel != nil {
		a.cancel()
	}
	if a.cmd != nil && a.cmd.Process != nil {
		_ = a.cmd.Process.Kill()
		_, _ = a.cmd.Process.Wait()
	}
	return nil
}
