package services

import (
	"sync"
	"time"

	"github.com/google/uuid"
)

// LapRecordedEvent is the public live-stream payload for spectator celebrations.
type LapRecordedEvent struct {
	Type            string    `json:"type"`
	EventID         string    `json:"event_id"`
	RaceID          string    `json:"race_id"`
	ParticipantID   string    `json:"participant_id"`
	ParticipantName string    `json:"participant_name"`
	BibNumber       string    `json:"bib_number,omitempty"`
	LapCount        int       `json:"lap_count"`
	RecordedAt      time.Time `json:"recorded_at"`
}

// LiveStreamHub fans out lap_recorded events to per-event WebSocket subscribers.
type LiveStreamHub struct {
	mu   sync.Mutex
	subs map[uuid.UUID][]chan LapRecordedEvent
}

func NewLiveStreamHub() *LiveStreamHub {
	return &LiveStreamHub{subs: make(map[uuid.UUID][]chan LapRecordedEvent)}
}

func (h *LiveStreamHub) Subscribe(eventID uuid.UUID, buffer int) (<-chan LapRecordedEvent, func()) {
	if buffer < 1 {
		buffer = 8
	}
	ch := make(chan LapRecordedEvent, buffer)
	h.mu.Lock()
	h.subs[eventID] = append(h.subs[eventID], ch)
	h.mu.Unlock()
	cancel := func() {
		h.unsubscribe(eventID, ch)
	}
	return ch, cancel
}

func (h *LiveStreamHub) unsubscribe(eventID uuid.UUID, ch chan LapRecordedEvent) {
	h.mu.Lock()
	defer h.mu.Unlock()
	list := h.subs[eventID]
	out := list[:0]
	for _, c := range list {
		if c != ch {
			out = append(out, c)
		}
	}
	if len(out) == 0 {
		delete(h.subs, eventID)
	} else {
		h.subs[eventID] = out
	}
}

func (h *LiveStreamHub) Publish(eventID uuid.UUID, ev LapRecordedEvent) {
	if ev.Type == "" {
		ev.Type = "lap_recorded"
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	for _, ch := range h.subs[eventID] {
		select {
		case ch <- ev:
		default:
			// drop if slow — celebration is best-effort
		}
	}
}
