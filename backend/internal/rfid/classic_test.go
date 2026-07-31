package rfid

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClassicReadBlock1Cmd(t *testing.T) {
	assert.Equal(t, "hf mf rdbl --blk 1 -k FFFFFFFFFFFF", classicReadBlock1Cmd())
}

func TestClassicWriteBlock1Cmd(t *testing.T) {
	hex16 := "1441674da011471aa601722b88b117f5"
	assert.Equal(t,
		"hf mf wrbl --blk 1 -d 1441674da011471aa601722b88b117f5 -k FFFFFFFFFFFF",
		classicWriteBlock1Cmd(hex16),
	)
}

func TestParseClassicBlockDump_PipeRow(t *testing.T) {
	stdout := strings.Join([]string{
		"[=] Block#  | Data                           | Ascii",
		"[=] -------------------------------------------------",
		"[=] 01/0x01 | 14 41 67 4D A0 11 47 1A A6 01 72 2B 88 B1 17 F5 | .AgM..G...r+....",
	}, "\n")
	raw, err := parseClassicBlockDump(stdout)
	require.NoError(t, err)
	require.Len(t, raw, 16)
	got, err := DecodeLogicalUUID(raw)
	require.NoError(t, err)
	assert.Equal(t, "1441674d-a011-471a-a601-722b88b117f5", got)
}

func TestParseClassicBlockDump_DataLine(t *testing.T) {
	stdout := "Data : 14 41 67 4D A0 11 47 1A A6 01 72 2B 88 B1 17 F5\n"
	raw, err := parseClassicBlockDump(stdout)
	require.NoError(t, err)
	require.Len(t, raw, 16)
}

func TestParseClassicBlockDump_Empty(t *testing.T) {
	raw, err := parseClassicBlockDump("")
	require.NoError(t, err)
	assert.Nil(t, raw)
}

func TestParseClassicBlockDump_NoPayload(t *testing.T) {
	_, err := parseClassicBlockDump("[!] Auth error\n")
	require.Error(t, err)
}
