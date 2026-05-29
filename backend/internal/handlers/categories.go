package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/keweenaw-endurance/backend/internal/models"
)

func (h *Handlers) GetCategoriesByRace(c *gin.Context) {
	raceID, err := parseUUID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid race id"})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	categories, total, err := h.services.Categories.ListCategoriesByRace(raceID, page, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list categories"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  categories,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}

func (h *Handlers) CreateCategory(c *gin.Context) {
	raceID, err := parseUUID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid race id"})
		return
	}

	var req createCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	category := &models.Category{
		RaceID:       raceID,
		Name:         req.Name,
		CategoryType: req.CategoryType,
		AgeMin:       req.AgeMin,
		AgeMax:       req.AgeMax,
		GenderFilter: req.GenderFilter,
		DisplayOrder: req.DisplayOrder,
	}

	created, err := h.services.Categories.CreateCategory(category)
	if err != nil {
		respondServiceError(c, err)
		return
	}

	c.JSON(http.StatusCreated, created)
}

func (h *Handlers) GetCategory(c *gin.Context) {
	id, err := parseUUID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid category id"})
		return
	}

	category, err := h.services.Categories.GetCategory(id)
	if err != nil {
		respondServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, category)
}

func (h *Handlers) UpdateCategory(c *gin.Context) {
	id, err := parseUUID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid category id"})
		return
	}

	var req updateCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	update := &models.Category{}
	if req.Name != nil {
		update.Name = *req.Name
	}
	if req.CategoryType != nil {
		update.CategoryType = *req.CategoryType
	}
	if req.AgeMin != nil {
		update.AgeMin = *req.AgeMin
	}
	if req.AgeMax != nil {
		update.AgeMax = *req.AgeMax
	}
	if req.GenderFilter != nil {
		update.GenderFilter = *req.GenderFilter
	}
	if req.DisplayOrder != nil {
		update.DisplayOrder = *req.DisplayOrder
	}

	category, err := h.services.Categories.UpdateCategory(id, update)
	if err != nil {
		respondServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, category)
}

func (h *Handlers) DeleteCategory(c *gin.Context) {
	id, err := parseUUID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid category id"})
		return
	}

	if err := h.services.Categories.DeleteCategory(id); err != nil {
		respondServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "category deleted"})
}
