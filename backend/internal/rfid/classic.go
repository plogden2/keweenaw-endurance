package rfid

import (
	"fmt"
	"regexp"
	"strings"
)

const (
	classicLogicalUUIDBlock = 1
	classicDefaultKeyHex    = "FFFFFFFFFFFF"
)

func classicReadBlock1Cmd() string {
	return fmt.Sprintf("hf mf rdbl --blk %d -k %s", classicLogicalUUIDBlock, classicDefaultKeyHex)
}

func classicWriteBlock1Cmd(hex16 string) string {
	return fmt.Sprintf(
		"hf mf wrbl --blk %d -d %s -k %s",
		classicLogicalUUIDBlock,
		strings.ToLower(hex16),
		classicDefaultKeyHex,
	)
}

// classicBlockPipePattern matches mf rdbl table rows for block 1:
//
//	[=] 01/0x01 | 14 41 67 4D … F5 | ....
//	[=]   1 | 14 41 67 4D … F5 | ....
var classicBlockPipePattern = regexp.MustCompile(
	`(?i)(?:^|[^0-9a-fx])(?:0x)?0*1(?:/0x[0-9a-f]+)?\s*\|\s*((?:[0-9a-f]{2}\s+){15}[0-9a-f]{2})\s*\|`,
)

var classicBlockLabelPattern = regexp.MustCompile(`(?i)(?:^|[^0-9a-f])0*1(?:[^0-9a-f]|$)`)

// parseClassicBlockDump extracts 16 bytes from `hf mf rdbl --blk 1` stdout.
func parseClassicBlockDump(stdout string) ([]byte, error) {
	if strings.TrimSpace(stdout) == "" {
		return nil, nil
	}
	if m := classicBlockPipePattern.FindStringSubmatch(stdout); len(m) == 2 {
		if raw, ok := extractHexBytes(m[1]); ok && len(raw) >= 16 {
			return raw[:16], nil
		}
	}
	// Fall back to unlabeled Data lines / any 16-byte hex dump on a data row.
	if raw, ok := parseSingleDataLine16(stdout); ok {
		return raw, nil
	}
	for _, line := range strings.Split(stdout, "\n") {
		if !strings.Contains(line, "|") {
			continue
		}
		label, data, ok := strings.Cut(line, "|")
		if !ok {
			continue
		}
		lowerLabel := strings.ToLower(label)
		if !(strings.Contains(lowerLabel, "01/0x01") || classicBlockLabelPattern.MatchString(label)) {
			continue
		}
		if raw, ok := extractHexBytes(data); ok && len(raw) >= 16 {
			return raw[:16], nil
		}
	}
	return nil, fmt.Errorf("parse classic block: no hex payload in output")
}
