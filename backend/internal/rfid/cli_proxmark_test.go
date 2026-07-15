package rfid

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCLIProxmarkReader_PollParsesFourPages(t *testing.T) {
	const logicalUUID = "1441674d-a011-471a-a601-722b88b117f5"
	pages := map[string]string{
		"hf mfu rdbl -b 4": "[=]   4 | 14 41 67 4d\n",
		"hf mfu rdbl -b 5": "[=]   5 | a0 11 47 1a\n",
		"hf mfu rdbl -b 6": "[=]   6 | a6 01 72 2b\n",
		"hf mfu rdbl -b 7": "[=]   7 | 88 b1 17 f5\n",
	}

	reader := NewCLIProxmarkReader(CLIProxmarkConfig{
		Enabled: true,
		Runner: func(command string) (string, error) {
			out, ok := pages[command]
			require.True(t, ok, "unexpected command %q", command)
			return out, nil
		},
	})

	got, err := reader.Poll()
	require.NoError(t, err)
	assert.Equal(t, logicalUUID, got)
}

func TestCLIProxmarkReader_WriteLogicalUUIDWritesFourPages(t *testing.T) {
	const logicalUUID = "1441674d-a011-471a-a601-722b88b117f5"
	var commands []string

	reader := NewCLIProxmarkReader(CLIProxmarkConfig{
		Enabled: true,
		Runner: func(command string) (string, error) {
			commands = append(commands, command)
			return "ok", nil
		},
	})

	err := reader.WriteLogicalUUID(logicalUUID)
	require.NoError(t, err)
	assert.Equal(t, []string{
		"hf mfu wrbl -b 4 -d 1441674d",
		"hf mfu wrbl -b 5 -d a011471a",
		"hf mfu wrbl -b 6 -d a601722b",
		"hf mfu wrbl -b 7 -d 88b117f5",
	}, commands)
}

func TestCLIProxmarkReader_PollEmptyPagesReturnsEmpty(t *testing.T) {
	reader := NewCLIProxmarkReader(CLIProxmarkConfig{
		Enabled: true,
		Runner: func(command string) (string, error) {
			return "Data : 00 00 00 00", nil
		},
	})

	got, err := reader.Poll()
	require.NoError(t, err)
	assert.Empty(t, got)
}

func TestCLIProxmarkReader_PollEmptyOutputReturnsEmpty(t *testing.T) {
	reader := NewCLIProxmarkReader(CLIProxmarkConfig{
		Enabled: true,
		Runner: func(command string) (string, error) {
			return "", nil
		},
	})

	got, err := reader.Poll()
	require.NoError(t, err)
	assert.Empty(t, got)
}

func TestCLIProxmarkReader_PollUnavailable(t *testing.T) {
	reader := NewCLIProxmarkReader(CLIProxmarkConfig{Enabled: false})
	_, err := reader.Poll()
	assert.ErrorIs(t, err, ErrHardwareUnavailable)
}

func TestCLIProxmarkReader_WriteUnavailable(t *testing.T) {
	reader := NewCLIProxmarkReader(CLIProxmarkConfig{Enabled: false})
	err := reader.WriteLogicalUUID("1441674d-a011-471a-a601-722b88b117f5")
	assert.ErrorIs(t, err, ErrHardwareUnavailable)
}

func TestCLIProxmarkReader_WriteRejectsInvalidUUID(t *testing.T) {
	reader := NewCLIProxmarkReader(CLIProxmarkConfig{
		Enabled: true,
		Runner: func(command string) (string, error) {
			t.Fatal("runner must not be invoked for invalid UUID")
			return "", nil
		},
	})

	err := reader.WriteLogicalUUID("DEMO-TAG-0001")
	require.Error(t, err)
}

func TestCLIProxmarkReader_PollRunnerError(t *testing.T) {
	reader := NewCLIProxmarkReader(CLIProxmarkConfig{
		Enabled: true,
		Runner: func(command string) (string, error) {
			return "no tag", errors.New("exit status 1")
		},
	})

	_, err := reader.Poll()
	require.Error(t, err)
}

