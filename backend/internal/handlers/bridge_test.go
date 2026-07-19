package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/keweenaw-endurance/backend/internal/config"
	"github.com/keweenaw-endurance/backend/internal/middleware"
	"github.com/keweenaw-endurance/backend/internal/models"
	"github.com/keweenaw-endurance/backend/internal/rfid"
	"github.com/keweenaw-endurance/backend/internal/services"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func bridgeTestConfig() *config.Config {
	cfg := testHandlerConfig()
	cfg.RFID.Hardware = false
	cfg.RFID.BridgeToken = "bridge-secret"
	cfg.RFID.BridgeDeviceID = "laptop-finish-1"
	return cfg
}

func setupBridgeHandlerTest(t *testing.T) (*gin.Engine, *services.Services, *httptest.Server) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&models.Event{},
		&models.Race{},
		&models.Participant{},
		&models.TimingCheckpoint{},
		&models.TimingRecord{},
		&models.Category{},
		&models.RFIDTagAssociation{},
		&models.ReaderStation{},
	))

	svc := services.NewServicesWithReader(db, bridgeTestConfig(), rfid.NewMockReader())
	h := NewHandlers(svc)

	auth := middleware.JWTAuth(svc.Auth)
	adminOnly := []gin.HandlerFunc{auth, middleware.RequireRoles(services.RoleAdmin, services.RoleOwner)}

	router := gin.New()
	api := router.Group("/api")
	{
		api.POST("/auth/pin", h.ExchangePIN)
		api.POST("/rfid/write-tag", append(adminOnly, h.WriteRFIDTag)...)
		api.GET("/rfid/bridge", h.BridgeWebSocket)
		api.GET("/rfid/bridge/status", h.GetBridgeStatus)
		api.PUT("/stations/current", append(adminOnly, h.PutCurrentStation)...)
	}

	server := httptest.NewServer(router)
	t.Cleanup(server.Close)
	return router, svc, server
}

func TestBridgeWebSocket_RejectsMissingAuth(t *testing.T) {
	_, _, server := setupBridgeHandlerTest(t)
	wsURL := "ws" + server.URL[len("http"):] + "/api/rfid/bridge?device_id=laptop-finish-1"
	_, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.Error(t, err)
	if resp != nil {
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
		resp.Body.Close()
	}
}

func TestBridgeWebSocket_AcceptsBridgeToken(t *testing.T) {
	_, svc, server := setupBridgeHandlerTest(t)
	wsURL := "ws" + server.URL[len("http"):] + "/api/rfid/bridge?device_id=laptop-finish-1"
	header := http.Header{}
	header.Set("X-Bridge-Token", "bridge-secret")
	conn, resp, err := websocket.DefaultDialer.Dial(wsURL, header)
	require.NoError(t, err)
	if resp != nil {
		resp.Body.Close()
	}
	defer conn.Close()

	assert.True(t, svc.Bridge.IsConnected("laptop-finish-1"))
}

func TestBridgeWebSocket_WriteTagRoundTrip(t *testing.T) {
	_, svc, server := setupBridgeHandlerTest(t)

	event, err := svc.Events.CreateEvent(&models.Event{
		Name: "Bridge Event", EventDate: time.Now().AddDate(0, 1, 0),
	})
	require.NoError(t, err)
	race, err := svc.Races.CreateRace(&models.Race{
		EventID: event.ID, Name: "Race", RaceType: "time_based", DistanceKm: 10, Status: "active",
	})
	require.NoError(t, err)
	participant, err := svc.Participants.CreateParticipant(&models.Participant{
		RaceID: race.ID, BibNumber: "11", FirstName: "Bridge", LastName: "Tag",
	})
	require.NoError(t, err)

	wsURL := "ws" + server.URL[len("http"):] + "/api/rfid/bridge?device_id=laptop-finish-1"
	header := http.Header{}
	header.Set("X-Bridge-Token", "bridge-secret")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, header)
	require.NoError(t, err)
	defer conn.Close()

	ackDone := make(chan struct{})
	go func() {
		var msg services.BridgeMessage
		require.NoError(t, conn.ReadJSON(&msg))
		assert.Equal(t, "write", msg.Type)
		ok := true
		require.NoError(t, conn.WriteJSON(services.BridgeMessage{
			Type:      "write_ack",
			RequestID: msg.RequestID,
			OK:        &ok,
		}))
		close(ackDone)
	}()

	body := map[string]string{"participant_id": participant.ID.Short()}
	payload, _ := json.Marshal(body)
	req, err := http.NewRequest(http.MethodPost, server.URL+"/api/rfid/write-tag", bytes.NewReader(payload))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", adminAuthHeader(t, svc))
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	select {
	case <-ackDone:
	case <-time.After(3 * time.Second):
		t.Fatal("write_ack not sent")
	}
}

