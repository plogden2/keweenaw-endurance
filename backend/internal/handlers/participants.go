package handlers

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/keweenaw-endurance/backend/internal/models"
	"github.com/keweenaw-endurance/backend/internal/uuidutil"
)

func (h *Handlers) GetParticipants(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	q := c.Query("q")

	var raceID *uuid.UUID
	if raceIDStr := c.Query("race_id"); raceIDStr != "" {
		id, err := h.resolveRaceID(raceIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid race_id"})
			return
		}
		raceID = &id
	}

	participants, total, err := h.services.Participants.ListParticipants(page, limit, raceID, q)
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

// GetRaceParticipants lists participants for a race (contract: GET /api/races/:id/participants?q=).
func (h *Handlers) GetRaceParticipants(c *gin.Context) {
	raceID, err := h.resolveRaceID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid race id"})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "200"))
	q := c.Query("q")

	participants, total, err := h.services.Participants.ListParticipants(page, limit, &raceID, q)
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
	if req.RaceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "race_id is required"})
		return
	}

	raceID, err := h.resolveRaceID(req.RaceID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid race_id"})
		return
	}

	participant, err := h.createParticipantFromRequest(raceID, req)
	if err != nil {
		respondServiceError(c, err)
		return
	}

	h.refreshLiveCSVForRace(raceID)
	c.JSON(http.StatusCreated, participant)
}

// CreateRaceParticipant creates a participant under a race (bib optional → sequential).
func (h *Handlers) CreateRaceParticipant(c *gin.Context) {
	raceID, err := h.resolveRaceID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid race id"})
		return
	}

	var req createParticipantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.CategoryID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "category_id is required"})
		return
	}

	participant, err := h.createParticipantFromRequest(raceID, req)
	if err != nil {
		respondServiceError(c, err)
		return
	}

	h.refreshLiveCSVForRace(raceID)
	c.JSON(http.StatusCreated, participant)
}

func (h *Handlers) createParticipantFromRequest(raceID uuid.UUID, req createParticipantRequest) (*models.Participant, error) {
	input := &models.Participant{
		RaceID:     uuidutil.NewPublicUUID(raceID),
		BibNumber:  req.BibNumber,
		FirstName:  req.FirstName,
		LastName:   req.LastName,
		Gender:     req.Gender,
		Age:        req.Age,
		Location:   req.Location,
		RFIDTagUID: req.RFIDTagUID,
		Status:     req.Status,
	}
	if req.CategoryID != "" {
		catID, err := h.resolveCategoryID(req.CategoryID)
		if err != nil {
			return nil, err
		}
		pub := uuidutil.NewPublicUUID(catID)
		input.CategoryID = &pub
	}
	return h.services.Participants.CreateParticipant(input)
}

func (h *Handlers) GetParticipant(c *gin.Context) {
	id, err := h.resolveParticipantID(c.Param("id"))
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
	id, err := h.resolveParticipantID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid participant id"})
		return
	}

	var req updateParticipantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	update := &models.Participant{}
	if req.BibNumber != nil {
		update.BibNumber = *req.BibNumber
	}
	if req.FirstName != nil {
		update.FirstName = *req.FirstName
	}
	if req.LastName != nil {
		update.LastName = *req.LastName
	}
	if req.Gender != nil {
		update.Gender = *req.Gender
	}
	if req.Age != nil {
		update.Age = *req.Age
	}
	if req.Location != nil {
		update.Location = *req.Location
	}
	if req.RFIDTagUID != nil {
		update.RFIDTagUID = *req.RFIDTagUID
	}
	if req.Status != nil {
		update.Status = *req.Status
	}
	if req.CategoryID != nil {
		if *req.CategoryID == "" {
			zero := uuidutil.PublicUUID{}
			update.CategoryID = &zero
		} else {
			catID, err := h.resolveCategoryID(*req.CategoryID)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid category_id"})
				return
			}
			pub := uuidutil.NewPublicUUID(catID)
			update.CategoryID = &pub
		}
	}

	participant, err := h.services.Participants.UpdateParticipant(id, update)
	if err != nil {
		respondServiceError(c, err)
		return
	}

	h.refreshLiveCSVForRace(participant.RaceID.UUID())
	c.JSON(http.StatusOK, participant)
}

func (h *Handlers) DeleteParticipant(c *gin.Context) {
	id, err := h.resolveParticipantID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid participant id"})
		return
	}

	existing, _ := h.services.Participants.GetParticipant(id)
	if err := h.services.Participants.DeleteParticipant(id); err != nil {
		respondServiceError(c, err)
		return
	}

	if existing != nil {
		h.refreshLiveCSVForRace(existing.RaceID.UUID())
	}
	c.JSON(http.StatusOK, gin.H{"message": "participant deleted"})
}

// GetParticipantTags handles GET /api/races/:id/participants/:participantId/tags
func (h *Handlers) GetParticipantTags(c *gin.Context) {
	if err := h.ensureParticipantInRace(c); err != nil {
		return
	}
	participantID, err := h.resolveParticipantID(c.Param("participantId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid participant id"})
		return
	}

	tags, err := h.services.RFID.ListParticipantTags(participantID)
	if err != nil {
		respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": tags})
}

// PostParticipantTag handles POST /api/races/:id/participants/:participantId/tags
func (h *Handlers) PostParticipantTag(c *gin.Context) {
	if err := h.ensureParticipantInRace(c); err != nil {
		return
	}
	participantID, err := h.resolveParticipantID(c.Param("participantId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid participant id"})
		return
	}

	var req participantTagRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Prefer hardware write when available; fall back to association-only for typed UIDs.
	participant, err := h.services.RFID.WriteTag(participantID, req.TagUID)
	if err != nil {
		assoc, assocErr := h.services.RFID.AssociateTag(participantID, req.TagUID)
		if assocErr != nil {
			respondServiceError(c, err)
			return
		}
		raceID, _ := h.resolveRaceID(c.Param("id"))
		h.refreshLiveCSVForRace(raceID)
		c.JSON(http.StatusCreated, assoc)
		return
	}
	h.refreshLiveCSVForRace(participant.RaceID.UUID())
	c.JSON(http.StatusCreated, gin.H{
		"tag_uid":        req.TagUID,
		"participant_id": participant.ID,
		"tag_uids":       participant.TagUIDs,
	})
}

func (h *Handlers) ensureParticipantInRace(c *gin.Context) error {
	raceID, err := h.resolveRaceID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid race id"})
		return err
	}
	participantID, err := h.resolveParticipantID(c.Param("participantId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid participant id"})
		return err
	}
	participant, err := h.services.Participants.GetParticipant(participantID)
	if err != nil {
		respondServiceError(c, err)
		return err
	}
	if participant.RaceID.UUID() != raceID {
		err := fmt.Errorf("participant does not belong to race")
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return err
	}
	return nil
}
