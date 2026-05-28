package models

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	
	// Auto migrate all models
	err = db.AutoMigrate(&Event{}, &Race{}, &Participant{}, &TimingCheckpoint{}, &TimingRecord{}, &Category{})
	require.NoError(t, err)
	
	return db
}

func TestEventModel(t *testing.T) {
	db := setupTestDB(t)
	
	t.Run("CreateEvent", func(t *testing.T) {
		event := Event{
			Name:        "Test Event",
			Description: "Test Description",
			EventDate:   time.Now().AddDate(0, 1, 0), // Next month
			Location:    "Test Location",
			WebsiteURL:  "https://example.com",
			Status:      "upcoming",
		}
		
		err := db.Create(&event).Error
		require.NoError(t, err)
		
		// Verify UUID was generated
		assert.NotEqual(t, uuid.Nil, event.ID)
		
		// Verify timestamps
		assert.False(t, event.CreatedAt.IsZero())
		assert.False(t, event.UpdatedAt.IsZero())
		
		// Verify fields
		assert.Equal(t, "Test Event", event.Name)
		assert.Equal(t, "Test Description", event.Description)
		assert.Equal(t, "Test Location", event.Location)
		assert.Equal(t, "https://example.com", event.WebsiteURL)
		assert.Equal(t, "upcoming", event.Status)
	})
	
	t.Run("EventValidation", func(t *testing.T) {
		// Test invalid status
		event := Event{
			Name:      "Invalid Event",
			EventDate: time.Now(),
			Status:    "invalid_status",
		}
		
		err := db.Create(&event).Error
		assert.Error(t, err)
	})
	
	t.Run("EventRelationships", func(t *testing.T) {
		event := Event{
			Name:      "Event with Races",
			EventDate: time.Now(),
			Status:    "upcoming",
		}
		
		err := db.Create(&event).Error
		require.NoError(t, err)
		
		// Create related races
		race1 := Race{
			EventID:  event.ID,
			Name:     "Race 1",
			RaceType: "time_based",
			Status:   "scheduled",
		}
		
		race2 := Race{
			EventID:  event.ID,
			Name:     "Race 2",
			RaceType: "lap_based",
			Status:   "scheduled",
		}
		
		err = db.Create(&race1).Error
		require.NoError(t, err)
		
		err = db.Create(&race2).Error
		require.NoError(t, err)
		
		// Load races with event
		var loadedEvent Event
		err = db.Preload("Races").First(&loadedEvent, event.ID).Error
		require.NoError(t, err)
		
		assert.Len(t, loadedEvent.Races, 2)
	})
}

func TestRaceModel(t *testing.T) {
	db := setupTestDB(t)
	
	// Create parent event first
	event := Event{
		Name:      "Parent Event",
		EventDate: time.Now(),
		Status:    "upcoming",
	}
	err := db.Create(&event).Error
	require.NoError(t, err)
	
	t.Run("CreateRace", func(t *testing.T) {
		race := Race{
			EventID:         event.ID,
			Name:            "Test Race",
			RaceType:        "time_based",
			DistanceKm:      42.195, // Marathon distance
			DurationMinutes: 0,
			StartTime:       time.Now().Add(time.Hour),
			Status:          "scheduled",
		}
		
		err := db.Create(&race).Error
		require.NoError(t, err)
		
		// Verify UUID was generated
		assert.NotEqual(t, uuid.Nil, race.ID)
		
		// Verify fields
		assert.Equal(t, event.ID, race.EventID)
		assert.Equal(t, "Test Race", race.Name)
		assert.Equal(t, "time_based", race.RaceType)
		assert.Equal(t, 42.195, race.DistanceKm)
		assert.Equal(t, "scheduled", race.Status)
	})
	
	t.Run("LapBasedRace", func(t *testing.T) {
		race := Race{
			EventID:         event.ID,
			Name:            "Lap Race",
			RaceType:        "lap_based",
			DistanceKm:      0,
			DurationMinutes: 60, // 1 hour
			StartTime:       time.Now().Add(time.Hour),
			Status:          "scheduled",
		}
		
		err := db.Create(&race).Error
		require.NoError(t, err)
		
		assert.Equal(t, "lap_based", race.RaceType)
		assert.Equal(t, 60, race.DurationMinutes)
	})
	
	t.Run("RaceValidation", func(t *testing.T) {
		// Test invalid race type
		race := Race{
			EventID:  event.ID,
			Name:     "Invalid Race",
			RaceType: "invalid_type",
			Status:   "scheduled",
		}
		
		err := db.Create(&race).Error
		assert.Error(t, err)
	})
}

