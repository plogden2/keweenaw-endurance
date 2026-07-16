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

// HostedAuth carries credentials for hosted API and WebSocket calls.
type HostedAuth struct {
	BaseURL     string
	BridgeToken string
	BearerToken string
}

type pinExchangeRequest struct {
	PIN string `json:"pin"`
}

type pinExchangeResponse struct {
	Token string `json:"token"`
}

// ResolveHostedAuth returns bridge-token or PIN-derived JWT credentials.
func ResolveHostedAuth(baseURL, bridgeToken, organizerPIN string, client *http.Client) (*HostedAuth, error) {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	bridgeToken = strings.TrimSpace(bridgeToken)
	organizerPIN = strings.TrimSpace(organizerPIN)

	if baseURL == "" {
		return nil, fmt.Errorf("HOSTED_API_URL is required")
	}
	if bridgeToken == "" && organizerPIN == "" {
		return nil, fmt.Errorf("BRIDGE_TOKEN or ORGANIZER_PIN is required")
	}

	auth := &HostedAuth{
		BaseURL:     baseURL,
		BridgeToken: bridgeToken,
	}
	if bridgeToken != "" {
		return auth, nil
	}

	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}

	body, err := json.Marshal(pinExchangeRequest{PIN: organizerPIN})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodPost, baseURL+"/api/auth/pin", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("pin exchange: %w", err)
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("pin exchange failed: %s", strings.TrimSpace(string(raw)))
	}

	var out pinExchangeResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	if strings.TrimSpace(out.Token) == "" {
		return nil, fmt.Errorf("pin exchange returned empty token")
	}
	auth.BearerToken = strings.TrimSpace(out.Token)
	return auth, nil
}

// WSHeaders returns dial headers for /api/rfid/bridge.
func (a *HostedAuth) WSHeaders() http.Header {
	h := make(http.Header)
	if a == nil {
		return h
	}
	if a.BridgeToken != "" {
		h.Set("X-Bridge-Token", a.BridgeToken)
	}
	if a.BearerToken != "" {
		h.Set("Authorization", "Bearer "+a.BearerToken)
	}
	return h
}

// BridgeWebSocketURL builds ws(s)://host/api/rfid/bridge?device_id=...
func BridgeWebSocketURL(baseURL, deviceID string) (string, error) {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	deviceID = strings.TrimSpace(deviceID)
	if baseURL == "" {
		return "", fmt.Errorf("base url is required")
	}
	if deviceID == "" {
		return "", fmt.Errorf("device id is required")
	}

	scheme := "ws"
	if strings.HasPrefix(baseURL, "https://") {
		scheme = "wss"
	}
	host := strings.TrimPrefix(strings.TrimPrefix(baseURL, "https://"), "http://")
	return fmt.Sprintf("%s://%s/api/rfid/bridge?device_id=%s", scheme, host, deviceID), nil
}

// ParticipantTag is the first active tag for a racer.
type ParticipantTag struct {
	TagUID string `json:"tag_uid"`
}

// FetchLogicalUUID loads the active logical tag for a participant from hosted.
func (a *HostedAuth) FetchLogicalUUID(client *http.Client, raceID, participantID string) (string, error) {
	if a == nil {
		return "", fmt.Errorf("auth not configured")
	}
	raceID = strings.TrimSpace(raceID)
	participantID = strings.TrimSpace(participantID)
	if raceID == "" || participantID == "" {
		return "", fmt.Errorf("race_id and participant_id are required")
	}
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}

	url := fmt.Sprintf("%s/api/races/%s/participants/%s/tags", a.BaseURL, raceID, participantID)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	for k, vals := range a.WSHeaders() {
		for _, v := range vals {
			req.Header.Set(k, v)
		}
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("fetch tags failed: %s", strings.TrimSpace(string(raw)))
	}

	var tags []ParticipantTag
	if err := json.Unmarshal(raw, &tags); err != nil {
		return "", err
	}
	for _, tag := range tags {
		if uid := strings.TrimSpace(tag.TagUID); uid != "" {
			return strings.ToLower(uid), nil
		}
	}
	return "", fmt.Errorf("no active tag for participant")
}
