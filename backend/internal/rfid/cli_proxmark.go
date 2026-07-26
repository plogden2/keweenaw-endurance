package rfid

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"
)

// NTAG / MIFARE Ultralight user memory: 4-byte pages. Logical UUID is 16 bytes
// starting at page 4 (pages 4–7). Commands use `-b` / `--block` (not `--blk`).
const (
	proxmarkUserMemoryStartPage = 4
	proxmarkPageSize            = 4
	proxmarkLogicalUUIDPages    = 4 // 16 bytes
	proxmarkCLIExecTimeout      = 15 * time.Second
)

// CLICommandRunner executes the Proxmark3 CLI with the given pm3 subcommand string.
// Tests inject a fake runner to avoid requiring real hardware. When set, the
// reader uses one-shot process mode instead of a persistent session.
type CLICommandRunner func(command string) (stdout string, err error)

// CLIProxmarkConfig configures the pm3 CLI bridge.
type CLIProxmarkConfig struct {
	CLIPath        string
	Port           string
	Enabled        bool
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
	runner         CLICommandRunner // one-shot (tests); nil ⇒ persistent session
	sessionFactory PM3SessionFactory
	beeper         Beeper

	mu         sync.Mutex
	session    PM3Session
	nextRetry  time.Time
	backoff    time.Duration
	useSession bool
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
		beeper:  beeper,
		backoff: proxmarkReconnectMinBackoff,
	}
	if cfg.Runner != nil {
		r.runner = cfg.Runner
		r.useSession = false
	} else if cfg.SessionFactory != nil {
		r.sessionFactory = cfg.SessionFactory
		r.useSession = true
	} else if preferOneShotCLI() {
		// Windows Proxmark clients using linenoise often never emit "pm3 -->"
		// when stdin/stdout are pipes, so persistent sessions hang on startup.
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

		args := []string{}
		if port != "" {
			args = append(args, "-p", port)
		}
		args = append(args, "-f", "--incognito", "-c", command)

		cmd := exec.CommandContext(ctx, cliPath, args...)
		cmd.Env = proxmarkCommandEnv(cliPath)
		out, err := cmd.CombinedOutput()
		if ctx.Err() == context.DeadlineExceeded {
			return string(out), fmt.Errorf("proxmark3 cli %q: timed out after %s", command, proxmarkCLIExecTimeout)
		}
		if err != nil {
			return string(out), fmt.Errorf("proxmark3 cli %q: %w: %s", command, err, strings.TrimSpace(string(out)))
		}
		return string(out), nil
	}
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
	r.mu.Lock()
	defer r.mu.Unlock()
	parts := make([]string, 0, proxmarkLogicalUUIDPages)
	for i := 0; i < proxmarkLogicalUUIDPages; i++ {
		page := proxmarkUserMemoryStartPage + i
		off := i * proxmarkPageSize
		hexData := fmt.Sprintf("%x", raw[off:off+proxmarkPageSize])
		parts = append(parts, fmt.Sprintf("hf mfu wrbl -b %d -d %s", page, hexData))
	}
	ctx, cancel := context.WithTimeout(context.Background(), proxmarkSessionWriteTimeout)
	defer cancel()
	stdout, writeErr := r.runLocked(ctx, strings.Join(parts, "; "))
	if writeErr != nil {
		// Proxmark CLI often exits -10 (0xfffffff6) even when the device ran the
		// script; confirm by reading user memory before failing the operator.
		if pm3DeviceResponded(stdout) {
			if verifyErr := r.verifyLogicalUUIDLocked(logicalUUID); verifyErr == nil {
				return nil
			}
		}
		return fmt.Errorf("write pages %d-%d: %w",
			proxmarkUserMemoryStartPage,
			proxmarkUserMemoryStartPage+proxmarkLogicalUUIDPages-1,
			writeErr)
	}
	return nil
}

// verifyLogicalUUIDLocked reads pages 4–7 and checks they decode to logicalUUID.
// Caller must hold r.mu.
func (r *CLIProxmarkReader) verifyLogicalUUIDLocked(logicalUUID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), proxmarkSessionPollTimeout)
	defer cancel()
	stdout, err := r.runLocked(ctx, "hf mfu rdbl -b 4")
	if err != nil && !pm3DeviceResponded(stdout) {
		return err
	}
	raw, parseErr := parseLogicalUUIDBytes(stdout)
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
	stdout, err := r.runLocked(ctx, "hf mfu rdbl -b 4")
	if err != nil {
		// Continuous poll loop: never fail the tick hard — empty means "try again".
		_ = stdout
		return "", nil
	}

	raw, parseErr := parseLogicalUUIDBytes(stdout)
	if parseErr != nil {
		return "", parseErr
	}
	if len(raw) == 0 || isZeroBlock(raw) {
		return "", nil
	}
	uid, err := DecodeLogicalUUID(raw)
	if err != nil {
		return "", err
	}
	r.beeper.Beep()
	return uid, nil
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

// Close tears down any persistent Proxmark session.
func (r *CLIProxmarkReader) Close() error {
	if r == nil {
		return nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.closeSessionLocked()
}

// runLocked executes a pm3 command. Caller must hold r.mu.
func (r *CLIProxmarkReader) runLocked(ctx context.Context, command string) (string, error) {
	if !r.useSession {
		return r.runner(command)
	}
	if err := r.ensureSessionLocked(ctx); err != nil {
		return "", err
	}
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
	return nil
}

func (r *CLIProxmarkReader) closeSessionLocked() error {
	if r.session == nil {
		return nil
	}
	err := r.session.Close()
	r.session = nil
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

// parseLogicalUUIDBytes extracts 16 user-memory bytes from a single
// `hf mfu rdbl -b 4` transcript (Data : line or four page rows).
func parseLogicalUUIDBytes(stdout string) ([]byte, error) {
	if strings.TrimSpace(stdout) == "" {
		return nil, nil
	}
	if raw, err := parseReadBlockOutput(stdout); err == nil && len(raw) >= 16 {
		return raw[:16], nil
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
