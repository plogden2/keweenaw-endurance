package rfid

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCLIProxmarkReader_PollParsesFourPages(t *testing.T) {
	const logicalUUID = "1441674d-a011-471a-a601-722b88b117f5"
	combined := strings.Join([]string{
		"[=]   4 | 14 41 67 4d\n",
		"[=]   5 | a0 11 47 1a\n",
		"[=]   6 | a6 01 72 2b\n",
		"[=]   7 | 88 b1 17 f5\n",
	}, "")

	reader := NewCLIProxmarkReader(CLIProxmarkConfig{
		Enabled: true,
		Runner: func(command string) (string, error) {
			if strings.HasPrefix(command, "hw sethfthresh") {
				return "Thresholds set.", nil
			}
			if command == "hf 14a reader" {
				return detectUltralightStdout, nil
			}
			assert.Equal(t, proxmarkReadLogicalUUIDCmd, command)
			return combined, nil
		},
	})

	got, err := reader.Poll()
	require.NoError(t, err)
	assert.Equal(t, logicalUUID, got)
}

func TestCLIProxmarkReader_PollParsesChainedRRGTables(t *testing.T) {
	// Real RRG client: one table per rdbl, header contains the word "Data".
	const logicalUUID = "23657b2d-aa08-5fe8-8553-e9e3affb4678"
	combined := strings.Join([]string{
		"[=] Session log C:/Users/gener/Documents/keweenaw-endurance/backend/.proxmark3/logs/log_20260730214312.txt\n",
		"[=] Block#  | Data        | Ascii\n",
		"[=] -----------------------------\n",
		"[=] 04/0x04 | 23 65 7B 2D | #e{-\n",
		"[=] Block#  | Data        | Ascii\n",
		"[=] 05/0x05 | AA 08 5F E8 | .._.\n",
		"[=] Block#  | Data        | Ascii\n",
		"[=] 06/0x06 | 85 53 E9 E3 | .S..\n",
		"[=] Block#  | Data        | Ascii\n",
		"[=] 07/0x07 | AF FB 46 78 | ..Fx\n",
	}, "")

	reader := NewCLIProxmarkReader(CLIProxmarkConfig{
		Enabled: true,
		Runner: func(command string) (string, error) {
			if strings.HasPrefix(command, "hw sethfthresh") {
				return "Thresholds set.", nil
			}
			if command == "hf 14a reader" {
				return detectUltralightStdout, nil
			}
			return combined, nil
		},
	})

	got, err := reader.Poll()
	require.NoError(t, err)
	assert.Equal(t, logicalUUID, got)
}

func TestCLIProxmarkReader_PollParsesDataLine16Bytes(t *testing.T) {
	const logicalUUID = "1441674d-a011-471a-a601-722b88b117f5"
	beep := &recordingBeeper{}
	reader := NewCLIProxmarkReader(CLIProxmarkConfig{
		Enabled: true,
		Beeper:  beep,
		Runner: func(command string) (string, error) {
			if strings.HasPrefix(command, "hw sethfthresh") {
				return "Thresholds set.", nil
			}
			if command == "hf 14a reader" {
				return detectUltralightStdout, nil
			}
			assert.Equal(t, proxmarkReadLogicalUUIDCmd, command)
			return "Data : 14 41 67 4D A0 11 47 1A A6 01 72 2B 88 B1 17 F5\n", nil
		},
	})

	got, err := reader.Poll()
	require.NoError(t, err)
	assert.Equal(t, logicalUUID, got)
	// Tap tone moved to bridgeapp.emitRead (write-tag scores without Poll).
	assert.Equal(t, 0, beep.calls)
}

func TestCLIProxmarkReader_PollEmptyDoesNotBeep(t *testing.T) {
	beep := &recordingBeeper{}
	reader := NewCLIProxmarkReader(CLIProxmarkConfig{
		Enabled: true,
		Beeper:  beep,
		Runner: func(command string) (string, error) {
			if command == "hf 14a reader" {
				return detectUltralightStdout, nil
			}
			return "Data : 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00\n", nil
		},
	})
	got, err := reader.Poll()
	require.NoError(t, err)
	assert.Empty(t, got)
	assert.Equal(t, 0, beep.calls)
}

