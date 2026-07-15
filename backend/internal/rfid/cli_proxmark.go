package rfid

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

// Proxmark3 CLI commands for NTAG/MIFARE Ultralight user-memory block 4.
// Task 7 may tune block index or command variants for the attached reader/card.
const (
	proxmarkReadUserBlockCmd  = "hf mfu rdbl --blk 4"
	proxmarkWriteUserBlockFmt = "hf mfu wrbl --blk 4 -d %s"
	proxmarkCLIExecTimeout    = 10 * time.Second
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
		args = append(args, "-c", command)

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
	hexPayload, err := EncodeLogicalUUIDHex(logicalUUID)
	if err != nil {
		return err
	}
	command := fmt.Sprintf(proxmarkWriteUserBlockFmt, hexPayload)
	if _, err := r.runner(command); err != nil {
		return err
	}
	return nil
}

func (r *CLIProxmarkReader) Poll() (string, error) {
	if !r.IsAvailable() {
		return "", ErrHardwareUnavailable
	}
	stdout, err := r.runner(proxmarkReadUserBlockCmd)
	if err != nil {
		return "", err
	}
	raw, err := parseReadBlockOutput(stdout)
	if err != nil {
		return "", err
	}
	if len(raw) == 0 || isZeroBlock(raw) {
		return "", nil
	}
	return DecodeLogicalUUID(raw)
}

// parseReadBlockOutput extracts 16 data bytes from pm3 `hf mfu rdbl` stdout.
//
// Expected formats (Task 7 may refine against real hardware output):
//
//	[=] blk | data
//	[=] ----+----------------------------------------------------------------
//	[=]   4 | 14 41 67 4d a0 11 47 1a a6 01 72 2b 88 b1 17 f5
//
// or:
//
//	Data : 14 41 67 4D A0 11 47 1A A6 01 72 2B 88 B1 17 F5
//
// Hex tokens are collected from the best matching line; at least 16 bytes are required.
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
