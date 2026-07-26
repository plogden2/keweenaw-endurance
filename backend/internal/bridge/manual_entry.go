package bridge

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// ManualEntryRequest is the hosted POST /api/rfid/manual-entry body.
type ManualEntryRequest struct {
	RaceID       string
	CheckpointID string
	BibNumber    string
	RFIDTagUID   string
	Timestamp    time.Time
	DeviceID     string
}

type manualEntryJSON struct {
	RaceID       string `json:"race_id"`
	CheckpointID string `json:"checkpoint_id"`
	BibNumber    string `json:"bib_number,omitempty"`
	RFIDTagUID   string `json:"rfid_tag_uid,omitempty"`
	Timestamp    string `json:"timestamp"`
	DeviceID     string `json:"device_id,omitempty"`
}

// EnsureBearer exchanges ORGANIZER_PIN for a JWT when BearerToken is empty.
// Bridge token alone cannot call timerWrite routes.
func EnsureBearer(auth *HostedAuth, client *http.Client, organizerPIN string) error {
	if auth == nil {
		return fmt.Errorf("auth not configured")
	}
	if strings.TrimSpace(auth.BearerToken) != "" {
		return nil
	}
	pin := strings.TrimSpace(organizerPIN)
	if pin == "" {
		return fmt.Errorf("ORGANIZER_PIN is required for manual entry")
	}
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}

	body, err := json.Marshal(pinExchangeRequest{PIN: pin})
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPost, auth.BaseURL+"/api/auth/pin", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("pin exchange: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("pin exchange failed: %s", strings.TrimSpace(string(raw)))
	}
	var out pinExchangeResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		return err
	}
	if strings.TrimSpace(out.Token) == "" {
		return fmt.Errorf("pin exchange returned empty token")
	}
	auth.BearerToken = strings.TrimSpace(out.Token)
	return nil
}

// PostManualEntry records a lap on hosted via admin JWT.
func PostManualEntry(auth *HostedAuth, client *http.Client, in ManualEntryRequest) error {
	if auth == nil {
		return fmt.Errorf("auth not configured")
	}
	if strings.TrimSpace(auth.BearerToken) == "" {
		return fmt.Errorf("bearer token required for manual entry")
	}
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}

	ts := in.Timestamp
	if ts.IsZero() {
		ts = time.Now().UTC()
	}
	payload := manualEntryJSON{
		RaceID:       strings.TrimSpace(in.RaceID),
		CheckpointID: strings.TrimSpace(in.CheckpointID),
		BibNumber:    strings.TrimSpace(in.BibNumber),
		RFIDTagUID:   strings.TrimSpace(in.RFIDTagUID),
		Timestamp:    ts.UTC().Format(time.RFC3339),
		DeviceID:     strings.TrimSpace(in.DeviceID),
	}
	if payload.RaceID == "" || payload.CheckpointID == "" {
		return fmt.Errorf("race_id and checkpoint_id are required")
	}
	if payload.BibNumber == "" && payload.RFIDTagUID == "" {
		return fmt.Errorf("bib_number or rfid_tag_uid is required")
	}

	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPost, auth.BaseURL+"/api/rfid/manual-entry", bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+auth.BearerToken)

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("manual entry failed (%d): %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return nil
}
