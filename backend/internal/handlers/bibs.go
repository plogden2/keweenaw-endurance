package handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/keweenaw-endurance/backend/internal/services"
	"github.com/keweenaw-endurance/backend/internal/uuidutil"
)

type bulkCreateEventBibsRequest struct {
	From int `json:"from"`
	To   int `json:"to"`
}

// ListEventBibs handles GET /api/events/:id/bibs.
func (h *Handlers) ListEventBibs(c *gin.Context) {
	eventID, err := h.resolveEventID(c.Param("id"))
	if err != nil {
		respondServiceError(c, err)
		return
	}

	items, err := h.services.Bibs.ListBibs(eventID)
	if err != nil {
		respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": items})
}

// BulkCreateEventBibs handles POST /api/events/:id/bibs/bulk.
func (h *Handlers) BulkCreateEventBibs(c *gin.Context) {
	eventID, err := h.resolveEventID(c.Param("id"))
	if err != nil {
		respondServiceError(c, err)
		return
	}

	var req bulkCreateEventBibsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	bibs, err := h.services.Bibs.BulkCreateBibs(eventID, req.From, req.To)
	if err != nil {
		respondServiceError(c, err)
		return
	}

	h.refreshLiveCSV(eventID)
	c.JSON(http.StatusCreated, bibs)
}

// ListBibTags handles GET /api/events/:id/bibs/:bibId/tags.
func (h *Handlers) ListBibTags(c *gin.Context) {
	eventID, err := h.resolveEventID(c.Param("id"))
	if err != nil {
		respondServiceError(c, err)
		return
	}
	bibID, err := h.resolveBibID(c.Param("bibId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid bib id"})
		return
	}
	if _, err := h.services.Bibs.GetBib(eventID, bibID); err != nil {
		respondServiceError(c, err)
		return
	}

	tags, err := h.services.Bibs.ListBibTags(bibID)
	if err != nil {
		respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": tags})
}

// PostBibTag handles POST /api/events/:id/bibs/:bibId/tags.
// Empty body (or no tag_uid) programs the chip with the bib UUID; {tag_uid} associates without hardware write.
func (h *Handlers) PostBibTag(c *gin.Context) {
	eventID, err := h.resolveEventID(c.Param("id"))
	if err != nil {
		respondServiceError(c, err)
		return
	}
	bibID, err := h.resolveBibID(c.Param("bibId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid bib id"})
		return
	}
	if _, err := h.services.Bibs.GetBib(eventID, bibID); err != nil {
		respondServiceError(c, err)
		return
	}

	var req participantTagRequest
	_ = c.ShouldBindJSON(&req) // body optional

	if req.TagUID != "" {
		if _, err := h.services.RFID.AssociateTagToBib(bibID, req.TagUID); err != nil {
			respondServiceError(c, err)
			return
		}
		h.refreshLiveCSV(eventID)
		c.JSON(http.StatusCreated, bibTagResponse(bibID, req.TagUID, h.services.Bibs))
		return
	}

	bib, err := h.services.RFID.WriteTagForBib(bibID)
	if err != nil {
		respondServiceError(c, err)
		return
	}
	logical := strings.ToLower(bib.ID.String())
	h.refreshLiveCSV(eventID)
	c.JSON(http.StatusCreated, bibTagResponse(bib.ID.UUID(), logical, h.services.Bibs))
}

func bibTagResponse(bibID uuid.UUID, tagUID string, bibs *services.BibService) gin.H {
	resp := gin.H{
		"tag_uid": tagUID,
		"bib_id":  uuidutil.NewPublicUUID(bibID).Short(),
	}
	if bibs != nil {
		if tags, err := bibs.ListBibTags(bibID); err == nil {
			uids := make([]string, 0, len(tags))
			for _, t := range tags {
				uids = append(uids, t.TagUID)
			}
			resp["tag_uids"] = uids
		}
	}
	return resp
}
