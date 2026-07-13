package handlers

import (
	"github.com/google/uuid"
	"github.com/keweenaw-endurance/backend/internal/services"
)

// Handlers exposes HTTP endpoints. Management mutations (create/update/delete,
// write-tag, CSV import when added, station PUT) require JWT auth. PIN exchange
// (POST /api/auth/pin) issues an admin-role token so existing adminOnly /
// timerWrite middleware applies. Public GETs (events, races, participants,
// live, leaderboard, results, RFID scan) remain unauthenticated.
type Handlers struct {
	services *services.Services
}

func NewHandlers(services *services.Services) *Handlers {
	return &Handlers{
		services: services,
	}
}

func (h *Handlers) refreshLiveCSV(eventID uuid.UUID) {
	if h.services == nil || h.services.CSV == nil || eventID == uuid.Nil {
		return
	}
	h.services.CSV.RefreshEvent(eventID)
}

func (h *Handlers) refreshLiveCSVForRace(raceID uuid.UUID) {
	if h.services == nil || h.services.Races == nil {
		return
	}
	race, err := h.services.Races.GetRace(raceID)
	if err != nil {
		return
	}
	h.refreshLiveCSV(race.EventID.UUID())
}
