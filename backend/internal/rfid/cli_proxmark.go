package rfid

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// NTAG / MIFARE Ultralight user memory: 4-byte pages. Logical UUID is 16 bytes
// starting at page 4 (pages 4–7). Commands use `-b` / `--block` (not `--blk`).
const (
	proxmarkUserMemoryStartPage = 4
	proxmarkPageSize            = 4
	proxmarkLogicalUUIDPages    = 4 // 16 bytes
	proxmarkCLIExecTimeout      = 15 * time.Second
	// Current RRG pm3 returns one 4-byte page per `hf mfu rdbl`; chain four reads.
	proxmarkReadLogicalUUIDCmd = "hf mfu rdbl -b 4; hf mfu rdbl -b 5; hf mfu rdbl -b 6; hf mfu rdbl -b 7"
	// Continuous arm wait: block until a tag enters the field; family-specific
	// read follows after SAK classification.
	proxmarkArmWaitCmd = "hf 14a reader -w --skip"
	// Legacy chained arm (Ultralight-only); kept for reference/tests.
	proxmarkArmScanCmd = proxmarkArmWaitCmd + "; " + proxmarkReadLogicalUUIDCmd
)

// postArmCancelSettle waits after killing continuous arm before detect/write.
// Overridable in tests; skipped entirely for injected runners.
var postArmCancelSettle = 800 * time.Millisecond

// killAllProxmarkHook is set from bridgeapp on Windows to terminate every
// proxmark3.exe (including the continuous-arm child) before a hardware write.
// Nil in unit tests / non-Windows.
var killAllProxmarkHook func() error

// SetKillAllProxmarkHook registers the OS-specific "kill every proxmark3" helper.
func SetKillAllProxmarkHook(fn func() error) {
	killAllProxmarkHook = fn
}

// CLICommandRunner executes the Proxmark3 CLI with the given pm3 subcommand string.
// Tests inject a fake runner to avoid requiring real hardware. When set, the
// reader uses one-shot process mode instead of a persistent session.
type CLICommandRunner func(command string) (stdout string, err error)

// CLIProxmarkConfig configures the pm3 CLI bridge.
type CLIProxmarkConfig struct {
	CLIPath        string
	Port           string
	Enabled        bool
	HFGain         int
	Runner         CLICommandRunner
	SessionFactory PM3SessionFactory
	Beeper         Beeper
}

// CLIProxmarkReader reads and writes logical UUIDs via the Proxmark3 CLI.
// A mutex serializes all CLI invocations so Poll and WriteTag cannot race on
// the serial port. Production mode keeps one interactive session open.
type CLIProxmarkReader struct {
	cliPath        string
	port           string
	enabled        bool
	hfGain         int
	threshApplied  bool
	runner         CLICommandRunner // one-shot (tests); nil ⇒ persistent session
	sessionFactory PM3SessionFactory
	beeper         Beeper
	beepEnabled    atomic.Bool

	mu         sync.Mutex
	session    PM3Session
	nextRetry  time.Time
	backoff    time.Duration
	useSession bool
	// injectedRunner is true when tests supply CLIProxmarkConfig.Runner.
	injectedRunner bool
	writing        bool
	armCancel      context.CancelFunc
	armDone        chan struct{}
	luaArm         *luaArmProcess
	luaArmStarting bool // serialize ensureLuaArm across cancel/write restarts
	lastTapBeep    time.Time
}

func NewCLIProxmarkReader(cfg CLIProxmarkConfig) *CLIProxmarkReader {
	cliPath := cfg.CLIPath
	if cliPath == "" {
		cliPath = "pm3"
	}
	beeper := cfg.Beeper
	if beeper == nil {
		beeper = defaultBeeper()
	}
	r := &CLIProxmarkReader{
		cliPath: cliPath,
		port:    cfg.Port,
		enabled: cfg.Enabled,
		hfGain:  ClampHFGain(cfg.HFGain),
		beeper:  beeper,
		backoff: proxmarkReconnectMinBackoff,
	}
	r.beepEnabled.Store(true)
	if cfg.Runner != nil {
		r.runner = cfg.Runner
		r.injectedRunner = true
		r.useSession = false
	} else if cfg.SessionFactory != nil {
		r.sessionFactory = cfg.SessionFactory
		r.useSession = true
	} else if preferOneShotCLI() {
		// Windows Proxmark clients using linenoise often never emit "pm3 -->"
		// when stdin/stdout are pipes, so persistent sessions hang on startup.
		// Continuous finish-line reads use ArmScan (hf 14a reader -w) instead.
		r.runner = defaultCLICommandRunner(cliPath, cfg.Port)
		r.useSession = false
	} else {
		r.sessionFactory = defaultSessionFactory(cliPath, cfg.Port)
		r.useSession = true
	}
	return r
}

