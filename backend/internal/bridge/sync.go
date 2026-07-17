package bridge

import (
	"fmt"
)

// ReadSender delivers a pending lap to the hosted bridge WebSocket.
type ReadSender interface {
	SendRead(lap PendingLap) error
}

// Syncer flushes the durable pending queue to hosted.
type Syncer struct {
	store *LocalStore
}

// NewSyncer creates a sync helper for a local store.
func NewSyncer(store *LocalStore) *Syncer {
	return &Syncer{store: store}
}

// Flush sends each pending lap via sender, removing successfully sent laps.
// Returns the number of laps flushed. Re-running after a successful flush is a no-op.
func (s *Syncer) Flush(sender ReadSender) (int, error) {
	if s == nil || s.store == nil {
		return 0, fmt.Errorf("syncer not configured")
	}
	if sender == nil {
		return 0, fmt.Errorf("read sender is required")
	}

	pending, err := s.store.ListPending()
	if err != nil {
		return 0, err
	}
	if len(pending) == 0 {
		return 0, nil
	}

	flushed := 0
	for _, lap := range pending {
		if err := sender.SendRead(lap); err != nil {
			return flushed, fmt.Errorf("flush lap %s: %w", lap.ID, err)
		}
		if err := s.store.RemovePending(lap.ID); err != nil {
			return flushed, fmt.Errorf("remove pending lap %s: %w", lap.ID, err)
		}
		flushed++
	}
	return flushed, nil
}
