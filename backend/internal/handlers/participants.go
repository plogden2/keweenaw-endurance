package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/keweenaw-endurance/backend/internal/models"
)

type createParticipantRequest struct {
	RaceID     string `json:"race_id" binding:"required"`
	BibNumber  string `json:"bib_number" binding:"required"`
	FirstName  string `json:"first_name" binding:"required"`
	LastName   string `json:"last_name" binding:"required"`
	Gender     string `json:"gender"`
	Age        int    `json:"age"`
	RFIDTagUID string `json:"rfid_tag_uid"`
	Status     string `json:"status"`
}

func (h *Handlers) GetParticipants(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	var raceID *uuid.UUID
	if raceIDStr := c.Query("race_id"); raceIDStr != "" {
		id, err := parseUUID(raceIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid race_id"})
			return
		}
		raceID = &id
	}

	participants, total, err := h.services.Participants.ListParticipants(page, limit, raceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list participants"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  participants,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}

func (h *Handlers) CreateParticipant(c *gin.Context) {
	var req createParticipantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	raceID, err := parseUUID(req.RaceID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid race_id"})
		return
	}

	participant, err := h.services.Participants.CreateParticipant(&models.Participant{
		RaceID:     raceID,
		BibNumber:  req.BibNumber,
		FirstName:  req.FirstName,
		LastName:   req.LastName,
		Gender:     req.Gender,
		Age:        req.Age,
		RFIDTagUID: req.RFIDTagUID,
		Status:     req.Status,
	})
	if err != nil {
		respondServiceError(c, err)
		return
	}

	c.JSON(http.StatusCreated, participant)
}

func (h *Handlers) GetParticipant(c *gin.Context) {
	id, err := parseUUID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid participant id"})
		return
	}

	participant, err := h.services.Participants.GetParticipant(id)
	if err != nil {
		respondServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, participant)
}

func (h *Handlers) UpdateParticipant(c *gin.Context) {
	id, err := parseUUID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid participant id"})
		return
	}

	var req createParticipantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	participant, err := h.services.Participants.UpdateParticipant(id, &models.Participant{
		BibNumber:  req.BibNumber,
		FirstName:  req.FirstName,
		LastName:   req.LastName,
		Gender:     req.Gender,
		Age:        req.Age,
		RFIDTagUID: req.RFIDTagUID,
		Status:     req.Status,
	})
	if err != nil {
		respondServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, participant)
}

func (h *Handlers) DeleteParticipant(c *gin.Context) {
	id, err := parseUUID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid participant id"})
		return
	}

	if err := h.services.Participants.DeleteParticipant(id); err != nil {
		respondServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "participant deleted"})
}