func preferOneShotCLI() bool {
	if v := strings.TrimSpace(os.Getenv("PROXMARK3_SESSION")); v == "1" || strings.EqualFold(v, "true") {
		return false
	}
	if v := strings.TrimSpace(os.Getenv("PROXMARK3_ONESHOT")); v == "0" || strings.EqualFold(v, "false") {
		return false
	}
	return runtime.GOOS == "windows"
}

func defaultCLICommandRunner(cliPath, port string) CLICommandRunner {
	return func(command string) (string, error) {
		ctx, cancel := context.WithTimeout(context.Background(), proxmarkCLIExecTimeout)
		defer cancel()
		return execProxmarkCLI(ctx, cliPath, port, command)
	}
}

// execProxmarkCLI runs proxmark3.exe -c <command>. ctx cancel kills the process
// (used by ArmScan wait-for-card so writes can reclaim COM).
func execProxmarkCLI(ctx context.Context, cliPath, port, command string) (string, error) {
	args := []string{}
	if port != "" {
		args = append(args, "-p", port)
	}
	args = append(args, "-f", "--incognito", "-c", command)

	// Run proxmark3.exe directly — not pm3.cmd. CommandContext kill on
	// timeout only signals the top-level process; killing cmd.exe leaves
	// orphaned proxmark3.exe holding COM3 and blocks all later polls.
	exe := resolveProxmarkExecutable(cliPath)
	cmd := exec.CommandContext(ctx, exe, args...)
	cmd.Env = proxmarkCommandEnv(cliPath)
	out, err := cmd.CombinedOutput()
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return string(out), fmt.Errorf("proxmark3 cli %q: timed out after %s", command, proxmarkCLIExecTimeout)
	}
	if ctx.Err() != nil {
		return string(out), ctx.Err()
	}
	if err != nil {
		return string(out), fmt.Errorf("proxmark3 cli %q: %w: %s", command, err, strings.TrimSpace(string(out)))
	}
	return string(out), nil
}

// resolveProxmarkExecutable maps wrapper scripts (pm3.cmd) to proxmark3.exe so
// interactive stdin sessions work. Batch files do not forward Go pipes reliably.
func resolveProxmarkExecutable(cliPath string) string {
	if v := strings.TrimSpace(os.Getenv("PROXMARK3_EXE")); v != "" {
		if st, err := os.Stat(v); err == nil && !st.IsDir() {
			return v
		}
	}
	lower := strings.ToLower(cliPath)
	if strings.HasSuffix(lower, ".exe") || (!strings.HasSuffix(lower, ".cmd") && !strings.HasSuffix(lower, ".bat")) {
		return cliPath
	}
	if exe := parsePM3ExeFromWrapper(cliPath); exe != "" {
		return exe
	}
	if la := os.Getenv("LOCALAPPDATA"); la != "" {
		candidate := filepath.Join(la, "KeweenawReader", "proxmark", "proxmark3.exe")
		if st, err := os.Stat(candidate); err == nil && !st.IsDir() {
			return candidate
		}
	}
	return cliPath
}

