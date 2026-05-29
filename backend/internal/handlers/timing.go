package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (h *Handlers) GetLiveTiming(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"error": "live timing not yet implemented"})
}

func (h *Handlers) CreateTimingRecord(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"error": "timing records not yet implemented"})
}

func (h *Handlers) GetRaceResults(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"error": "race results not yet implemented"})
}

func (h *Handlers) GetLeaderboard(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"error": "leaderboard not yet implemented"})
}
