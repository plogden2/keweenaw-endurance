package services

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestLiveStreamHub_PublishDeliversToSubscriber(t *testing.T) {
	hub := NewLiveStreamHub()
	eventID := uuid.New()
	ch, cancel := hub.Subscribe(eventID, 4)
	defer cancel()

	ev := LapRecordedEvent{
		Type:            "lap_recorded",
		EventID:         eventID.String(),
		RaceID:          uuid.New().String(),
		ParticipantID:   uuid.New().String(),
		ParticipantName: "Alex Rivera",
		BibNumber:       "42",
		LapCount:        7,
		RecordedAt:      time.Now().UTC(),
	}
	hub.Publish(eventID, ev)

	select {
	case got := <-ch:
		require.Equal(t, "lap_recorded", got.Type)
		require.Equal(t, "Alex Rivera", got.ParticipantName)
		require.Equal(t, 7, got.LapCount)
	case <-time.After(time.Second):
		t.Fatal("expected event")
	}
}

func TestLiveStreamHub_DoesNotDeliverToOtherEvent(t *testing.T) {
	hub := NewLiveStreamHub()
	a, b := uuid.New(), uuid.New()
	ch, cancel := hub.Subscribe(a, 4)
	defer cancel()
	hub.Publish(b, LapRecordedEvent{Type: "lap_recorded", EventID: b.String()})

	select {
	case <-ch:
		t.Fatal("should not receive other event")
	case <-time.After(50 * time.Millisecond):
	}
}

func TestLiveStreamHub_DropsWhenSubscriberSlow(t *testing.T) {
	hub := NewLiveStreamHub()
	eventID := uuid.New()
	ch, cancel := hub.Subscribe(eventID, 1)
	defer cancel()

	ev1 := LapRecordedEvent{Type: "lap_recorded", EventID: eventID.String(), LapCount: 1}
	ev2 := LapRecordedEvent{Type: "lap_recorded", EventID: eventID.String(), LapCount: 2}

	hub.Publish(eventID, ev1)

	done := make(chan struct{})
	go func() {
		hub.Publish(eventID, ev2)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Publish blocked on slow subscriber")
	}

	select {
	case got := <-ch:
		require.Equal(t, 1, got.LapCount)
	case <-time.After(time.Second):
		t.Fatal("expected to read buffered event")
	}
}