func parsePM3ExeFromWrapper(cmdPath string) string {
	data, err := os.ReadFile(cmdPath)
	if err != nil {
		return ""
	}
	vars := map[string]string{}
	for _, line := range strings.Split(string(data), "\n") {
		trim := strings.TrimSpace(line)
		upper := strings.ToUpper(trim)
		if !strings.HasPrefix(upper, "SET ") {
			continue
		}
		assign := strings.TrimSpace(trim[4:])
		assign = strings.Trim(assign, `"'`)
		key, val, ok := strings.Cut(assign, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		val = expandCmdVars(strings.TrimSpace(val), vars)
		vars[strings.ToUpper(key)] = val
	}
	if exe := vars["PM3_EXE"]; exe != "" {
		if st, err := os.Stat(exe); err == nil && !st.IsDir() {
			return exe
		}
	}
	return ""
}

func expandCmdVars(s string, vars map[string]string) string {
	out := s
	for k, v := range vars {
		out = strings.ReplaceAll(out, "%"+k+"%", v)
		out = strings.ReplaceAll(out, "%"+strings.ToLower(k)+"%", v)
	}
	return out
}

// proxmarkRuntimeBin returns a directory to prepend to PATH so a Proxmark
// Windows client can resolve MinGW/Qt DLLs when spawned from Go.
// Prefers PROXMARK3_MINGW_BIN, then ProxSpace mingw64\bin, then the CLI's
// own directory when a slim side-by-side runtime is installed.
func proxmarkRuntimeBin(cliPath string) string {
	if runtime.GOOS != "windows" {
		return ""
	}
	if mingw := os.Getenv("PROXMARK3_MINGW_BIN"); mingw != "" {
		return mingw
	}
	if strings.Contains(strings.ToLower(cliPath), "proxspace") {
		candidate := filepath.Clean(filepath.Join(filepath.Dir(cliPath), "..", "..", "..", "msys2", "mingw64", "bin"))
		if st, err := os.Stat(candidate); err == nil && st.IsDir() {
			return candidate
		}
	}
	cliDir := filepath.Dir(cliPath)
	if proxmarkSideBySideRuntime(cliDir) {
		return cliDir
	}
	return ""
}

// proxmarkMingwBin is kept for callers/tests; same as proxmarkRuntimeBin.
func proxmarkMingwBin(cliPath string) string {
	return proxmarkRuntimeBin(cliPath)
}

func proxmarkSideBySideRuntime(cliDir string) bool {
	if st, err := os.Stat(filepath.Join(cliDir, "platforms")); err == nil && st.IsDir() {
		return true
	}
	for _, name := range []string{"libgcc_s_seh-1.dll", "Qt5Core.dll", "libwinpthread-1.dll"} {
		if _, err := os.Stat(filepath.Join(cliDir, name)); err == nil {
			return true
		}
	}
	return false
}

func proxmarkCommandEnv(cliPath string) []string {
	env := os.Environ()
	if bin := proxmarkRuntimeBin(cliPath); bin != "" {
		env = withPrependedPath(env, bin)
	}
	cliDir := filepath.Dir(cliPath)
	if st, err := os.Stat(filepath.Join(cliDir, "platforms")); err == nil && st.IsDir() {
		env = withEnvVar(env, "QT_PLUGIN_PATH", cliDir)
	}
	return env
}

func withPrependedPath(environ []string, dir string) []string {
	newPath := dir + string(os.PathListSeparator) + os.Getenv("PATH")
	out := make([]string, 0, len(environ)+1)
	replaced := false
	for _, e := range environ {
		if strings.HasPrefix(strings.ToUpper(e), "PATH=") {
			out = append(out, "PATH="+newPath)
			replaced = true
			continue
		}
		out = append(out, e)
	}
	if !replaced {
		out = append(out, "PATH="+newPath)
	}
	return out
}

func withEnvVar(environ []string, key, value string) []string {
	prefix := strings.ToUpper(key) + "="
	out := make([]string, 0, len(environ)+1)
	replaced := false
	for _, e := range environ {
		if strings.HasPrefix(strings.ToUpper(e), prefix) {
			out = append(out, key+"="+value)
			replaced = true
			continue
		}
		out = append(out, e)
	}
	if !replaced {
		out = append(out, key+"="+value)
	}
	return out
}

func (r *CLIProxmarkReader) IsAvailable() bool {
	return r != nil && r.enabled
}

func (r *CLIProxmarkReader) WriteLogicalUUID(logicalUUID string) error {
	if !r.IsAvailable() {
		return ErrHardwareUnavailable
	}
	raw, err := EncodeLogicalUUID(logicalUUID)
	if err != nil {
		return err
	}
	// Mark writing BEFORE cancelling the arm so runContinuousArm cannot reopen
	// Lua/COM between cancel and the write (that race causes BCC0 aborts).
	r.mu.Lock()
	r.writing = true
	r.mu.Unlock()
	defer func() {
		r.mu.Lock()
		r.writing = false
		r.mu.Unlock()
	}()

	r.cancelArmAndWait()
	// Belt-and-suspenders: cancelArmAndWait should Kill the Lua arm, but Windows
	// sometimes leaves a proxmark3.exe holding COM after a write-only toggle /
	// failed Close. writing=true blocks ArmScan from respawning until we finish.
	if !r.injectedRunner {
		if killAll := killAllProxmarkHook; killAll != nil {
			_ = killAll()
		}
	}
	// Let USB-CDC / RF field settle after killing the continuous arm process.
	// A tag still on the antenna often spuriously reports "Multiple tags /
	// Collision after Bit 1" on the first anticollision after arm cancel.
	if !r.injectedRunner && postArmCancelSettle > 0 {
		time.Sleep(postArmCancelSettle)
	}

	const writeAttempts = 3
	var (
		stdout   string
		writeErr error
		lastMsg  string
		fam      TagFamily
	)
	for attempt := 0; attempt < writeAttempts; attempt++ {
		if attempt > 0 {
			time.Sleep(400 * time.Millisecond)
		}

		r.mu.Lock()
		ctx, cancel := context.WithTimeout(context.Background(), proxmarkSessionWriteTimeout)
		detectOut, detectErr := r.runLocked(ctx, "hf 14a reader")
		cancel()
		var classErr error
		fam, classErr = ClassifyISO14443A(detectOut)
		if classErr != nil {
			r.mu.Unlock()
			return classErr
		}
		switch fam {
		case TagFamilyNone:
			msg := classifyProxmarkWriteError(detectOut, detectErr)
			if msg == "" {
				msg = noTagSelectedMessage()
			}
			r.mu.Unlock()
			lastMsg = msg
			// Transient anticollision glitch after continuous arm — retry.
			if strings.Contains(msg, "Multiple tags") || strings.Contains(msg, "collision") {
				continue
			}
			return errors.New(msg)
		case TagFamilyUnsupported:
			r.mu.Unlock()
			return errors.New(unsupportedTagTypeMessageFromStdout(detectOut))
		}
		cmd := writeCmdForFamily(fam, raw)
		ctx, cancel = context.WithTimeout(context.Background(), proxmarkSessionWriteTimeout)
		stdout, writeErr = r.runLocked(ctx, cmd)
		cancel()
		if writeErr == nil {
			r.mu.Unlock()
			return nil
		}
		// Proxmark CLI often exits -10 (0xfffffff6) even when the device ran the
		// script; confirm by reading user memory before failing the operator.
		if pm3DeviceResponded(stdout) {
			if verifyErr := r.verifyLogicalUUIDLocked(logicalUUID, fam); verifyErr == nil {
				r.mu.Unlock()
				return nil
			}
		}
		r.mu.Unlock()
		if msg := classifyProxmarkWriteError(stdout, writeErr); msg != "" {
			lastMsg = msg
			// Same transient collision — retry detect+write instead of aborting.
			if strings.Contains(msg, "Multiple tags") || strings.Contains(msg, "collision") {
				continue
			}
		}
	}
	if lastMsg != "" {
		return errors.New(lastMsg)
	}
	if msg := classifyProxmarkWriteError(stdout, writeErr); msg != "" {
		return errors.New(msg)
	}
	if fam == TagFamilyClassic1K {
		return fmt.Errorf("write classic block %d: %w", classicLogicalUUIDBlock, writeErr)
	}
	return fmt.Errorf("write pages %d-%d: %w",
		proxmarkUserMemoryStartPage,
		proxmarkUserMemoryStartPage+proxmarkLogicalUUIDPages-1,
		writeErr)
}

func writeCmdForFamily(fam TagFamily, raw []byte) string {
	if fam == TagFamilyClassic1K {
		return classicWriteBlock1Cmd(fmt.Sprintf("%x", raw))
	}
	parts := make([]string, 0, proxmarkLogicalUUIDPages)
	for i := 0; i < proxmarkLogicalUUIDPages; i++ {
		page := proxmarkUserMemoryStartPage + i
		off := i * proxmarkPageSize
		hexData := fmt.Sprintf("%x", raw[off:off+proxmarkPageSize])
		parts = append(parts, fmt.Sprintf("hf mfu wrbl -b %d -d %s", page, hexData))
	}
	return strings.Join(parts, "; ")
}

func readCmdForFamily(fam TagFamily) string {
	if fam == TagFamilyClassic1K {
		return classicReadBlock1Cmd()
	}
	return proxmarkReadLogicalUUIDCmd
}

// classifyProxmarkWriteError turns Proxmark stdout into a short operator-facing message.
func classifyProxmarkWriteError(stdout string, writeErr error) string {
	lower := strings.ToLower(stdout)
	switch {
	case strings.Contains(lower, "multiple tags"):
		return "Multiple tags on antenna — leave only one chip and retry"
	case strings.Contains(lower, "can't select card"), strings.Contains(lower, "no known/supported"):
		return "No tag selected — hold one chip steady on the antenna and retry"
	case strings.Contains(lower, "bcc0 incorrect"), strings.Contains(lower, "bcc1 incorrect"):
		return "Tag collision/BCC error — use one chip only, hold steady, retry"
	case strings.Contains(lower, "hf field is off") && writeErr != nil:
		return "Proxmark HF field off during write — retry write"
	case strings.Contains(lower, "invalid serial port"), strings.Contains(lower, "error: serial"):
		return "COM port busy — Kill orphans, then retry"
	default:
		return ""
	}
}

// verifyLogicalUUIDLocked reads family-specific user memory and checks it
// decodes to logicalUUID. Caller must hold r.mu.
func (r *CLIProxmarkReader) verifyLogicalUUIDLocked(logicalUUID string, fam TagFamily) error {
	ctx, cancel := context.WithTimeout(context.Background(), proxmarkSessionPollTimeout)
	defer cancel()
	stdout, err := r.runLocked(ctx, readCmdForFamily(fam))
	if err != nil && !pm3DeviceResponded(stdout) {
		return err
	}
	raw, parseErr := parseUUIDBytesForFamily(stdout, fam)
	if parseErr != nil {
		return parseErr
	}
	if len(raw) == 0 {
		return fmt.Errorf("readback empty")
	}
	got, err := DecodeLogicalUUID(raw)
	if err != nil {
		return err
	}
	if !strings.EqualFold(got, logicalUUID) {
		return fmt.Errorf("readback mismatch: got %s want %s", got, logicalUUID)
	}
	return nil
}

func (r *CLIProxmarkReader) Poll() (string, error) {
	if !r.IsAvailable() {
		return "", ErrHardwareUnavailable
	}
	// Skip this tick if a write holds the port — writes must not wait behind a
	// poll (Playwright write-tag timeout is otherwise too tight).
	if !r.mu.TryLock() {
		return "", nil
	}
	defer r.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), proxmarkSessionPollTimeout)
	defer cancel()
	detectOut, err := r.runLocked(ctx, "hf 14a reader")
	if err != nil && !pm3DeviceResponded(detectOut) {
		return "", nil
	}
	fam, _ := ClassifyISO14443A(detectOut)
	switch fam {
	case TagFamilyNone, TagFamilyUnsupported:
		// Soft-fail empty tick so the poll loop keeps running.
		return "", nil
	}
	stdout, err := r.runLocked(ctx, readCmdForFamily(fam))
	if err != nil {
		_ = stdout
		return "", nil
	}
	return parsePollUUID(stdout)
}

