package handlers

import (
	"github.com/google/uuid"
	"github.com/keweenaw-endurance/backend/internal/models"
	"github.com/keweenaw-endurance/backend/internal/uuidutil"
)

func (h *Handlers) resolveEventID(value string) (uuid.UUID, error) {
	return uuidutil.Resolve(h.services.DB, &models.Event{}, value)
}

func (h *Handlers) resolveRaceID(value string) (uuid.UUID, error) {
	return uuidutil.Resolve(h.services.DB, &models.Race{}, value)
}

func (h *Handlers) resolveParticipantID(value string) (uuid.UUID, error) {
	return uuidutil.Resolve(h.services.DB, &models.Participant{}, value)
}

func (h *Handlers) resolveCheckpointID(value string) (uuid.UUID, error) {
	return uuidutil.Resolve(h.services.DB, &models.TimingCheckpoint{}, value)
}

func (h *Handlers) resolveCategoryID(value string) (uuid.UUID, error) {
	return uuidutil.Resolve(h.services.DB, &models.Category{}, value)
}

func (h *Handlers) resolveTimingRecordID(value string) (uuid.UUID, error) {
	return uuidutil.Resolve(h.services.DB, &models.TimingRecord{}, value)
}
