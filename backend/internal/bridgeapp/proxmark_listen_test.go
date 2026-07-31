package bridgeapp

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListenProxmarkLoop_BeepsAndReportsTaps(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	uids := []string{
		"aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee",
		"aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee",
		"11111111-2222-3333-4444-555555555555",
	}
	idx := 0
	armScan := func(ctx context.Context) (string, error) {
		if idx >= len(uids) {
			cancel()
			<-ctx.Done()
			return "", ctx.Err()
		}
		u := uids[idx]
		idx++
		return u, nil
	}

	var beeps int
	var got []string
	var mu sync.Mutex
	err := listenProxmarkLoop(ctx, armScan, func() { beeps++ }, func(uid string) {
		mu.Lock()
		got = append(got, uid)
		mu.Unlock()
	}, 0)
	require.NoError(t, err)

	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, uids, got)
	assert.Equal(t, 3, beeps)
}

func TestListenProxmarkLoop_DebouncesSameUUID(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	n := 0
	armScan := func(ctx context.Context) (string, error) {
		n++
		if n > 2 {
			cancel()
			<-ctx.Done()
			return "", ctx.Err()
		}
		return "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee", nil
	}

	var beeps int
	var got []string
	err := listenProxmarkLoop(ctx, armScan, func() { beeps++ }, func(uid string) {
		got = append(got, uid)
	}, time.Second)
	require.NoError(t, err)
	assert.Equal(t, []string{"aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"}, got)
	assert.Equal(t, 1, beeps)
}

func TestListenProxmark_RequiresHardware(t *testing.T) {
	err := ListenProxmark(context.Background(), Config{RFIDHardware: false}, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "RFID_HARDWARE")

	err = ListenProxmark(context.Background(), Config{RFIDHardware: true, BridgeMock: true}, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "mock")
}