// ArmScan blocks until a tag enters the HF field (or ctx is cancelled), then
// reads logical UUID pages. Used by the finish-line bridge instead of
// spawn-per-poll one-shot ticks.
func (r *CLIProxmarkReader) ArmScan(ctx context.Context) (string, error) {
	if !r.IsAvailable() {
		return "", ErrHardwareUnavailable
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}

	armCtx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})

	r.mu.Lock()
	for r.writing {
		r.mu.Unlock()
		select {
		case <-armCtx.Done():
			cancel()
			return "", armCtx.Err()
		case <-time.After(20 * time.Millisecond):
		}
		r.mu.Lock()
	}
	if r.armCancel != nil {
		r.armCancel()
	}
	r.armCancel = cancel
	r.armDone = done
	gain := r.hfGain
	useSession := r.useSession
	injected := r.injectedRunner
	r.mu.Unlock()

	defer func() {
		cancel()
		close(done)
		r.mu.Lock()
		if r.armDone == done {
			r.armDone = nil
			r.armCancel = nil
		}
		r.mu.Unlock()
	}()

	if !useSession && !injected && r.preferLuaArm() {
		return r.armScanLua(armCtx)
	}

	waitCmd := hfThreshCommand(gain) + "; " + proxmarkArmWaitCmd
	var detectOut string
	var err error
	switch {
	case useSession:
		// Keep reader mu free while -w blocks so Write can cancelArmAndWait.
		r.mu.Lock()
		ensErr := r.ensureSessionLocked(armCtx)
		var sess PM3Session
		if ensErr == nil {
			sess = r.session
		}
		r.mu.Unlock()
		if ensErr != nil {
			err = ensErr
			break
		}
		detectOut, err = sess.Run(armCtx, waitCmd)
		if err != nil && !pm3DeviceResponded(detectOut) {
			r.mu.Lock()
			_ = r.closeSessionLocked()
			r.scheduleRetryLocked()
			r.mu.Unlock()
		}
	case injected:
		detectOut, err = r.runner(waitCmd)
	default:
		detectOut, err = execProxmarkCLI(armCtx, r.cliPath, r.port, waitCmd)
	}
	if armCtx.Err() != nil {
		return "", armCtx.Err()
	}
	if err != nil && !pm3DeviceResponded(detectOut) {
		return "", err
	}
	fam, _ := ClassifyISO14443A(detectOut)
	switch fam {
	case TagFamilyNone:
		return "", nil
	case TagFamilyUnsupported:
		return "", errors.New(unsupportedTagTypeMessageFromStdout(detectOut))
	}

	readCmd := readCmdForFamily(fam)
	var readOut string
	switch {
	case useSession:
		r.mu.Lock()
		sess := r.session
		r.mu.Unlock()
		if sess == nil {
			return "", errors.New("proxmark session closed after arm wait")
		}
		readOut, err = sess.Run(armCtx, readCmd)
	case injected:
		readOut, err = r.runner(readCmd)
	default:
		readOut, err = execProxmarkCLI(armCtx, r.cliPath, r.port, readCmd)
	}
	if armCtx.Err() != nil {
		return "", armCtx.Err()
	}
	if err != nil && !pm3DeviceResponded(readOut) {
		return "", err
	}
	return parsePollUUID(readOut)
}

