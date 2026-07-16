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
// Tests inject a fake runner to avoid requiring real hardware.
type CLICommandRunner func(command string) (stdout string, err error)

// CLIProxmarkConfig configures the pm3 CLI bridge.
type CLIProxmarkConfig struct {
	CLIPath string
	Port    string
	Enabled bool
	Runner  CLICommandRunner
}

// CLIProxmarkReader reads and writes logical UUIDs via the Proxmark3 CLI.
// A mutex serializes all CLI invocations so Poll and WriteTag cannot race on
// the serial port (each pm3 process needs exclusive COM access).
type CLIProxmarkReader struct {
	cliPath string
	port    string
	enabled bool
	runner  CLICommandRunner
	mu      sync.Mutex
}

func NewCLIProxmarkReader(cfg CLIProxmarkConfig) *CLIProxmarkReader {
	cliPath := cfg.CLIPath
	if cliPath == "" {
		cliPath = "pm3"
	}
	runner := cfg.Runner
	if runner == nil {
		runner = defaultCLICommandRunner(cliPath, cfg.Port)
	}
	return &CLIProxmarkReader{
		cliPath: cliPath,
		port:    cfg.Port,
		enabled: cfg.Enabled,
		runner:  runner,
	}
}

func defaultCLICommandRunner(cliPath, port string) CLICommandRunner {
	mingwBin := proxmarkMingwBin(cliPath)
	return func(command string) (string, error) {
		ctx, cancel := context.WithTimeout(context.Background(), proxmarkCLIExecTimeout)
		defer cancel()

		args := []string{}
		if port != "" {
			args = append(args, "-p", port)
		}
		args = append(args, "-f", "--incognito", "-c", command)

		cmd := exec.CommandContext(ctx, cliPath, args...)
		if mingwBin != "" {
			cmd.Env = withPrependedPath(os.Environ(), mingwBin)
		}
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

// proxmarkMingwBin returns the mingw64 bin dir needed for ProxSpace-built
// Windows clients to resolve DLLs when spawned from Go.
func proxmarkMingwBin(cliPath string) string {
	if runtime.GOOS != "windows" {
		return ""
	}
	if mingw := os.Getenv("PROXMARK3_MINGW_BIN"); mingw != "" {
		return mingw
	}
	if !strings.Contains(strings.ToLower(cliPath), "proxspace") {
		return ""
	}
	candidate := filepath.Clean(filepath.Join(filepath.Dir(cliPath), "..", "..", "..", "msys2", "mingw64", "bin"))
	if st, err := os.Stat(candidate); err == nil && st.IsDir() {
		return candidate
	}
	return ""
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
	// One pm3 process for all four pages — each spawn is expensive on Windows/COM.
	parts := make([]string, 0, proxmarkLogicalUUIDPages)
	for i := 0; i < proxmarkLogicalUUIDPages; i++ {
		page := proxmarkUserMemoryStartPage + i
		off := i * proxmarkPageSize
		hexData := fmt.Sprintf("%x", raw[off:off+proxmarkPageSize])
		parts = append(parts, fmt.Sprintf("hf mfu wrbl -b %d -d %s", page, hexData))
	}
	if _, err := r.runner(strings.Join(parts, "; ")); err != nil {
		return fmt.Errorf("write pages %d-%d: %w",
			proxmarkUserMemoryStartPage,
			proxmarkUserMemoryStartPage+proxmarkLogicalUUIDPages-1,
			err)
	}
	return nil
}

func (r *CLIProxmarkReader) Poll() (string, error) {
	if !r.IsAvailable() {
		return "", ErrHardwareUnavailable
	}
	// Skip this tick if a write holds the port — writes must not wait behind a
	// full multi-page poll (Playwright write-tag timeout is otherwise too tight).
	if !r.mu.TryLock() {
		return "", nil
	}
	defer r.mu.Unlock()

	parts := make([]string, 0, proxmarkLogicalUUIDPages)
	for i := 0; i < proxmarkLogicalUUIDPages; i++ {
		page := proxmarkUserMemoryStartPage + i
		parts = append(parts, fmt.Sprintf("hf mfu rdbl -b %d", page))
	}
	stdout, err := r.runner(strings.Join(parts, "; "))
	if err != nil {
		return "", fmt.Errorf("read pages: %w", err)
	}

	raw := make([]byte, 0, 16)
	for i := 0; i < proxmarkLogicalUUIDPages; i++ {
		page := proxmarkUserMemoryStartPage + i
		pageBytes, err := parseReadPageOutput(stdout, page)
		if err != nil {
			return "", fmt.Errorf("read page %d: %w", page, err)
		}
		if len(pageBytes) == 0 {
			return "", nil
		}
		raw = append(raw, pageBytes...)
	}
	if isZeroBlock(raw) {
		return "", nil
	}
	return DecodeLogicalUUID(raw)
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
	stdout, runErr := r.runner("hf 14a reader")
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