func TestCLIProxmarkReader_PollParseFailure(t *testing.T) {
	reader := NewCLIProxmarkReader(CLIProxmarkConfig{
		Enabled: true,
		Runner: func(command string) (string, error) {
			return "tag present but no hex dump", nil
		},
	})

	_, err := reader.Poll()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no hex payload")
}

func TestParseReadBlockOutput_DataLineFormat(t *testing.T) {
	raw, err := parseReadBlockOutput("Data : 14 41 67 4D A0 11 47 1A A6 01 72 2B 88 B1 17 F5\n")
	require.NoError(t, err)
	require.Len(t, raw, 16)

	got, err := DecodeLogicalUUID(raw)
	require.NoError(t, err)
	assert.Equal(t, "1441674d-a011-471a-a601-722b88b117f5", got)
}

func TestParseReadPageOutput(t *testing.T) {
	raw, err := parseReadPageOutput("[=]   4 | 14 41 67 4d | ....\n", 4)
	require.NoError(t, err)
	require.Equal(t, []byte{0x14, 0x41, 0x67, 0x4d}, raw)
}

func TestCLIProxmarkReader_DetectISO14443A(t *testing.T) {
	reader := NewCLIProxmarkReader(CLIProxmarkConfig{
		Enabled: true,
		Runner: func(command string) (string, error) {
			assert.Equal(t, "hf 14a reader", command)
			return "[+] UID: 04 12 34 56 78 9A 80\n[+] ATQA: 00 44\n[+] SAK: 00\n", nil
		},
	})
	present, out, err := reader.DetectISO14443A()
	require.NoError(t, err)
	assert.True(t, present)
	assert.Contains(t, strings.ToLower(out), "uid:")
}

func TestCLIProxmarkReader_DetectISO14443A_NoTagExitCode(t *testing.T) {
	reader := NewCLIProxmarkReader(CLIProxmarkConfig{
		Enabled: true,
		Runner: func(command string) (string, error) {
			return "[+] Using UART port COM3\n[+] Communicating with PM3 over USB-CDC\n[usb|script] pm3 --> hf 14a reader\n", errors.New("exit status 0xfffffff6")
		},
	})
	present, _, err := reader.DetectISO14443A()
	require.NoError(t, err)
	assert.False(t, present)
}

func TestCLIProxmarkReader_IsAvailable(t *testing.T) {
	assert.True(t, NewCLIProxmarkReader(CLIProxmarkConfig{Enabled: true}).IsAvailable())
	assert.False(t, NewCLIProxmarkReader(CLIProxmarkConfig{Enabled: false}).IsAvailable())
}

func TestCLIProxmarkReader_WritePageCommandFormat(t *testing.T) {
	// Guard against regressing to invalid --blk / 16-byte single-page writes.
	reader := NewCLIProxmarkReader(CLIProxmarkConfig{
		Enabled: true,
		Runner: func(command string) (string, error) {
			assert.NotContains(t, command, "--blk")
			assert.Contains(t, command, "-b ")
			assert.True(t, strings.HasPrefix(command, "hf mfu wrbl "))
			parts := strings.Fields(command)
			require.GreaterOrEqual(t, len(parts), 6)
			dIdx := -1
			for i, p := range parts {
				if p == "-d" {
					dIdx = i
					break
				}
			}
			require.Greater(t, dIdx, 0)
			require.Equal(t, 8, len(parts[dIdx+1]), "page writes must be 4 bytes (8 hex chars): %s", command)
			return "ok", nil
		},
	})
	require.NoError(t, reader.WriteLogicalUUID("550e8400-e29b-41d4-a716-446655440099"))
}

func TestCLIProxmarkReader_PollPageCommandFormat(t *testing.T) {
	seen := map[int]bool{}
	reader := NewCLIProxmarkReader(CLIProxmarkConfig{
		Enabled: true,
		Runner: func(command string) (string, error) {
			var page int
			_, err := fmt.Sscanf(command, "hf mfu rdbl -b %d", &page)
			require.NoError(t, err)
			seen[page] = true
			return fmt.Sprintf("[=]   %d | 00 00 00 00\n", page), nil
		},
	})
	_, err := reader.Poll()
	require.NoError(t, err)
	assert.Equal(t, map[int]bool{4: true, 5: true, 6: true, 7: true}, seen)
}