func TestParticipantModel(t *testing.T) {
	db := setupTestDB(t)
	
	// Create parent event and race first
	event := Event{
		Name:      "Parent Event",
		EventDate: time.Now(),
		Status:    "upcoming",
	}
	err := db.Create(&event).Error
	require.NoError(t, err)
	
	race := Race{
		EventID:  event.ID,
		Name:     "Parent Race",
		RaceType: "time_based",
		Status:   "scheduled",
	}
	err = db.Create(&race).Error
	require.NoError(t, err)
	
	t.Run("CreateParticipant", func(t *testing.T) {
		participant := Participant{
			RaceID:    race.ID,
			BibNumber: "001",
			FirstName: "John",
			LastName:  "Doe",
			Gender:    "male",
			Age:       30,
			RFIDTagUID: "RFID123456",
			Status:    "registered",
		}
		
		err := db.Create(&participant).Error
		require.NoError(t, err)
		
		// Verify UUID was generated
		assert.NotEqual(t, uuid.Nil, participant.ID)
		
		// Verify fields
		assert.Equal(t, race.ID, participant.RaceID)
		assert.Equal(t, "001", participant.BibNumber)
		assert.Equal(t, "John", participant.FirstName)
		assert.Equal(t, "Doe", participant.LastName)
		assert.Equal(t, "male", participant.Gender)
		assert.Equal(t, 30, participant.Age)
		assert.Equal(t, "RFID123456", participant.RFIDTagUID)
		assert.Equal(t, "registered", participant.Status)
	})
	
	t.Run("ParticipantValidation", func(t *testing.T) {
		// Test invalid gender
		participant := Participant{
			RaceID:    race.ID,
			BibNumber: "002",
			FirstName: "Jane",
			LastName:  "Smith",
			Gender:    "invalid_gender",
			Status:    "registered",
		}
		
		err := db.Create(&participant).Error
		assert.Error(t, err)
	})
	
	t.Run("UniqueRFIDTag", func(t *testing.T) {
		// Create first participant with RFID tag
		participant1 := Participant{
			RaceID:     race.ID,
			BibNumber:  "003",
			FirstName:  "Alice",
			LastName:   "Johnson",
			RFIDTagUID: "UNIQUE_RFID_123",
			Status:     "registered",
		}
		
		err := db.Create(&participant1).Error
		require.NoError(t, err)
		
		// Try to create second participant with same RFID tag
		participant2 := Participant{
			RaceID:     race.ID,
			BibNumber:  "004",
			FirstName:  "Bob",
			LastName:   "Brown",
			RFIDTagUID: "UNIQUE_RFID_123", // Same RFID tag
			Status:     "registered",
		}
		
		err = db.Create(&participant2).Error
		assert.Error(t, err) // Should fail due to unique constraint
	})
}

