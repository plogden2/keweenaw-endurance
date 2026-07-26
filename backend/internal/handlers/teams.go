package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/keweenaw-endurance/backend/internal/models"
	"github.com/keweenaw-endurance/backend/internal/uuidutil"
)

func (h *Handlers) GetTeamsByRace(c *gin.Context) {
	raceID, err := h.resolveRaceID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid race id"})
		return
	}
	teams, err := h.services.Teams.ListTeamsByRace(raceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list teams"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": teams})
}

func (h *Handlers) CreateTeam(c *gin.Context) {
	raceID, err := h.resolveRaceID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid race id"})
		return
	}
	var req createTeamRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	created, err := h.services.Teams.CreateTeam(&models.Team{
		RaceID:       uuidutil.NewPublicUUID(raceID),
		Name:         req.Name,
		DisplayOrder: req.DisplayOrder,
	})
	if err != nil {
		respondServiceError(c, err)
		return
	}
	h.refreshLiveCSVForRace(raceID)
	c.JSON(http.StatusCreated, created)
}

func (h *Handlers) GetTeam(c *gin.Context) {
	id, err := h.resolveTeamID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid team id"})
		return
	}
	team, err := h.services.Teams.GetTeam(id)
	if err != nil {
		respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, team)
}

func (h *Handlers) UpdateTeam(c *gin.Context) {
	id, err := h.resolveTeamID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid team id"})
		return
	}
	var req updateTeamRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	team, err := h.services.Teams.UpdateTeamFields(id, req.Name, req.DisplayOrder)
	if err != nil {
		respondServiceError(c, err)
		return
	}
	h.refreshLiveCSVForRace(team.RaceID.UUID())
	c.JSON(http.StatusOK, team)
}

func (h *Handlers) DeleteTeam(c *gin.Context) {
	id, err := h.resolveTeamID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid team id"})
		return
	}
	team, _ := h.services.Teams.GetTeam(id)
	if err := h.services.Teams.DeleteTeam(id); err != nil {
		respondServiceError(c, err)
		return
	}
	if team != nil {
		h.refreshLiveCSVForRace(team.RaceID.UUID())
	}
	c.JSON(http.StatusOK, gin.H{"message": "team deleted"})
}

func (h *Handlers) SetTeamMembers(c *gin.Context) {
	id, err := h.resolveTeamID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid team id"})
		return
	}
	var req setTeamMembersRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ids := make([]uuid.UUID, 0, len(req.ParticipantIDs))
	for _, raw := range req.ParticipantIDs {
		pid, err := h.resolveParticipantID(raw)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid participant_id"})
			return
		}
		ids = append(ids, pid)
	}
	team, err := h.services.Teams.SetMembers(id, ids)
	if err != nil {
		respondServiceError(c, err)
		return
	}
	h.refreshLiveCSVForRace(team.RaceID.UUID())
	c.JSON(http.StatusOK, team)
}

func (h *Handlers) GetTeamLeaderboard(c *gin.Context) {
	raceID, err := h.resolveRaceID(c.Param("raceId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid race id"})
		return
	}
	board, err := h.services.Results.BuildTeamLeaderboard(raceID)
	if err != nil {
		respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": board})
}