func TestClassifyProxmarkWriteError_MultipleTags(t *testing.T) {
	msg := classifyProxmarkWriteError("[#] Multiple tags detected. Collision after Bit 1\n", errors.New("exit"))
	assert.Contains(t, msg, "Multiple tags")
}

func TestCLIProxmarkReader_WriteBlocksLuaArmRestart(t *testing.T) {
	reader := NewCLIProxmarkReader(CLIProxmarkConfig{
		Enabled: true,
		Runner: func(command string) (string, error) {
			return "ok", nil
		},
	})
	reader.mu.Lock()
	reader.writing = true
	reader.mu.Unlock()

	err := reader.ensureLuaArm(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "write in progress")
}

func TestCLIProxmarkReader_WriteLogicalUUIDWritesFourPages(t *testing.T) {
	const logicalUUID = "1441674d-a011-471a-a601-722b88b117f5"
	var commands []string

	reader := NewCLIProxmarkReader(CLIProxmarkConfig{
		Enabled: true,
		Runner: func(command string) (string, error) {
			commands = append(commands, command)
			if strings.HasPrefix(command, "hw sethfthresh") {
				return "Thresholds set.", nil
			}
			if command == "hf 14a reader" {
				return detectUltralightStdout, nil
			}
			return "ok", nil
		},
	})

	err := reader.WriteLogicalUUID(logicalUUID)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(commands), 3)
	assert.Equal(t, "hw sethfthresh -t 3", commands[0])
	assert.Equal(t, "hf 14a reader", commands[1])
	assert.Equal(t,
		"hf mfu wrbl -b 4 -d 1441674d; hf mfu wrbl -b 5 -d a011471a; hf mfu wrbl -b 6 -d a601722b; hf mfu wrbl -b 7 -d 88b117f5",
		commands[2],
	)
}

func TestCLIProxmarkReader_WriteLogicalUUID_RetriesTransientMultipleTagsOnDetect(t *testing.T) {
	const logicalUUID = "1441674d-a011-471a-a601-722b88b117f5"
	var nDetect, nWrite int

	reader := NewCLIProxmarkReader(CLIProxmarkConfig{
		Enabled: true,
		Runner: func(command string) (string, error) {
			if strings.HasPrefix(command, "hw sethfthresh") {
				return "Thresholds set.", nil
			}
			if command == "hf 14a reader" {
				nDetect++
				if nDetect == 1 {
					return "[#] Multiple tags detected. Collision after Bit 1\n", errors.New("exit")
				}
				return detectUltralightStdout, nil
			}
			if strings.Contains(command, "wrbl") {
				nWrite++
				return "ok", nil
			}
			return "ok", nil
		},
	})

	err := reader.WriteLogicalUUID(logicalUUID)
	require.NoError(t, err)
	assert.Equal(t, 2, nDetect)
	assert.Equal(t, 1, nWrite)
}

func TestCLIProxmarkReader_WriteLogicalUUID_RetriesTransientMultipleTagsOnWrite(t *testing.T) {
	const logicalUUID = "1441674d-a011-471a-a601-722b88b117f5"
	var nWrite int

	reader := NewCLIProxmarkReader(CLIProxmarkConfig{
		Enabled: true,
		Runner: func(command string) (string, error) {
			if strings.HasPrefix(command, "hw sethfthresh") {
				return "Thresholds set.", nil
			}
			if command == "hf 14a reader" {
				return detectUltralightStdout, nil
			}
			if strings.Contains(command, "wrbl") {
				nWrite++
				if nWrite == 1 {
					return "[#] Multiple tags detected. Collision after Bit 1\n", errors.New("exit")
				}
				return "ok", nil
			}
			return "ok", nil
		},
	})

	err := reader.WriteLogicalUUID(logicalUUID)
	require.NoError(t, err)
	assert.Equal(t, 2, nWrite)
}