func parsePollUUID(stdout string) (string, error) {
	raw, parseErr := parseUUIDBytesForFamily(stdout, familyFromStdout(stdout))
	if parseErr != nil {
		// Partial/one-page dumps from older clients: treat as empty tick, not hard error.
		return "", nil
	}
	if len(raw) == 0 || isZeroBlock(raw) {
		return "", nil
	}
	uid, err := DecodeLogicalUUID(raw)
	if err != nil {
		return "", err
	}
	// Beep is intentionally not here: bridgeapp.emitRead / armScanLua play the
	// tone only after a decoded UUID (avoids coin-on-garbage from dual-arm COM noise).
	return uid, nil
}

// parseUUIDBytesForFamily extracts 16 logical-UUID bytes for the given family.
// Continuous-arm transcripts always include both UL and Classic read attempts;
// when the SAK-selected family fails, try the other before giving up.
func parseUUIDBytesForFamily(stdout string, fam TagFamily) ([]byte, error) {
	switch fam {
	case TagFamilyClassic1K:
		raw, err := parseClassicBlockDump(stdout)
		if err == nil && len(raw) == 16 && !isZeroBlock(raw) {
			return raw, nil
		}
		if uraw, uerr := parseLogicalUUIDBytes(stdout); uerr == nil && len(uraw) == 16 && !isZeroBlock(uraw) {
			return uraw, nil
		}
		return raw, err
	case TagFamilyUltralight:
		raw, err := parseLogicalUUIDBytes(stdout)
		if err == nil && len(raw) == 16 && !isZeroBlock(raw) {
			return raw, nil
		}
		if craw, cerr := parseClassicBlockDump(stdout); cerr == nil && len(craw) == 16 && !isZeroBlock(craw) {
			return craw, nil
		}
		return raw, err
	default:
		raw, err := parseLogicalUUIDBytes(stdout)
		if err == nil && len(raw) == 16 && !isZeroBlock(raw) {
			return raw, nil
		}
		if craw, cerr := parseClassicBlockDump(stdout); cerr == nil && len(craw) == 16 {
			return craw, nil
		}
		return raw, err
	}
}