func TestTimingCheckpointModel(t *testing.T) {
	db := setupTestDB(t)
	
	// Create parent event and race first
	event := Event{
		Name:      "Parent Event",
		EventDate: time.Now(),
		Status:    "upcoming",
	}
	err := db.Create(&event).Error
	require.NoError(t, err)
	
	race := Race{
		EventID:  event.ID,
		Name:     "Parent Race",
		RaceType: "time_based",
		Status:   "scheduled",
	}
	err = db.Create(&race).Error
	require.NoError(t, err)
	
	t.Run("CreateTimingCheckpoint", func(t *testing.T) {
		checkpoint := TimingCheckpoint{
			RaceID:              race.ID,
			Name:                "Start Line",
			CheckpointType:      "start",
			DistanceFromStartKm: 0.0,
			LocationDescription: "Main start area",
			IsActive:            true,
		}
		
		err := db.Create(&checkpoint).Error
		require.NoError(t, err)
		
		// Verify UUID was generated
		assert.NotEqual(t, uuid.Nil, checkpoint.ID)
		
		// Verify fields
		assert.Equal(t, race.ID, checkpoint.RaceID)
		assert.Equal(t, "Start Line", checkpoint.Name)
		assert.Equal(t, "start", checkpoint.CheckpointType)
		assert.Equal(t, 0.0, checkpoint.DistanceFromStartKm)
		assert.Equal(t, "Main start area", checkpoint.LocationDescription)
		assert.True(t, checkpoint.IsActive)
	})
	
	t.Run("DifferentCheckpointTypes", func(t *testing.T) {
		// Start checkpoint
		startCheckpoint := TimingCheckpoint{
			RaceID:         race.ID,
			Name:           "Start",
			CheckpointType: "start",
		}
		err := db.Create(&startCheckpoint).Error
		require.NoError(t, err)
		
		// Finish checkpoint
		finishCheckpoint := TimingCheckpoint{
			RaceID:         race.ID,
			Name:           "Finish",
			CheckpointType: "finish",
		}
		err = db.Create(&finishCheckpoint).Error
		require.NoError(t, err)
		
		// Intermediate checkpoint
		intermediateCheckpoint := TimingCheckpoint{
			RaceID:         race.ID,
			Name:           "CP1",
			CheckpointType: "intermediate",
		}
		err = db.Create(&intermediateCheckpoint).Error
		require.NoError(t, err)
	})
	
	t.Run("CheckpointValidation", func(t *testing.T) {
		// Test invalid checkpoint type
		checkpoint := TimingCheckpoint{
			RaceID:         race.ID,
			Name:           "Invalid",
			CheckpointType: "invalid_type",
		}
		
		err := db.Create(&checkpoint).Error
		assert.Error(t, err)
	})
}

func TestTimingRecordModel(t *testing.T) {
	db := setupTestDB(t)
	
	// Create parent entities first
	event := Event{
		Name:      "Parent Event",
		EventDate: time.Now(),
		Status:    "upcoming",
	}
	err := db.Create(&event).Error
	require.NoError(t, err)
	
	race := Race{
		EventID:  event.ID,
		Name:     "Parent Race",
		RaceType: "time_based",
		Status:   "scheduled",
	}
	err = db.Create(&race).Error
	require.NoError(t, err)
	
	participant := Participant{
		RaceID:    race.ID,
		BibNumber: "001",
		FirstName: "John",
		LastName:  "Doe",
		Status:    "registered",
	}
	err = db.Create(&participant).Error
	require.NoError(t, err)
	
	checkpoint := TimingCheckpoint{
		RaceID:         race.ID,
		Name:           "Start",
		CheckpointType: "start",
	}
	err = db.Create(&checkpoint).Error
	require.NoError(t, err)
	
	t.Run("CreateTimingRecord", func(t *testing.T) {
		now := time.Now()
		record := TimingRecord{
			ParticipantID:  participant.ID,
			CheckpointID:   checkpoint.ID,
			Timestamp:      now,
			LocalTimestamp: now,
			DeviceID:       "DEVICE_001",
			SyncStatus:     "synced",
		}
		
		err := db.Create(&record).Error
		require.NoError(t, err)
		
		// Verify UUID was generated
		assert.NotEqual(t, uuid.Nil, record.ID)
		
		// Verify fields
		assert.Equal(t, participant.ID, record.ParticipantID)
		assert.Equal(t, checkpoint.ID, record.CheckpointID)
		assert.Equal(t, now.Unix(), record.Timestamp.Unix())
		assert.Equal(t, now.Unix(), record.LocalTimestamp.Unix())
		assert.Equal(t, "DEVICE_001", record.DeviceID)
		assert.Equal(t, "synced", record.SyncStatus)
	})
	
	t.Run("DifferentSyncStatuses", func(t *testing.T) {
		now := time.Now()
		
		// Synced record
		record1 := TimingRecord{
			ParticipantID:  participant.ID,
			CheckpointID:   checkpoint.ID,
			Timestamp:      now,
			LocalTimestamp: now,
			SyncStatus:     "synced",
		}
		err := db.Create(&record1).Error
		require.NoError(t, err)
		
		// Pending sync record
		record2 := TimingRecord{
			ParticipantID:  participant.ID,
			CheckpointID:   checkpoint.ID,
			Timestamp:      now,
			LocalTimestamp: now,
			SyncStatus:     "pending_sync",
		}
		err = db.Create(&record2).Error
		require.NoError(t, err)
		
		// Failed sync record
		record3 := TimingRecord{
			ParticipantID:  participant.ID,
			CheckpointID:   checkpoint.ID,
			Timestamp:      now,
			LocalTimestamp: now,
			SyncStatus:     "failed_sync",
		}
		err = db.Create(&record3).Error
		require.NoError(t, err)
	})
}