func TestCLIProxmarkReader_PollEmptyPagesReturnsEmpty(t *testing.T) {
	reader := NewCLIProxmarkReader(CLIProxmarkConfig{
		Enabled: true,
		Runner: func(command string) (string, error) {
			if command == "hf 14a reader" {
				return detectUltralightStdout, nil
			}
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

	got, err := reader.Poll()
	require.NoError(t, err)
	assert.Empty(t, got)
}

func TestCLIProxmarkReader_PollParseFailure(t *testing.T) {
	reader := NewCLIProxmarkReader(CLIProxmarkConfig{
		Enabled: true,
		Runner: func(command string) (string, error) {
			if command == "hf 14a reader" {
				return detectUltralightStdout, nil
			}
			return "tag present but no hex dump", nil
		},
	})

	got, err := reader.Poll()
	require.NoError(t, err)
	assert.Empty(t, got) // soft-fail empty tick so the poll loop keeps running
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

func TestParseReadPageOutput_HexBlockLabel(t *testing.T) {
	// Real pm3 output uses 04/0x04 — must not treat the label hex as data bytes.
	raw, err := parseReadPageOutput("[=] 04/0x04 | 11 22 33 44 | .\"3D\n", 4)
	require.NoError(t, err)
	require.Equal(t, []byte{0x11, 0x22, 0x33, 0x44}, raw)
}

func TestCLIProxmarkReader_DetectISO14443A(t *testing.T) {
	reader := NewCLIProxmarkReader(CLIProxmarkConfig{
		Enabled: true,
		Runner: func(command string) (string, error) {
			if strings.HasPrefix(command, "hw sethfthresh") {
				return "Thresholds set.", nil
			}
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
	// Guard against regressing Ultralight to invalid --blk / 16-byte single-page writes.
	reader := NewCLIProxmarkReader(CLIProxmarkConfig{
		Enabled: true,
		Runner: func(command string) (string, error) {
			if strings.HasPrefix(command, "hw sethfthresh") {
				return "Thresholds set.", nil
			}
			if command == "hf 14a reader" {
				return detectUltralightStdout, nil
			}
			assert.NotContains(t, command, "--blk")
			assert.Contains(t, command, "-b ")
			for _, part := range strings.Split(command, ";") {
				part = strings.TrimSpace(part)
				assert.True(t, strings.HasPrefix(part, "hf mfu wrbl "), part)
				parts := strings.Fields(part)
				require.GreaterOrEqual(t, len(parts), 6)
				dIdx := -1
				for i, p := range parts {
					if p == "-d" {
						dIdx = i
						break
					}
				}
				require.Greater(t, dIdx, 0)
				require.Equal(t, 8, len(parts[dIdx+1]), "page writes must be 4 bytes (8 hex chars): %s", part)
			}
			return "ok", nil
		},
	})
	require.NoError(t, reader.WriteLogicalUUID("550e8400-e29b-41d4-a716-446655440099"))
}

func TestCLIProxmarkReader_PollPageCommandFormat(t *testing.T) {
	var cmds []string
	reader := NewCLIProxmarkReader(CLIProxmarkConfig{
		Enabled: true,
		Runner: func(command string) (string, error) {
			cmds = append(cmds, command)
			if strings.HasPrefix(command, "hw sethfthresh") {
				return "Thresholds set.", nil
			}
			if command == "hf 14a reader" {
				return detectUltralightStdout, nil
			}
			return "Data : 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00\n", nil
		},
	})
	_, err := reader.Poll()
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(cmds), 3)
	assert.Equal(t, "hw sethfthresh -t 3", cmds[0])
	assert.Equal(t, "hf 14a reader", cmds[1])
	assert.Equal(t, proxmarkReadLogicalUUIDCmd, cmds[2])
}

func TestCLIProxmarkReader_AppliesHFThreshBeforePoll(t *testing.T) {
	var cmds []string
	r := NewCLIProxmarkReader(CLIProxmarkConfig{
		Enabled: true,
		HFGain:  63,
		Runner: func(command string) (string, error) {
			cmds = append(cmds, command)
			if strings.HasPrefix(command, "hw sethfthresh") {
				return "Thresholds set.", nil
			}
			if command == "hf 14a reader" {
				return detectUltralightStdout, nil
			}
			return "Data : 14 41 67 4D A0 11 47 1A A6 01 72 2B 88 B1 17 F5\n", nil
		},
	})
	_, err := r.Poll()
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(cmds), 3)
	assert.Equal(t, "hw sethfthresh -t 3", cmds[0])
	assert.Equal(t, "hf 14a reader", cmds[1])
	assert.Equal(t, proxmarkReadLogicalUUIDCmd, cmds[2])
}

func TestCLIProxmarkReader_SetHFGainReapplies(t *testing.T) {
	var cmds []string
	r := NewCLIProxmarkReader(CLIProxmarkConfig{
		Enabled: true,
		HFGain:  63,
		Runner: func(command string) (string, error) {
			cmds = append(cmds, command)
			if strings.HasPrefix(command, "hw sethfthresh") {
				return "Thresholds set.", nil
			}
			if command == "hf 14a reader" {
				return detectUltralightStdout, nil
			}
			return "Data : 14 41 67 4D A0 11 47 1A A6 01 72 2B 88 B1 17 F5\n", nil
		},
	})
	_, err := r.Poll()
	require.NoError(t, err)

	r.SetHFGain(50)
	_, err = r.Poll()
	require.NoError(t, err)

	var threshCmds []string
	for _, c := range cmds {
		if strings.HasPrefix(c, "hw sethfthresh") {
			threshCmds = append(threshCmds, c)
		}
	}
	require.Len(t, threshCmds, 2)
	assert.Equal(t, "hw sethfthresh -t 3", threshCmds[0])
	assert.Equal(t, "hw sethfthresh -t 14", threshCmds[1])
}

func TestCLIProxmarkReader_ArmScanUsesWaitForCard(t *testing.T) {
	const logicalUUID = "23657b2d-aa08-5fe8-8553-e9e3affb4678"
	var commands []string
	reader := NewCLIProxmarkReader(CLIProxmarkConfig{
		Enabled: true,
		HFGain:  57,
		Runner: func(command string) (string, error) {
			commands = append(commands, command)
			if strings.Contains(command, "hf 14a reader -w") {
				return detectUltralightStdout, nil
			}
			return strings.Join([]string{
				"[=] 04/0x04 | 23 65 7B 2D | #e{-\n",
				"[=] 05/0x05 | AA 08 5F E8 | .._.\n",
				"[=] 06/0x06 | 85 53 E9 E3 | .S..\n",
				"[=] 07/0x07 | AF FB 46 78 | ..Fx\n",
			}, ""), nil
		},
	})

	got, err := reader.ArmScan(context.Background())
	require.NoError(t, err)
	assert.Equal(t, logicalUUID, got)
	require.GreaterOrEqual(t, len(commands), 2)
	assert.Contains(t, commands[0], "hf 14a reader -w --skip")
	assert.Contains(t, commands[0], "hw sethfthresh")
	assert.Equal(t, proxmarkReadLogicalUUIDCmd, commands[1])
}

func TestParseRaw14aRead16(t *testing.T) {
	stdout := strings.Join([]string{
		"[+]  UID: 04 26 98 02 47 20 91",
		"[+] 23 65 7B 2D AA 08 5F E8 85 53 E9 E3 AF FB 46 78 [ 4B A1 ]",
		"KEWEENAW_TAP_END",
	}, "\n")
	raw, err := parseLogicalUUIDBytes(stdout)
	require.NoError(t, err)
	uid, err := DecodeLogicalUUID(raw)
	require.NoError(t, err)
	assert.Equal(t, "23657b2d-aa08-5fe8-8553-e9e3affb4678", uid)
}

const (
	detectUltralightStdout = "[+] UID: 04 12 34 56 78 9A 80\n[+] ATQA: 00 44\n[+] SAK: 00\n"
	detectClassicStdout    = "[+] UID: 11 22 33 44\n[+] ATQA: 00 04\n[+] SAK: 08\n"
	detectDESFireStdout    = "[+] UID: AA BB CC DD\n[+] ATQA: 03 44\n[+] SAK: 20\n"
)

func TestCLIProxmarkReader_WriteLogicalUUID_DispatchesClassic(t *testing.T) {
	const logicalUUID = "1441674d-a011-471a-a601-722b88b117f5"
	var commands []string
	reader := NewCLIProxmarkReader(CLIProxmarkConfig{
		Enabled: true,
		Runner: func(command string) (string, error) {
			commands = append(commands, command)
			if strings.HasPrefix(command, "hw sethfthresh") {
				return "Thresholds set.", nil
			}
			if command == "hf 14a reader" {
				return detectClassicStdout, nil
			}
			if strings.HasPrefix(command, "hf mf wrbl") {
				return "ok", nil
			}
			if command == classicReadBlock1Cmd() {
				return "[=] 01/0x01 | 14 41 67 4D A0 11 47 1A A6 01 72 2B 88 B1 17 F5 | ....\n", nil
			}
			return "ok", nil
		},
	})

	err := reader.WriteLogicalUUID(logicalUUID)
	require.NoError(t, err)
	var writeCmd string
	for _, c := range commands {
		if strings.HasPrefix(c, "hf mf wrbl") {
			writeCmd = c
			break
		}
	}
	require.NotEmpty(t, writeCmd)
	assert.Equal(t, classicWriteBlock1Cmd("1441674da011471aa601722b88b117f5"), writeCmd)
	assert.NotContains(t, writeCmd, "hf mfu")
}

func TestCLIProxmarkReader_WriteLogicalUUID_DispatchesUltralight(t *testing.T) {
	const logicalUUID = "1441674d-a011-471a-a601-722b88b117f5"
	var commands []string
	reader := NewCLIProxmarkReader(CLIProxmarkConfig{
		Enabled: true,
		Runner: func(command string) (string, error) {
			commands = append(commands, command)
			if strings.HasPrefix(command, "hw sethfthresh") {
				return "Thresholds set.", nil
			}
			if command == "hf 14a reader" {
				return detectUltralightStdout, nil
			}
			return "ok", nil
		},
	})

	require.NoError(t, reader.WriteLogicalUUID(logicalUUID))
	var writeCmd string
	for _, c := range commands {
		if strings.Contains(c, "hf mfu wrbl") {
			writeCmd = c
			break
		}
	}
	require.NotEmpty(t, writeCmd)
	assert.Contains(t, writeCmd, "hf mfu wrbl -b 4")
	assert.NotContains(t, writeCmd, "hf mf wrbl")
}

func TestCLIProxmarkReader_WriteLogicalUUID_UnsupportedTag(t *testing.T) {
	var unexpected []string
	reader := NewCLIProxmarkReader(CLIProxmarkConfig{
		Enabled: true,
		Runner: func(command string) (string, error) {
			if strings.HasPrefix(command, "hw sethfthresh") {
				return "Thresholds set.", nil
			}
			if command == "hf 14a reader" {
				return detectDESFireStdout, nil
			}
			unexpected = append(unexpected, command)
			return "", errors.New("should not write unsupported tag")
		},
	})

	err := reader.WriteLogicalUUID("1441674d-a011-471a-a601-722b88b117f5")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported tag type")
	assert.Contains(t, err.Error(), "20")
	assert.NotContains(t, err.Error(), "BCC")
	assert.Empty(t, unexpected)
}

func TestCLIProxmarkReader_Poll_DispatchesClassic(t *testing.T) {
	const logicalUUID = "1441674d-a011-471a-a601-722b88b117f5"
	var commands []string
	reader := NewCLIProxmarkReader(CLIProxmarkConfig{
		Enabled: true,
		Runner: func(command string) (string, error) {
			commands = append(commands, command)
			if strings.HasPrefix(command, "hw sethfthresh") {
				return "Thresholds set.", nil
			}
			if command == "hf 14a reader" {
				return detectClassicStdout, nil
			}
			if command == classicReadBlock1Cmd() {
				return "[=] 01/0x01 | 14 41 67 4D A0 11 47 1A A6 01 72 2B 88 B1 17 F5 | ....\n", nil
			}
			t.Fatalf("unexpected poll command: %s", command)
			return "", nil
		},
	})

	got, err := reader.Poll()
	require.NoError(t, err)
	assert.Equal(t, logicalUUID, got)
	assert.Contains(t, commands, classicReadBlock1Cmd())
	assert.NotContains(t, commands, proxmarkReadLogicalUUIDCmd)
}

func TestCLIProxmarkReader_Poll_DispatchesUltralight(t *testing.T) {
	const logicalUUID = "1441674d-a011-471a-a601-722b88b117f5"
	reader := NewCLIProxmarkReader(CLIProxmarkConfig{
		Enabled: true,
		Runner: func(command string) (string, error) {
			if strings.HasPrefix(command, "hw sethfthresh") {
				return "Thresholds set.", nil
			}
			if command == "hf 14a reader" {
				return detectUltralightStdout, nil
			}
			assert.Equal(t, proxmarkReadLogicalUUIDCmd, command)
			return "Data : 14 41 67 4D A0 11 47 1A A6 01 72 2B 88 B1 17 F5\n", nil
		},
	})

	got, err := reader.Poll()
	require.NoError(t, err)
	assert.Equal(t, logicalUUID, got)
}

func TestCLIProxmarkReader_ArmScan_DispatchesClassicAfterWait(t *testing.T) {
	const logicalUUID = "1441674d-a011-471a-a601-722b88b117f5"
	var commands []string
	reader := NewCLIProxmarkReader(CLIProxmarkConfig{
		Enabled: true,
		HFGain:  57,
		Runner: func(command string) (string, error) {
			commands = append(commands, command)
			if strings.Contains(command, "hf 14a reader -w") {
				return detectClassicStdout, nil
			}
			if command == classicReadBlock1Cmd() {
				return "[=] 01/0x01 | 14 41 67 4D A0 11 47 1A A6 01 72 2B 88 B1 17 F5 | ....\n", nil
			}
			return "", fmt.Errorf("unexpected: %s", command)
		},
	})

	got, err := reader.ArmScan(context.Background())
	require.NoError(t, err)
	assert.Equal(t, logicalUUID, got)
	require.GreaterOrEqual(t, len(commands), 2)
	assert.Contains(t, commands[0], "hf 14a reader -w --skip")
	assert.Equal(t, classicReadBlock1Cmd(), commands[1])
}

func TestParsePollUUID_ClassicBlockDump(t *testing.T) {
	stdout := detectClassicStdout + "[=] 01/0x01 | 14 41 67 4D A0 11 47 1A A6 01 72 2B 88 B1 17 F5 | ....\n"
	got, err := parsePollUUID(stdout)
	require.NoError(t, err)
	assert.Equal(t, "1441674d-a011-471a-a601-722b88b117f5", got)
}

func TestParsePollUUID_FallsBackWhenFamilyReadFails(t *testing.T) {
	// Continuous arm always runs both family reads. Mis-detected Classic SAK
	// must still accept a valid Ultralight Type-2 payload.
	stdout := strings.Join([]string{
		detectClassicStdout,
		"[!] Auth error",
		"[+] 23 65 7B 2D AA 08 5F E8 85 53 E9 E3 AF FB 46 78 [ 4B A1 ]",
		"KEWEENAW_TAP_END",
	}, "\n")
	got, err := parsePollUUID(stdout)
	require.NoError(t, err)
	assert.Equal(t, "23657b2d-aa08-5fe8-8553-e9e3affb4678", got)
}
