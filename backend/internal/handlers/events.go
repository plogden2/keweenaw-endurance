package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/keweenaw-endurance/backend/internal/models"
	"github.com/keweenaw-endurance/backend/internal/services"
	"github.com/keweenaw-endurance/backend/internal/uuidutil"
)

func (h *Handlers) GetEvents(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	events, total, err := h.services.Events.ListEvents(page, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list events"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  events,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}

func (h *Handlers) CreateEvent(c *gin.Context) {
	var req createEventRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	eventDate, err := parseDate(req.EventDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid event_date format, use YYYY-MM-DD"})
		return
	}

	event, err := h.services.Events.CreateEvent(&models.Event{
		Name:        req.Name,
		Description: req.Description,
		EventDate:   eventDate,
		Location:    req.Location,
		WebsiteURL:  req.WebsiteURL,
		LogoURL:     req.LogoURL,
		Status:      req.Status,
	})
	if err != nil {
		respondServiceError(c, err)
		return
	}

	c.JSON(http.StatusCreated, event)
}

func (h *Handlers) GetEvent(c *gin.Context) {
	id, err := h.resolveEventID(c.Param("id"))
	if err != nil {
		respondServiceError(c, err)
		return
	}

	event, err := h.services.Events.GetEvent(id)
	if err != nil {
		respondServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, event)
}

func (h *Handlers) UpdateEvent(c *gin.Context) {
	id, err := h.resolveEventID(c.Param("id"))
	if err != nil {
		respondServiceError(c, err)
		return
	}

	var req updateEventRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	update := &models.Event{}
	if req.Name != nil {
		update.Name = *req.Name
	}
	if req.Description != nil {
		update.Description = *req.Description
	}
	if req.Location != nil {
		update.Location = *req.Location
	}
	if req.WebsiteURL != nil {
		update.WebsiteURL = *req.WebsiteURL
	}
	if req.LogoURL != nil {
		update.LogoURL = *req.LogoURL
	}
	if req.Status != nil {
		update.Status = *req.Status
	}
	if req.EventDate != nil {
		eventDate, err := parseDate(*req.EventDate)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid event_date format, use YYYY-MM-DD"})
			return
		}
		update.EventDate = eventDate
	}

	event, err := h.services.Events.UpdateEvent(id, update)
	if err != nil {
		respondServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, event)
}

func (h *Handlers) DeleteEvent(c *gin.Context) {
	id, err := h.resolveEventID(c.Param("id"))
	if err != nil {
		respondServiceError(c, err)
		return
	}

	if err := h.services.Events.DeleteEvent(id); err != nil {
		respondServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "event deleted"})
}

// GetEventLive is public GET /api/events/:id/live.
func (h *Handlers) GetEventLive(c *gin.Context) {
	id, err := h.resolveEventID(c.Param("id"))
	if err != nil {
		respondServiceError(c, err)
		return
	}

	var categoryID *uuid.UUID
	if cat := c.Query("category_id"); cat != "" {
		resolved, err := h.resolveCategoryID(cat)
		if err != nil {
			respondServiceError(c, err)
			return
		}
		categoryID = &resolved
	}

	view, err := h.services.Results.GetEventLive(id, categoryID)
	if err != nil {
		respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, view)
}

func respondServiceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, services.ErrEventNotFound),
		errors.Is(err, services.ErrRaceNotFound),
		errors.Is(err, services.ErrParticipantNotFound),
		errors.Is(err, services.ErrCheckpointNotFound),
		errors.Is(err, services.ErrCategoryNotFound),
		errors.Is(err, services.ErrTimingRecordNotFound),
		errors.Is(err, services.ErrRFIDTagNotFound),
		errors.Is(err, services.ErrStationNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	case errors.Is(err, services.ErrLiveCSVNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	case errors.Is(err, services.ErrInvalidCSV),
		errors.Is(err, services.ErrInvalidEventInput),
		errors.Is(err, services.ErrInvalidRaceInput),
		errors.Is(err, services.ErrInvalidRaceTransition),
		errors.Is(err, services.ErrInvalidParticipantInput),
		errors.Is(err, services.ErrInvalidCheckpointInput),
		errors.Is(err, services.ErrInvalidCategoryInput),
		errors.Is(err, services.ErrInvalidTimingInput),
		errors.Is(err, services.ErrInvalidRFIDInput),
		errors.Is(err, services.ErrInvalidStationInput),
		errors.Is(err, uuidutil.ErrInvalidID),
		errors.Is(err, uuidutil.ErrAmbiguousID):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case errors.Is(err, services.ErrHardwareUnavailable):
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
	}
}