func TestGetBridgeStatus(t *testing.T) {
	_, svc, server := setupBridgeHandlerTest(t)

	wsURL := "ws" + server.URL[len("http"):] + "/api/rfid/bridge?device_id=laptop-finish-1"
	header := http.Header{}
	header.Set("X-Bridge-Token", "bridge-secret")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, header)
	require.NoError(t, err)
	defer conn.Close()

	req, err := http.NewRequest(http.MethodGet, server.URL+"/api/rfid/bridge/status?device_id=laptop-finish-1", nil)
	require.NoError(t, err)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var status services.BridgeStatus
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&status))
	assert.True(t, status.Connected)
	assert.True(t, svc.Bridge.IsConnected("laptop-finish-1"))
}

func TestBridgeRead_DoesNotDoubleScoreViaInjectFanout(t *testing.T) {
	router, svc, server := setupBridgeHandlerTest(t)
	_, _, tagUID := seedBridgeReadFixture(t, svc)

	// Simulate the reader UI subscribed to /api/rfid/stream — historically
	// InjectTag fan-out caused a second ProcessScan for the same physical tap.
	streamSub := svc.RFID.SubscribeTagReads(8)
	_ = router

	wsURL := "ws" + server.URL[len("http"):] + "/api/rfid/bridge?device_id=laptop-finish-1"
	header := http.Header{}
	header.Set("X-Bridge-Token", "bridge-secret")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, header)
	require.NoError(t, err)
	defer conn.Close()

	ts := time.Now().UTC().Format(time.RFC3339Nano)
	require.NoError(t, conn.WriteJSON(services.BridgeMessage{
		Type:        "read",
		LogicalUUID: tagUID,
		TS:          ts,
	}))

	deadline := time.Now().Add(2 * time.Second)
	var sawScanResult bool
	for time.Now().Before(deadline) {
		select {
		case ev := <-streamSub:
			if ev.Type == "scan_result" {
				sawScanResult = true
			}
			assert.NotEqual(t, "tag_read", ev.Type, "bridge must not fan out raw tag_read (re-scores via reader UI)")
		default:
			time.Sleep(20 * time.Millisecond)
		}
	}
	assert.True(t, sawScanResult, "reader UI should receive scan_result for popup feedback")

	var count int64
	require.NoError(t, svc.DB.Model(&models.TimingRecord{}).
		Where("record_type = ?", "rfid_lap").
		Count(&count).Error)
	assert.Equal(t, int64(1), count, "one physical bridge read must score exactly one lap")
}

func TestBridgeRead_SkipsDuplicateSourceLapID(t *testing.T) {
	_, svc, server := setupBridgeHandlerTest(t)
	_, _, tagUID := seedBridgeReadFixture(t, svc)

	wsURL := "ws" + server.URL[len("http"):] + "/api/rfid/bridge?device_id=laptop-finish-1"
	header := http.Header{}
	header.Set("X-Bridge-Token", "bridge-secret")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, header)
	require.NoError(t, err)
	defer conn.Close()

	sourceLapID := "b7040fdd-b6cb-4005-a38c-c124dfcf7cc2"
	ts := time.Now().UTC().Format(time.RFC3339)
	read := services.BridgeMessage{
		Type:        "read",
		LogicalUUID: tagUID,
		SourceLapID: sourceLapID,
		TS:          ts,
	}

	require.NoError(t, conn.WriteJSON(read))
	time.Sleep(200 * time.Millisecond)
	require.NoError(t, conn.WriteJSON(read))
	time.Sleep(200 * time.Millisecond)

	var count int64
	require.NoError(t, svc.DB.Model(&models.TimingRecord{}).
		Where("id = ? AND record_type = ?", sourceLapID, "rfid_lap").
		Count(&count).Error)
	assert.Equal(t, int64(1), count)
}

func seedBridgeReadFixture(t *testing.T, svc *services.Services) (*models.Event, *models.Race, string) {
	t.Helper()

	event, err := svc.Events.CreateEvent(&models.Event{
		Name: "Bridge Read Event", EventDate: time.Now().AddDate(0, 1, 0), Status: "upcoming",
	})
	require.NoError(t, err)
	race, err := svc.Races.CreateRace(&models.Race{
		EventID: event.ID, Name: "12 Hour", RaceType: "lap_based", DurationMinutes: 720,
		StartTime: time.Now().Add(-time.Hour), Status: "active",
	})
	require.NoError(t, err)
	_, err = svc.Checkpoints.CreateCheckpoint(&models.TimingCheckpoint{
		RaceID: race.ID, Name: "Finish", CheckpointType: "finish", IsActive: true,
	})
	require.NoError(t, err)
	participant, err := svc.Participants.CreateParticipant(&models.Participant{
		RaceID: race.ID, BibNumber: "7", FirstName: "Replay", LastName: "Runner", Status: "started",
	})
	require.NoError(t, err)

	tagUID := "9fe78eeb-a21c-594a-acc2-7e1efe378201"
	_, err = svc.RFID.AssociateTag(participant.ID.UUID(), tagUID)
	require.NoError(t, err)

	_, err = svc.Stations.PutCurrent(&services.StationConfigInput{
		EventID:  event.ID.UUID(),
		Mode:     "finish",
		DeviceID: "laptop-finish-1",
		Name:     "Finish",
	})
	require.NoError(t, err)

	return event, race, tagUID
}
