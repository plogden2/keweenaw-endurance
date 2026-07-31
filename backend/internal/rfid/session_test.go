package rfid

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeSession struct {
	mu       sync.Mutex
	commands []string
	outputs  []string
	errs     []error
	closed   bool
	runCalls int
}

func (f *fakeSession) Run(_ context.Context, command string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.commands = append(f.commands, command)
	i := f.runCalls
	f.runCalls++
	var out string
	var err error
	if i < len(f.outputs) {
		out = f.outputs[i]
	}
	if i < len(f.errs) {
		err = f.errs[i]
	}
	return out, err
}

func (f *fakeSession) Close() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.closed = true
	return nil
}

func TestCLIProxmarkReader_PersistentSessionPollAndReconnect(t *testing.T) {
	const logicalUUID = "1441674d-a011-471a-a601-722b88b117f5"
	beep := &recordingBeeper{}
	var sessions []*fakeSession
	var mu sync.Mutex

	factory := func(ctx context.Context) (PM3Session, error) {
		mu.Lock()
		defer mu.Unlock()
		s := &fakeSession{}
		if len(sessions) == 0 {
			s.outputs = []string{
				"Thresholds set.",
				detectUltralightStdout,
				"Data : 14 41 67 4D A0 11 47 1A A6 01 72 2B 88 B1 17 F5\n",
			}
			s.errs = []error{nil, nil, nil}
		} else if len(sessions) == 1 {
			// Dead session on first Run after recreate path exercised below.
			s.outputs = []string{"Thresholds set.", ""}
			s.errs = []error{nil, errors.New("EOF")}
		} else {
			s.outputs = []string{
				"Thresholds set.",
				detectUltralightStdout,
				"Data : 14 41 67 4D A0 11 47 1A A6 01 72 2B 88 B1 17 F5\n",
			}
			s.errs = []error{nil, nil, nil}
		}
		sessions = append(sessions, s)
		return s, nil
	}

	reader := NewCLIProxmarkReader(CLIProxmarkConfig{
		Enabled:        true,
		SessionFactory: factory,
		Beeper:         beep,
	})

	got, err := reader.Poll()
	require.NoError(t, err)
	assert.Equal(t, logicalUUID, got)
	assert.Equal(t, 0, beep.calls) // tone is in bridgeapp.emitRead
	require.Len(t, sessions, 1)
	assert.Equal(t, []string{"hw sethfthresh -t 3", "hf 14a reader", proxmarkReadLogicalUUIDCmd}, sessions[0].commands)

	// Force session death on next poll.
	reader.mu.Lock()
	_ = reader.closeSessionLocked()
	reader.nextRetry = time.Time{}
	reader.mu.Unlock()

	got, err = reader.Poll()
	require.NoError(t, err)
	assert.Empty(t, got)
	require.GreaterOrEqual(t, len(sessions), 2)
	assert.True(t, sessions[1].closed)

	// During backoff, Poll should soft-fail without opening another session.
	before := len(sessions)
	got, err = reader.Poll()
	require.NoError(t, err)
	assert.Empty(t, got)
	assert.Equal(t, before, len(sessions))

	// Clear backoff and recover.
	reader.mu.Lock()
	reader.nextRetry = time.Time{}
	reader.mu.Unlock()
	got, err = reader.Poll()
	require.NoError(t, err)
	assert.Equal(t, logicalUUID, got)
	assert.Equal(t, 0, beep.calls)
}

func TestPM3PromptPattern(t *testing.T) {
	assert.True(t, pm3PromptPattern.MatchString("pm3 -->"))
	assert.True(t, pm3PromptPattern.MatchString("[usb] pm3 -->"))
	assert.True(t, pm3PromptPattern.MatchString("[usb|script] pm3 -->"))
	assert.True(t, pm3PromptPattern.MatchString("boot banner\n[usb] pm3 --> "))
	assert.False(t, pm3PromptPattern.MatchString("executing pm3 --> later"))
}

func TestCLIProxmarkReader_KeepsSessionOnCardMiss(t *testing.T) {
	var sessions []*fakeSession
	factory := func(context.Context) (PM3Session, error) {
		s := &fakeSession{
			outputs: []string{
				"Thresholds set.",
				"[#] can't select card\n[usb] pm3 -->\n",
				"[#] can't select card\n[usb] pm3 -->\n",
			},
			errs: []error{nil, errors.New("exit status 1"), errors.New("exit status 1")},
		}
		sessions = append(sessions, s)
		return s, nil
	}
	reader := NewCLIProxmarkReader(CLIProxmarkConfig{
		Enabled:        true,
		SessionFactory: factory,
		Beeper:         &recordingBeeper{},
	})

	got, err := reader.Poll()
	require.NoError(t, err)
	assert.Empty(t, got)
	require.Len(t, sessions, 1)
	assert.False(t, sessions[0].closed)
	assert.Same(t, sessions[0], reader.session)

	// Second poll reuses the warm session (no reconnect thrash).
	got, err = reader.Poll()
	require.NoError(t, err)
	assert.Empty(t, got)
	require.Len(t, sessions, 1)
	assert.Equal(t, 3, sessions[0].runCalls)
}
