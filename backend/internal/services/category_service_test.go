package services

import (
	"testing"

	"github.com/google/uuid"
	"github.com/keweenaw-endurance/backend/internal/models"
	"github.com/keweenaw-endurance/backend/internal/uuidutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCategoryService_CreateAndGet(t *testing.T) {
	db := setupServiceTestDB(t)
	race := createTestRace(t, db)
	svc := NewCategoryService(db)

	category, err := svc.CreateCategory(&models.Category{
		RaceID:       race.ID,
		Name:         "Overall",
		CategoryType: "overall",
		DisplayOrder: 1,
	})
	require.NoError(t, err)
	assert.False(t, category.ID.IsZero())

	fetched, err := svc.GetCategory(category.ID.UUID())
	require.NoError(t, err)
	assert.Equal(t, "Overall", fetched.Name)
	assert.Equal(t, race.ID, fetched.RaceID)
}

func TestCategoryService_CreateValidation(t *testing.T) {
	db := setupServiceTestDB(t)
	race := createTestRace(t, db)
	svc := NewCategoryService(db)

	_, err := svc.CreateCategory(&models.Category{
		RaceID: race.ID,
	})
	assert.ErrorIs(t, err, ErrInvalidCategoryInput)

	_, err = svc.CreateCategory(&models.Category{
		RaceID:       race.ID,
		Name:         "Bad",
		CategoryType: "invalid",
	})
	assert.ErrorIs(t, err, ErrInvalidCategoryInput)

	_, err = svc.CreateCategory(&models.Category{
		RaceID:       uuidutil.NewPublicUUID(uuid.New()),
		Name:         "Orphan",
		CategoryType: "male",
	})
	assert.ErrorIs(t, err, ErrInvalidCategoryInput)

	_, err = svc.CreateCategory(&models.Category{
		RaceID:       race.ID,
		Name:         "M40-49",
		CategoryType: "age_group",
	})
	assert.ErrorIs(t, err, ErrInvalidCategoryInput)
}

func TestCategoryService_AgeGroup(t *testing.T) {
	db := setupServiceTestDB(t)
	race := createTestRace(t, db)
	svc := NewCategoryService(db)

	category, err := svc.CreateCategory(&models.Category{
		RaceID:       race.ID,
		Name:         "Men 40-49",
		CategoryType: "age_group",
		AgeMin:       40,
		AgeMax:       49,
		GenderFilter: "male",
	})
	require.NoError(t, err)
	assert.Equal(t, 40, category.AgeMin)
	assert.Equal(t, 49, category.AgeMax)
}

func TestCategoryService_ListByRace(t *testing.T) {
	db := setupServiceTestDB(t)
	race := createTestRace(t, db)
	svc := NewCategoryService(db)

	for _, name := range []string{"Overall", "Female"} {
		_, err := svc.CreateCategory(&models.Category{
			RaceID:       race.ID,
			Name:         name,
			CategoryType: "overall",
		})
		require.NoError(t, err)
	}

	categories, total, err := svc.ListCategoriesByRace(race.ID.UUID(), 1, 20)
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
	assert.Len(t, categories, 2)
}

func TestCategoryService_UpdateAndDelete(t *testing.T) {
	db := setupServiceTestDB(t)
	race := createTestRace(t, db)
	svc := NewCategoryService(db)

	category, err := svc.CreateCategory(&models.Category{
		RaceID:       race.ID,
		Name:         "Male",
		CategoryType: "male",
	})
	require.NoError(t, err)

	updated, err := svc.UpdateCategory(category.ID.UUID(), &models.Category{
		Name:         "Men",
		DisplayOrder: 2,
	})
	require.NoError(t, err)
	assert.Equal(t, "Men", updated.Name)
	assert.Equal(t, 2, updated.DisplayOrder)

	require.NoError(t, svc.DeleteCategory(category.ID.UUID()))

	_, err = svc.GetCategory(category.ID.UUID())
	assert.ErrorIs(t, err, ErrCategoryNotFound)
}

func TestCategoryService_GetNotFound(t *testing.T) {
	db := setupServiceTestDB(t)
	svc := NewCategoryService(db)

	_, err := svc.GetCategory(uuid.New())
	assert.ErrorIs(t, err, ErrCategoryNotFound)
}