func (r *CLIProxmarkReader) cancelArmAndWait() {
	if r == nil {
		return
	}
	r.mu.Lock()
	cancel := r.armCancel
	done := r.armDone
	arm := r.luaArm
	r.luaArm = nil
	if cancel != nil {
		cancel()
	}
	if r.useSession {
		_ = r.closeSessionLocked()
	}
	r.mu.Unlock()
	if arm != nil {
		_ = arm.Close()
	}
	if done == nil {
		return
	}
	select {
	case <-done:
	case <-time.After(3 * time.Second):
	}
}

// SetHFGain sets the HF antenna gain (1–63, default 63) and reapplies the
// Proxmark threshold when a session or one-shot runner is active.
func (r *CLIProxmarkReader) SetHFGain(g int) {
	if r == nil {
		return
	}
	r.mu.Lock()
	r.hfGain = ClampHFGain(g)
	r.threshApplied = false
	// Lua arm bakes thresh into the script — restart so the new gain applies.
	arm := r.luaArm
	r.luaArm = nil
	if (!r.useSession && r.runner != nil) || r.session != nil {
		r.applyThreshLocked(context.Background())
	}
	r.mu.Unlock()
	if arm != nil {
		_ = arm.Close()
	}
}

func hfThreshCommand(gain int) string {
	return fmt.Sprintf("hw sethfthresh -t %d", HFThreshFromGain(gain))
}

// applyThreshLocked runs ensureThreshLocked and logs failures without aborting
// the caller's primary command. Caller must hold r.mu.
func (r *CLIProxmarkReader) applyThreshLocked(ctx context.Context) {
	if err := r.ensureThreshLocked(ctx); err != nil {
		log.Printf("proxmark hf thresh apply failed (gain=%d): %v", r.hfGain, err)
	}
}

// ensureThreshLocked applies hw sethfthresh once per connection/session.
// Caller must hold r.mu. On failure, threshApplied stays false for retry.
func (r *CLIProxmarkReader) ensureThreshLocked(ctx context.Context) error {
	if r.threshApplied {
		return nil
	}
	cmd := hfThreshCommand(r.hfGain)
	var err error
	if !r.useSession {
		_, err = r.runner(cmd)
	} else if r.session != nil {
		_, err = r.session.Run(ctx, cmd)
	} else {
		return nil
	}
	if err != nil {
		return err
	}
	r.threshApplied = true
	return nil
}

// SetBeepEnabled toggles the laptop beep on successful Poll (write-only disables it).
func (r *CLIProxmarkReader) SetBeepEnabled(enabled bool) {
	if r == nil {
		return
	}
	r.beepEnabled.Store(enabled)
}