func TestCategoryModel(t *testing.T) {
	db := setupTestDB(t)
	
	// Create parent event and race first
	event := Event{
		Name:      "Parent Event",
		EventDate: time.Now(),
		Status:    "upcoming",
	}
	err := db.Create(&event).Error
	require.NoError(t, err)
	
	race := Race{
		EventID:  event.ID,
		Name:     "Parent Race",
		RaceType: "time_based",
		Status:   "scheduled",
	}
	err = db.Create(&race).Error
	require.NoError(t, err)
	
	t.Run("CreateCategory", func(t *testing.T) {
		category := Category{
			RaceID:       race.ID,
			Name:         "Overall",
			CategoryType: "overall",
			DisplayOrder: 1,
		}
		
		err := db.Create(&category).Error
		require.NoError(t, err)
		
		// Verify UUID was generated
		assert.NotEqual(t, uuid.Nil, category.ID)
		
		// Verify fields
		assert.Equal(t, race.ID, category.RaceID)
		assert.Equal(t, "Overall", category.Name)
		assert.Equal(t, "overall", category.CategoryType)
		assert.Equal(t, 1, category.DisplayOrder)
	})
	
	t.Run("AgeGroupCategory", func(t *testing.T) {
		category := Category{
			RaceID:       race.ID,
			Name:         "Masters (40-49)",
			CategoryType: "age_group",
			AgeMin:       40,
			AgeMax:       49,
			GenderFilter: "male",
			DisplayOrder: 2,
		}
		
		err := db.Create(&category).Error
		require.NoError(t, err)
		
		assert.Equal(t, "age_group", category.CategoryType)
		assert.Equal(t, 40, category.AgeMin)
		assert.Equal(t, 49, category.AgeMax)
		assert.Equal(t, "male", category.GenderFilter)
	})
	
	t.Run("CategoryValidation", func(t *testing.T) {
		// Test invalid category type
		category := Category{
			RaceID:       race.ID,
			Name:         "Invalid",
			CategoryType: "invalid_type",
			DisplayOrder: 3,
		}
		
		err := db.Create(&category).Error
		assert.Error(t, err)
	})
}

func TestModelUUIDGeneration(t *testing.T) {
	db := setupTestDB(t)
	
	t.Run("AllModelsGenerateUUID", func(t *testing.T) {
		// Create parent entities
		event := Event{
			Name:      "Test Event",
			EventDate: time.Now(),
			Status:    "upcoming",
		}
		err := db.Create(&event).Error
		require.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, event.ID)
		
		race := Race{
			EventID:  event.ID,
			Name:     "Test Race",
			RaceType: "time_based",
			Status:   "scheduled",
		}
		err = db.Create(&race).Error
		require.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, race.ID)
		
		participant := Participant{
			RaceID:    race.ID,
			BibNumber: "001",
			FirstName: "Test",
			LastName:  "Participant",
			Status:    "registered",
		}
		err = db.Create(&participant).Error
		require.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, participant.ID)
		
		checkpoint := TimingCheckpoint{
			RaceID:         race.ID,
			Name:           "Test Checkpoint",
			CheckpointType: "intermediate",
		}
		err = db.Create(&checkpoint).Error
		require.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, checkpoint.ID)
		
		record := TimingRecord{
			ParticipantID: participant.ID,
			CheckpointID:  checkpoint.ID,
			Timestamp:     time.Now(),
			LocalTimestamp: time.Now(),
		}
		err = db.Create(&record).Error
		require.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, record.ID)
		
		category := Category{
			RaceID:       race.ID,
			Name:         "Test Category",
			CategoryType: "custom",
		}
		err = db.Create(&category).Error
		require.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, category.ID)
	})
}