package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

const resultsExcelContentType = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"

// GetEventResultsExcel returns the event standings as an Excel workbook.
func (h *Handlers) GetEventResultsExcel(c *gin.Context) {
	eventID, err := h.resolveEventID(c.Param("id"))
	if err != nil {
		respondServiceError(c, err)
		return
	}

	data, filename, err := h.services.Results.BuildEventResultsWorkbook(eventID)
	if err != nil {
		respondServiceError(c, err)
		return
	}

	c.Header("Content-Disposition", `attachment; filename="`+filename+`"`)
	c.Data(http.StatusOK, resultsExcelContentType, data)
}
