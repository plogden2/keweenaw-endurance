package handlers

// Create request DTOs require all mandatory fields. Update DTOs use pointers so
// clients may send only the fields they want to change (JSON partial update).

type createEventRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	EventDate   string `json:"event_date" binding:"required"`
	Location    string `json:"location"`
	WebsiteURL  string `json:"website_url"`
	Status      string `json:"status"`
}

type updateEventRequest struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
	EventDate   *string `json:"event_date"`
	Location    *string `json:"location"`
	WebsiteURL  *string `json:"website_url"`
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
	RaceID     string `json:"race_id" binding:"required"`
	BibNumber  string `json:"bib_number" binding:"required"`
	FirstName  string `json:"first_name" binding:"required"`
	LastName   string `json:"last_name" binding:"required"`
	Gender     string `json:"gender"`
	Age        int    `json:"age"`
	RFIDTagUID string `json:"rfid_tag_uid"`
	Status     string `json:"status"`
}

type updateParticipantRequest struct {
	BibNumber  *string `json:"bib_number"`
	FirstName  *string `json:"first_name"`
	LastName   *string `json:"last_name"`
	Gender     *string `json:"gender"`
	Age        *int    `json:"age"`
	RFIDTagUID *string `json:"rfid_tag_uid"`
	Status     *string `json:"status"`
}
