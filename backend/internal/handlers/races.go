package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/keweenaw-endurance/backend/internal/models"
)

type createRaceRequest struct {
	EventID         string  `json:"event_id" binding:"required"`
	Name            string  `json:"name" binding:"required"`
	RaceType        string  `json:"race_type" binding:"required"`
	DistanceKm      float64 `json:"distance_km"`
	DurationMinutes int     `json:"duration_minutes"`
	StartTime       string  `json:"start_time"`
	Status          string  `json:"status"`
}

func (h *Handlers) GetRaces(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	var eventID *uuid.UUID
	if eventIDStr := c.Query("event_id"); eventIDStr != "" {
		id, err := parseUUID(eventIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid event_id"})
			return
		}
		eventID = &id
	}

	races, total, err := h.services.Races.ListRaces(page, limit, eventID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list races"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  races,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}

func (h *Handlers) CreateRace(c *gin.Context) {
	var req createRaceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	eventID, err := parseUUID(req.EventID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid event_id"})
		return
	}

	race := &models.Race{
		EventID:         eventID,
		Name:            req.Name,
		RaceType:        req.RaceType,
		DistanceKm:      req.DistanceKm,
		DurationMinutes: req.DurationMinutes,
		Status:          req.Status,
	}
	if req.StartTime != "" {
		startTime, err := parseTimestamp(req.StartTime)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid start_time format, use RFC3339"})
			return
		}
		race.StartTime = startTime
	}

	created, err := h.services.Races.CreateRace(race)
	if err != nil {
		respondServiceError(c, err)
		return
	}

	c.JSON(http.StatusCreated, created)
}

func (h *Handlers) GetRace(c *gin.Context) {
	id, err := parseUUID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid race id"})
		return
	}

	race, err := h.services.Races.GetRace(id)
	if err != nil {
		respondServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, race)
}

func (h *Handlers) UpdateRace(c *gin.Context) {
	id, err := parseUUID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid race id"})
		return
	}

	var req createRaceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	update := &models.Race{
		Name:            req.Name,
		RaceType:        req.RaceType,
		DistanceKm:      req.DistanceKm,
		DurationMinutes: req.DurationMinutes,
		Status:          req.Status,
	}
	if req.StartTime != "" {
		startTime, err := parseTimestamp(req.StartTime)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid start_time format, use RFC3339"})
			return
		}
		update.StartTime = startTime
	}

	race, err := h.services.Races.UpdateRace(id, update)
	if err != nil {
		respondServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, race)
}

func (h *Handlers) DeleteRace(c *gin.Context) {
	id, err := parseUUID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid race id"})
		return
	}

	if err := h.services.Races.DeleteRace(id); err != nil {
		respondServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "race deleted"})
}

func parseDate(value string) (time.Time, error) {
	return time.Parse("2006-01-02", value)
}

func parseTimestamp(value string) (time.Time, error) {
	return time.Parse(time.RFC3339, value)
}
