package rfid

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
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
type CLIProxmarkReader struct {
	cliPath string
	port    string
	enabled bool
	runner  CLICommandRunner
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
	return func(command string) (string, error) {
		ctx, cancel := context.WithTimeout(context.Background(), proxmarkCLIExecTimeout)
		defer cancel()

		args := []string{}
		if port != "" {
			args = append(args, "-p", port)
		}
		args = append(args, "-f", "--incognito", "-c", command)

		cmd := exec.CommandContext(ctx, cliPath, args...)
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
	// Ultralight/NTAG pages are 4 bytes. A 16-byte "compatibility write" only
	// commits the first 4 bytes on these cards — write four pages explicitly.
	for i := 0; i < proxmarkLogicalUUIDPages; i++ {
		page := proxmarkUserMemoryStartPage + i
		off := i * proxmarkPageSize
		hexData := fmt.Sprintf("%x", raw[off:off+proxmarkPageSize])
		command := fmt.Sprintf("hf mfu wrbl -b %d -d %s", page, hexData)
		if _, err := r.runner(command); err != nil {
			return fmt.Errorf("write page %d: %w", page, err)
		}
	}
	return nil
}

func (r *CLIProxmarkReader) Poll() (string, error) {
	if !r.IsAvailable() {
		return "", ErrHardwareUnavailable
	}
	raw := make([]byte, 0, 16)
	for i := 0; i < proxmarkLogicalUUIDPages; i++ {
		page := proxmarkUserMemoryStartPage + i
		command := fmt.Sprintf("hf mfu rdbl -b %d", page)
		stdout, err := r.runner(command)
		if err != nil {
			return "", fmt.Errorf("read page %d: %w", page, err)
		}
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

// parseReadPageOutput extracts 4 data bytes from pm3 `hf mfu rdbl -b N` stdout.
//
// Typical formats:
//
//	[=]   4 | 14 41 67 4d | ....
//	Data : 14 41 67 4D
func parseReadPageOutput(stdout string, page int) ([]byte, error) {
	if strings.TrimSpace(stdout) == "" {
		return nil, nil
	}

	pageStr := fmt.Sprintf("%d", page)
	lines := strings.Split(stdout, "\n")
	for _, line := range lines {
		lower := strings.ToLower(line)
		if strings.Contains(lower, "data") || strings.Contains(line, "|") {
			if !strings.Contains(line, pageStr) && !strings.Contains(lower, "data") {
				continue
			}
			if raw, ok := extractHexBytes(line); ok && len(raw) >= proxmarkPageSize {
				return raw[:proxmarkPageSize], nil
			}
		}
	}

	if raw, ok := extractHexBytes(stdout); ok {
		if len(raw) < proxmarkPageSize {
			return nil, fmt.Errorf("parse read page: found %d bytes, need %d", len(raw), proxmarkPageSize)
		}
		// Prefer trailing page-sized slice when CLI dumps more hex (UID, etc.).
		if len(raw) > proxmarkPageSize {
			return raw[len(raw)-proxmarkPageSize:], nil
		}
		return raw[:proxmarkPageSize], nil
	}

	return nil, fmt.Errorf("parse read page: no hex payload in output")
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