// DetectISO14443A probes for an ISO14443-A tag (NTAG / Ultralight / Classic).
// Returns combined CLI stdout for diagnostics.
//
// Proxmark3 often exits non-zero (e.g. -10) when no card answers; that is treated
// as present=false when the device itself responded.
func (r *CLIProxmarkReader) DetectISO14443A() (present bool, stdout string, err error) {
	if !r.IsAvailable() {
		return false, "", ErrHardwareUnavailable
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	ctx, cancel := context.WithTimeout(context.Background(), proxmarkSessionWriteTimeout)
	defer cancel()
	stdout, runErr := r.runLocked(ctx, "hf 14a reader")
	lower := strings.ToLower(stdout)
	// Require "uid:" (with colon) — bare "uid" false-positives on paths containing "uuid".
	present = strings.Contains(lower, "uid:") ||
		(strings.Contains(lower, "atqa") && strings.Contains(lower, "sak"))
	if present {
		return true, stdout, nil
	}
	if runErr != nil && !pm3DeviceResponded(stdout) {
		return false, stdout, runErr
	}
	return false, stdout, nil
}

// Close tears down any persistent Proxmark session and cancels an armed scan.
func (r *CLIProxmarkReader) Close() error {
	if r == nil {
		return nil
	}
	r.cancelArmAndWait()
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.closeSessionLocked()
}

// runLocked executes a pm3 command. Caller must hold r.mu.
func (r *CLIProxmarkReader) runLocked(ctx context.Context, command string) (string, error) {
	if r.useSession {
		if err := r.ensureSessionLocked(ctx); err != nil {
			return "", err
		}
		r.applyThreshLocked(ctx)
		stdout, err := r.session.Run(ctx, command)
		if err != nil {
			// Card-select / empty-antenna failures still print a prompt. Keep the
			// session warm so continuous Poll does not thrash COM reconnect backoff.
			if pm3DeviceResponded(stdout) {
				return stdout, err
			}
			_ = r.closeSessionLocked()
			r.scheduleRetryLocked()
			return stdout, err
		}
		r.backoff = proxmarkReconnectMinBackoff
		return stdout, nil
	}
	r.applyThreshLocked(ctx)
	return r.runner(command)
}

func (r *CLIProxmarkReader) ensureSessionLocked(ctx context.Context) error {
	if r.session != nil {
		return nil
	}
	now := time.Now()
	if now.Before(r.nextRetry) {
		return fmt.Errorf("proxmark session backoff until %s", r.nextRetry.Format(time.RFC3339))
	}
	sess, err := r.sessionFactory(ctx)
	if err != nil {
		r.scheduleRetryLocked()
		return err
	}
	r.session = sess
	r.threshApplied = false
	return nil
}

func (r *CLIProxmarkReader) closeSessionLocked() error {
	if r.session == nil {
		return nil
	}
	err := r.session.Close()
	r.session = nil
	r.threshApplied = false
	return err
}

func (r *CLIProxmarkReader) scheduleRetryLocked() {
	if r.backoff <= 0 {
		r.backoff = proxmarkReconnectMinBackoff
	}
	r.nextRetry = time.Now().Add(r.backoff)
	r.backoff *= 2
	if r.backoff > proxmarkReconnectMaxBackoff {
		r.backoff = proxmarkReconnectMaxBackoff
	}
}

// parseLogicalUUIDBytes extracts 16 user-memory bytes from pm3 read output:
// a Type-2 raw READ line, a legacy `Data :` line with 16 bytes, or four
// labeled page rows (4–7). It must not scrape hex from the whole transcript
// (log paths / page labels otherwise fabricate garbage UUIDs like bebebe04-…).
func parseLogicalUUIDBytes(stdout string) ([]byte, error) {
	if strings.TrimSpace(stdout) == "" {
		return nil, nil
	}
	if raw, ok := parseRaw14aRead16(stdout); ok {
		return raw, nil
	}
	if raw, ok := parseSingleDataLine16(stdout); ok {
		return raw, nil
	}
	raw := make([]byte, 0, 16)
	for i := 0; i < proxmarkLogicalUUIDPages; i++ {
		page := proxmarkUserMemoryStartPage + i
		pageBytes, err := parseReadPageOutput(stdout, page)
		if err != nil {
			return nil, fmt.Errorf("read page %d: %w", page, err)
		}
		if len(pageBytes) == 0 {
			return nil, nil
		}
		raw = append(raw, pageBytes...)
	}
	if len(raw) != 16 {
		return nil, fmt.Errorf("parse read: got %d bytes, need 16", len(raw))
	}
	return raw, nil
}

// raw14aReadPattern matches `hf 14a raw` Type-2 READ responses:
//
//	[+] 23 65 7B 2D AA 08 5F E8 85 53 E9 E3 AF FB 46 78 [ 4B A1 ]
var raw14aReadPattern = regexp.MustCompile(`(?m)^\[\+\]\s*((?:[0-9A-Fa-f]{2}\s+){15}[0-9A-Fa-f]{2})\s*(?:\[|$)`)

func parseRaw14aRead16(stdout string) ([]byte, bool) {
	m := raw14aReadPattern.FindStringSubmatch(stdout)
	if m == nil {
		return nil, false
	}
	raw, ok := extractHexBytes(m[1])
	if !ok || len(raw) < 16 {
		return nil, false
	}
	return raw[:16], true
}

// parseSingleDataLine16 accepts only an unlabeled `Data : aa bb …` line with
// ≥16 bytes. Table headers like `Block# | Data | Ascii` are ignored.
func parseSingleDataLine16(stdout string) ([]byte, bool) {
	for _, line := range strings.Split(stdout, "\n") {
		if strings.Contains(line, "|") {
			continue
		}
		lower := strings.ToLower(line)
		if !strings.Contains(lower, "data") {
			continue
		}
		if raw, ok := extractHexBytes(line); ok && len(raw) >= 16 {
			return raw[:16], true
		}
	}
	return nil, false
}

func pm3DeviceResponded(stdout string) bool {
	return strings.Contains(stdout, "Communicating with PM3") ||
		strings.Contains(stdout, "pm3 -->") ||
		strings.Contains(stdout, "Using UART port")
}

// pipePageDataPattern matches pm3 table rows like:
//
//	[=] 04/0x04 | 11 22 33 44 | ....
//	[=]   4 | 14 41 67 4d | ....
//
// Captures only the data column so block labels (04/0x04) are not mistaken for payload.
var pipePageDataPattern = regexp.MustCompile(`(?i)\|\s*((?:[0-9a-f]{2}\s+){3}[0-9a-f]{2})\s*\|`)

// parseReadPageOutput extracts 4 data bytes from pm3 `hf mfu rdbl -b N` stdout.
//
// Typical formats:
//
//	[=] 04/0x04 | 14 41 67 4d | ....
//	[=]   4 | 14 41 67 4d | ....
//	Data : 14 41 67 4D
func parseReadPageOutput(stdout string, page int) ([]byte, error) {
	if strings.TrimSpace(stdout) == "" {
		return nil, nil
	}

	// Match the page label only in the column before the first `|`, so hex
	// nibbles like "4d" / "17" in later pages are not mistaken for page 4 / 7.
	pageLabel := regexp.MustCompile(fmt.Sprintf(
		`(?i)(?:^|[^0-9a-fx])(?:0x)?0*%d(?:/0x[0-9a-f]+)?(?:[^0-9a-f]|$)`,
		page,
	))
	lines := strings.Split(stdout, "\n")
	var dataFallback []byte
	for _, line := range lines {
		lower := strings.ToLower(line)
		label, data, hasPipe := strings.Cut(line, "|")
		if hasPipe {
			if !pageLabel.MatchString(label) {
				continue
			}
			if raw, ok := extractPipeColumnPage("|" + data); ok {
				return raw, nil
			}
			if raw, ok := extractHexBytes(data); ok && len(raw) >= proxmarkPageSize {
				return raw[:proxmarkPageSize], nil
			}
			continue
		}
		if strings.Contains(lower, "data") {
			if raw, ok := extractHexBytes(line); ok && len(raw) >= proxmarkPageSize {
				// Prefer page-labeled rows; keep unlabeled "Data :" as last resort
				// for single-page CLI dumps used in unit tests.
				if pageLabel.MatchString(line) {
					return raw[:proxmarkPageSize], nil
				}
				if dataFallback == nil {
					dataFallback = raw[:proxmarkPageSize]
				}
			}
		}
	}
	if dataFallback != nil {
		return dataFallback, nil
	}

	return nil, fmt.Errorf("parse read page: no hex payload in output")
}

func extractPipeColumnPage(line string) ([]byte, bool) {
	if m := pipePageDataPattern.FindStringSubmatch(line); len(m) == 2 {
		if raw, ok := extractHexBytes(m[1]); ok && len(raw) >= proxmarkPageSize {
			return raw[:proxmarkPageSize], true
		}
	}
	// Single-column rows used in tests: "[=]   4 | 14 41 67 4d"
	if i := strings.Index(line, "|"); i >= 0 {
		if raw, ok := extractHexBytes(line[i+1:]); ok && len(raw) >= proxmarkPageSize {
			return raw[:proxmarkPageSize], true
		}
	}
	return nil, false
}

// parseReadBlockOutput is kept for unit tests of legacy 16-byte dumps.
func parseReadBlockOutput(stdout string) ([]byte, error) {
	if strings.TrimSpace(stdout) == "" {
		return nil, nil
	}

	lines := strings.Split(stdout, "\n")
	for _, line := range lines {
		lower := strings.ToLower(line)
		if strings.Contains(lower, "data") || strings.Contains(line, "|") {
			if raw, ok := extractHexBytes(line); ok && len(raw) >= 16 {
				return raw[:16], nil
			}
		}
	}

	if raw, ok := extractHexBytes(stdout); ok {
		if len(raw) < 16 {
			return nil, fmt.Errorf("parse read block: found %d bytes, need 16", len(raw))
		}
		return raw[:16], nil
	}

	return nil, fmt.Errorf("parse read block: no hex payload in output")
}

var hexBytePattern = regexp.MustCompile(`(?i)\b[0-9a-f]{2}\b`)

func extractHexBytes(s string) ([]byte, bool) {
	matches := hexBytePattern.FindAllString(s, -1)
	if len(matches) == 0 {
		return nil, false
	}
	raw := make([]byte, 0, len(matches))
	for _, token := range matches {
		var b byte
		if _, err := fmt.Sscanf(token, "%02x", &b); err != nil {
			continue
		}
		raw = append(raw, b)
	}
	if len(raw) == 0 {
		return nil, false
	}
	return raw, true
}

func isZeroBlock(b []byte) bool {
	for _, v := range b {
		if v != 0 {
			return false
		}
	}
	return true
}
