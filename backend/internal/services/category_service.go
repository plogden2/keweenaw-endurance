package services

import (
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/keweenaw-endurance/backend/internal/models"
	"gorm.io/gorm"
)

var (
	ErrCategoryNotFound     = errors.New("category not found")
	ErrInvalidCategoryInput = errors.New("invalid category input")
)

var validCategoryTypes = map[string]bool{
	"overall":   true,
	"male":      true,
	"female":    true,
	"age_group": true,
	"custom":    true,
}

type CategoryService struct {
	db *gorm.DB
}

func NewCategoryService(db *gorm.DB) *CategoryService {
	return &CategoryService{db: db}
}

func (s *CategoryService) ListCategoriesByRace(raceID uuid.UUID, page, limit int) ([]models.Category, int64, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	query := s.db.Model(&models.Category{}).Where("race_id = ?", raceID)

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var categories []models.Category
	offset := (page - 1) * limit
	if err := query.Order("display_order ASC").Offset(offset).Limit(limit).Find(&categories).Error; err != nil {
		return nil, 0, err
	}

	return categories, total, nil
}

func (s *CategoryService) GetCategory(id uuid.UUID) (*models.Category, error) {
	var category models.Category
	if err := s.db.First(&category, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrCategoryNotFound
		}
		return nil, err
	}
	return &category, nil
}

func (s *CategoryService) CreateCategory(input *models.Category) (*models.Category, error) {
	if err := validateCategoryInput(input); err != nil {
		return nil, err
	}

	if err := s.ensureRaceExists(input.RaceID); err != nil {
		return nil, err
	}

	category := *input
	if err := s.db.Create(&category).Error; err != nil {
		return nil, err
	}

	return &category, nil
}

func (s *CategoryService) UpdateCategory(id uuid.UUID, input *models.Category) (*models.Category, error) {
	category, err := s.GetCategory(id)
	if err != nil {
		return nil, err
	}

	if input.Name != "" {
		category.Name = input.Name
	}
	if input.CategoryType != "" {
		if !validCategoryTypes[input.CategoryType] {
			return nil, fmt.Errorf("%w: invalid category_type", ErrInvalidCategoryInput)
		}
		category.CategoryType = input.CategoryType
	}
	if input.AgeMin > 0 {
		category.AgeMin = input.AgeMin
	}
	if input.AgeMax > 0 {
		category.AgeMax = input.AgeMax
	}
	if input.GenderFilter != "" {
		category.GenderFilter = input.GenderFilter
	}
	if input.DisplayOrder > 0 {
		category.DisplayOrder = input.DisplayOrder
	}

	if err := validateCategoryInput(category); err != nil {
		return nil, err
	}

	if err := s.db.Save(category).Error; err != nil {
		return nil, err
	}

	return category, nil
}

func (s *CategoryService) DeleteCategory(id uuid.UUID) error {
	result := s.db.Delete(&models.Category{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrCategoryNotFound
	}
	return nil
}

func (s *CategoryService) ensureRaceExists(raceID uuid.UUID) error {
	var race models.Race
	if err := s.db.First(&race, "id = ?", raceID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("%w: race not found", ErrInvalidCategoryInput)
		}
		return err
	}
	return nil
}

func validateCategoryInput(category *models.Category) error {
	if category == nil {
		return fmt.Errorf("%w: category is required", ErrInvalidCategoryInput)
	}
	if category.RaceID == uuid.Nil {
		return fmt.Errorf("%w: race_id is required", ErrInvalidCategoryInput)
	}
	if strings.TrimSpace(category.Name) == "" {
		return fmt.Errorf("%w: name is required", ErrInvalidCategoryInput)
	}
	if category.CategoryType == "" {
		return fmt.Errorf("%w: category_type is required", ErrInvalidCategoryInput)
	}
	if !validCategoryTypes[category.CategoryType] {
		return fmt.Errorf("%w: invalid category_type", ErrInvalidCategoryInput)
	}
	if category.CategoryType == "age_group" && (category.AgeMin <= 0 || category.AgeMax <= 0 || category.AgeMin > category.AgeMax) {
		return fmt.Errorf("%w: age_min and age_max are required for age_group categories", ErrInvalidCategoryInput)
	}
	return nil
}
