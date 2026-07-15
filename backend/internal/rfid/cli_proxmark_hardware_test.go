package rfid

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestHardwareProxmark3Smoke exercises a real Proxmark3 CLI read/write round-trip.
//
// Skipped unless RFID_HARDWARE=true (never required in CI). Attach a blank NTAG/MIFARE
// Ultralight tag on the reader before running.
//
// Full manual acceptance (stack + UI):
//  1. Place tag on reader
//  2. PIN auth → POST /api/rfid/write-tag {"participant_id":"<seeded racer id>"}
//  3. Confirm Poll / live WebSocket delivers that racer's logical UUID
//  4. Confirm lap or test-read popup in the SPA
//
// Run from backend/:
//
//	RFID_HARDWARE=true go test ./internal/rfid -run TestHardwareProxmark3Smoke -v
//
// Optional: PROXMARK3_CLI, PROXMARK3_PORT, PROXMARK3_SMOKE_UUID (defaults to a test UUID).
func TestHardwareProxmark3Smoke(t *testing.T) {
	if !envBool("RFID_HARDWARE") {
		t.Skip("skipping hardware smoke: set RFID_HARDWARE=true with Proxmark3 attached")
	}

	logicalUUID := strings.TrimSpace(os.Getenv("PROXMARK3_SMOKE_UUID"))
	if logicalUUID == "" {
		logicalUUID = "550e8400-e29b-41d4-a716-446655440099"
	}

	cliPath := os.Getenv("PROXMARK3_CLI")
	if cliPath == "" {
		cliPath = "pm3"
	}

	reader := NewCLIProxmarkReader(CLIProxmarkConfig{
		CLIPath: cliPath,
		Port:    os.Getenv("PROXMARK3_PORT"),
		Enabled: true,
	})
	require.True(t, reader.IsAvailable(), "CLI Proxmark reader must be enabled")

	require.NoError(t, reader.WriteLogicalUUID(logicalUUID), "write logical UUID to tag on reader")

	got, err := reader.Poll()
	require.NoError(t, err, "poll user memory from tag on reader")
	assert.Equal(t, strings.ToLower(logicalUUID), strings.ToLower(got))
}
