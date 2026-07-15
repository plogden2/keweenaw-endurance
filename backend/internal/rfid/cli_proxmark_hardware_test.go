package rfid

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestHardwareProxmark3Smoke exercises a real Proxmark3 CLI read/write round-trip.
//
// Skipped unless RFID_HARDWARE=true (never required in CI). Place an NTAG /
// MIFARE Ultralight (ISO14443-A) tag on the HF antenna before running.
//
// Windows (this machine):
//
//	$env:RFID_HARDWARE="true"
//	$env:PROXMARK3_CLI="C:\Users\gener\sdk\ProxSpace\pm3\proxmark3\client\proxmark3.exe"
//	$env:PROXMARK3_PORT="COM3"
//	$env:PATH="C:\Users\gener\sdk\ProxSpace\msys2\mingw64\bin;$env:PATH"
//	go test ./internal/rfid -run 'TestHardware' -count=1 -v
//
// Or use scripts/pm3.cmd from the repo root for manual CLI checks.
func TestHardwareProxmark3DetectTag(t *testing.T) {
	reader := requireHardwareReader(t)

	present, stdout, err := reader.DetectISO14443A()
	require.NoError(t, err, "hf 14a reader failed; stdout:\n%s", stdout)
	if !present {
		t.Fatalf("no ISO14443-A tag detected on the HF antenna.\n"+
			"Place an NTAG213/215/216 or MIFARE Ultralight on the Proxmark HF coil and re-run.\n"+
			"CLI output:\n%s", stdout)
	}
	t.Logf("tag present:\n%s", stdout)
}

func TestHardwareProxmark3Smoke(t *testing.T) {
	reader := requireHardwareReader(t)

	present, stdout, err := reader.DetectISO14443A()
	require.NoError(t, err, "detect failed; stdout:\n%s", stdout)
	require.True(t, present, "no ISO14443-A tag on antenna; stdout:\n%s", stdout)

	logicalUUID := strings.TrimSpace(os.Getenv("PROXMARK3_SMOKE_UUID"))
	if logicalUUID == "" {
		logicalUUID = "550e8400-e29b-41d4-a716-446655440099"
	}

	require.NoError(t, reader.WriteLogicalUUID(logicalUUID), "write logical UUID to tag on reader")

	got, err := reader.Poll()
	require.NoError(t, err, "poll user memory from tag on reader")
	assert.Equal(t, strings.ToLower(logicalUUID), strings.ToLower(got))
}

func TestHardwareProxmark3RewriteSameUUID(t *testing.T) {
	reader := requireHardwareReader(t)

	present, stdout, err := reader.DetectISO14443A()
	require.NoError(t, err, "detect failed; stdout:\n%s", stdout)
	require.True(t, present, "no ISO14443-A tag on antenna; stdout:\n%s", stdout)

	logicalUUID := "1441674d-a011-471a-a601-722b88b117f5"
	require.NoError(t, reader.WriteLogicalUUID(logicalUUID))
	got1, err := reader.Poll()
	require.NoError(t, err)
	require.Equal(t, logicalUUID, strings.ToLower(got1))

	// Replacement-tag model: re-program the same logical UUID onto the chip.
	require.NoError(t, reader.WriteLogicalUUID(logicalUUID))
	got2, err := reader.Poll()
	require.NoError(t, err)
	assert.Equal(t, logicalUUID, strings.ToLower(got2))
}

func requireHardwareReader(t *testing.T) *CLIProxmarkReader {
	t.Helper()
	if !envBool("RFID_HARDWARE") {
		t.Skip("skipping hardware test: set RFID_HARDWARE=true with Proxmark3 + HF tag attached")
	}

	cliPath := os.Getenv("PROXMARK3_CLI")
	if cliPath == "" {
		cliPath = "pm3"
	}
	port := os.Getenv("PROXMARK3_PORT")
	if port == "" && runtime.GOOS == "windows" {
		port = "COM3"
	}

	// Ensure mingw DLLs resolve when using a ProxSpace-built Windows client.
	if runtime.GOOS == "windows" {
		if mingw := os.Getenv("PROXMARK3_MINGW_BIN"); mingw != "" {
			os.Setenv("PATH", mingw+string(os.PathListSeparator)+os.Getenv("PATH"))
		} else if strings.Contains(strings.ToLower(cliPath), "proxspace") {
			candidate := filepath.Clean(filepath.Join(filepath.Dir(cliPath), "..", "..", "..", "msys2", "mingw64", "bin"))
			if st, err := os.Stat(candidate); err == nil && st.IsDir() {
				os.Setenv("PATH", candidate+string(os.PathListSeparator)+os.Getenv("PATH"))
			}
		}
	}

	reader := NewCLIProxmarkReader(CLIProxmarkConfig{
		CLIPath: cliPath,
		Port:    port,
		Enabled: true,
	})
	require.True(t, reader.IsAvailable(), "CLI Proxmark reader must be enabled")
	return reader
}
