package rfid

import (
	"os"
	"strings"
)

// envBool reports whether key is set to a truthy value (1, true, yes, on).
// Matches config.getEnvAsBool semantics for RFID_HARDWARE and similar flags.
func envBool(key string) bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(key))) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}
