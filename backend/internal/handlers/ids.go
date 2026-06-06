package handlers

import (
	"errors"

	"github.com/google/uuid"
	"github.com/keweenaw-endurance/backend/internal/models"
	"github.com/keweenaw-endurance/backend/internal/services"
	"github.com/keweenaw-endurance/backend/internal/uuidutil"
	"gorm.io/gorm"
)

func (h *Handlers) resolveEventID(value string) (uuid.UUID, error) {
	return resolveEntityID(h.services.DB, &models.Event{}, value, services.ErrEventNotFound)
}

func (h *Handlers) resolveRaceID(value string) (uuid.UUID, error) {
	return resolveEntityID(h.services.DB, &models.Race{}, value, services.ErrRaceNotFound)
}

func (h *Handlers) resolveParticipantID(value string) (uuid.UUID, error) {
	return resolveEntityID(h.services.DB, &models.Participant{}, value, services.ErrParticipantNotFound)
}

func (h *Handlers) resolveCheckpointID(value string) (uuid.UUID, error) {
	return resolveEntityID(h.services.DB, &models.TimingCheckpoint{}, value, services.ErrCheckpointNotFound)
}

func (h *Handlers) resolveCategoryID(value string) (uuid.UUID, error) {
	return resolveEntityID(h.services.DB, &models.Category{}, value, services.ErrCategoryNotFound)
}

func (h *Handlers) resolveTimingRecordID(value string) (uuid.UUID, error) {
	return resolveEntityID(h.services.DB, &models.TimingRecord{}, value, services.ErrTimingRecordNotFound)
}

func resolveEntityID(db *gorm.DB, model interface{}, value string, notFound error) (uuid.UUID, error) {
	id, err := uuidutil.Resolve(db, model, value)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return uuid.Nil, notFound
	}
	return id, err
}
