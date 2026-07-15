package rfid

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCLIProxmarkReader_PollParsesBlockData(t *testing.T) {
	const logicalUUID = "1441674d-a011-471a-a601-722b88b117f5"
	fakeOut := `[=] blk | data
[=] ----+----------------------------------------------------------------
[=]   4 | 14 41 67 4d a0 11 47 1a a6 01 72 2b 88 b1 17 f5
`

	reader := NewCLIProxmarkReader(CLIProxmarkConfig{
		Enabled: true,
		Runner: func(command string) (string, error) {
			assert.Equal(t, proxmarkReadUserBlockCmd, command)
			return fakeOut, nil
		},
	})

	got, err := reader.Poll()
	require.NoError(t, err)
	assert.Equal(t, logicalUUID, got)
}

func TestCLIProxmarkReader_WriteLogicalUUIDInvokesWriteCommand(t *testing.T) {
	const logicalUUID = "1441674d-a011-471a-a601-722b88b117f5"
	var ranCommand string

	reader := NewCLIProxmarkReader(CLIProxmarkConfig{
		Enabled: true,
		Runner: func(command string) (string, error) {
			ranCommand = command
			return "ok", nil
		},
	})

	err := reader.WriteLogicalUUID(logicalUUID)
	require.NoError(t, err)
	assert.Equal(t, "hf mfu wrbl --blk 4 -d 1441674da011471aa601722b88b117f5", ranCommand)
}

func TestCLIProxmarkReader_PollEmptyBlockReturnsEmpty(t *testing.T) {
	reader := NewCLIProxmarkReader(CLIProxmarkConfig{
		Enabled: true,
		Runner: func(command string) (string, error) {
			return "Data : 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00", nil
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

func TestCLIProxmarkReader_IsAvailable(t *testing.T) {
	assert.True(t, NewCLIProxmarkReader(CLIProxmarkConfig{Enabled: true}).IsAvailable())
	assert.False(t, NewCLIProxmarkReader(CLIProxmarkConfig{Enabled: false}).IsAvailable())
}
