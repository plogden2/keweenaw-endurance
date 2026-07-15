package handlers

// Create request DTOs require all mandatory fields. Update DTOs use pointers so
// clients may send only the fields they want to change (JSON partial update).

type createEventRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	EventDate   string `json:"event_date" binding:"required"`
	Location    string `json:"location"`
	WebsiteURL  string `json:"website_url"`
	LogoURL     string `json:"logo_url"`
	Status      string `json:"status"`
}

type updateEventRequest struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
	EventDate   *string `json:"event_date"`
	Location    *string `json:"location"`
	WebsiteURL  *string `json:"website_url"`
	LogoURL     *string `json:"logo_url"`
	Status      *string `json:"status"`
}

type createRaceRequest struct {
	EventID         string  `json:"event_id" binding:"required"`
	Name            string  `json:"name" binding:"required"`
	RaceType        string  `json:"race_type" binding:"required"`
	DistanceKm      float64 `json:"distance_km"`
	DurationMinutes int     `json:"duration_minutes"`
	StartTime       string  `json:"start_time"`
	Status          string  `json:"status"`
}

type updateRaceRequest struct {
	Name            *string  `json:"name"`
	RaceType        *string  `json:"race_type"`
	DistanceKm      *float64 `json:"distance_km"`
	DurationMinutes *int     `json:"duration_minutes"`
	StartTime       *string  `json:"start_time"`
	Status          *string  `json:"status"`
}

type createParticipantRequest struct {
	RaceID     string `json:"race_id"`
	BibNumber  string `json:"bib_number"`
	FirstName  string `json:"first_name" binding:"required"`
	LastName   string `json:"last_name" binding:"required"`
	Gender     string `json:"gender"`
	Age        int    `json:"age"`
	Location   string `json:"location"`
	RFIDTagUID string `json:"rfid_tag_uid"`
	Status     string `json:"status"`
	CategoryID string `json:"category_id"`
}

type updateParticipantRequest struct {
	BibNumber  *string `json:"bib_number"`
	FirstName  *string `json:"first_name"`
	LastName   *string `json:"last_name"`
	Gender     *string `json:"gender"`
	Age        *int    `json:"age"`
	Location   *string `json:"location"`
	RFIDTagUID *string `json:"rfid_tag_uid"`
	Status     *string `json:"status"`
	CategoryID *string `json:"category_id"`
}

type createCheckpointRequest struct {
	Name                  string  `json:"name" binding:"required"`
	CheckpointType        string  `json:"checkpoint_type" binding:"required"`
	DistanceFromStartKm   float64 `json:"distance_from_start_km"`
	LocationDescription   string  `json:"location_description"`
	IsActive              *bool   `json:"is_active"`
}

type updateCheckpointRequest struct {
	Name                *string  `json:"name"`
	CheckpointType      *string  `json:"checkpoint_type"`
	DistanceFromStartKm *float64 `json:"distance_from_start_km"`
	LocationDescription *string  `json:"location_description"`
	IsActive            *bool    `json:"is_active"`
}

type createCategoryRequest struct {
	Name         string `json:"name" binding:"required"`
	CategoryType string `json:"category_type" binding:"required"`
	AgeMin       int    `json:"age_min"`
	AgeMax       int    `json:"age_max"`
	GenderFilter string `json:"gender_filter"`
	DisplayOrder int    `json:"display_order"`
}

type updateCategoryRequest struct {
	Name         *string `json:"name"`
	CategoryType *string `json:"category_type"`
	AgeMin       *int    `json:"age_min"`
	AgeMax       *int    `json:"age_max"`
	GenderFilter *string `json:"gender_filter"`
	DisplayOrder *int    `json:"display_order"`
}

type createTimingRecordRequest struct {
	ParticipantID  string `json:"participant_id" binding:"required"`
	CheckpointID   string `json:"checkpoint_id" binding:"required"`
	Timestamp      string `json:"timestamp" binding:"required"`
	LocalTimestamp string `json:"local_timestamp"`
	DeviceID       string `json:"device_id"`
	SyncStatus     string `json:"sync_status"`
}

type updateTimingRecordRequest struct {
	Timestamp      *string `json:"timestamp"`
	LocalTimestamp *string `json:"local_timestamp"`
	DeviceID       *string `json:"device_id"`
	SyncStatus     *string `json:"sync_status"`
}

type writeRFIDTagRequest struct {
	ParticipantID string `json:"participant_id" binding:"required"`
}

type participantTagRequest struct {
	// Optional: associate a UID without hardware write (e.g. typed/demo tags).
	TagUID string `json:"tag_uid"`
}

type injectRFIDTagRequest struct {
	TagUID string `json:"tag_uid" binding:"required"`
}

type manualTimingEntryRequest struct {
	RaceID       string `json:"race_id" binding:"required"`
	CheckpointID string `json:"checkpoint_id" binding:"required"`
	BibNumber    string `json:"bib_number"`
	RFIDTagUID   string `json:"rfid_tag_uid"`
	Timestamp    string `json:"timestamp" binding:"required"`
	DeviceID     string `json:"device_id"`
	SyncStatus   string `json:"sync_status"`
}

type processScanRequest struct {
	TagUID         string `json:"tag_uid" binding:"required"`
	DeviceID       string `json:"device_id"`
	LocalTimestamp string `json:"local_timestamp"`
}

type putStationRequest struct {
	EventID      string  `json:"event_id" binding:"required"`
	Mode         string  `json:"mode"`
	CheckpointID *string `json:"checkpoint_id"`
	DeviceID     string  `json:"device_id"`
	Name         string  `json:"name"`
}
