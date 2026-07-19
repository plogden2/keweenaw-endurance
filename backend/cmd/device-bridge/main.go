package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
	"github.com/keweenaw-endurance/backend/internal/bridge"
	"github.com/keweenaw-endurance/backend/internal/rfid"
	"github.com/keweenaw-endurance/backend/internal/services"
)

type config struct {
	HostedAPIURL      string
	BridgeToken       string
	OrganizerPIN      string
	DeviceID          string
	DataDir           string
	EventID           string
	LocalAddr         string
	PartitionSignal   string
	RFIDHardware      bool
	BridgeMock        bool
	PollInterval      time.Duration
}

func loadConfig() config {
	poll := 500 * time.Millisecond
	if v := strings.TrimSpace(os.Getenv("BRIDGE_POLL_MS")); v != "" {
		if ms, err := time.ParseDuration(v + "ms"); err == nil && ms > 0 {
			poll = ms
		}
	}

	dataDir := strings.TrimSpace(os.Getenv("BRIDGE_DATA_DIR"))
	if dataDir == "" {
		dataDir = "./bridge-data"
	}
	localAddr := strings.TrimSpace(os.Getenv("BRIDGE_LOCAL_ADDR"))
	if localAddr == "" {
		localAddr = "127.0.0.1:8091"
	}
	partitionSignal := strings.TrimSpace(os.Getenv("BRIDGE_PARTITION_SIGNAL"))
	if partitionSignal == "" {
		partitionSignal = filepath.Join(os.TempDir(), "keweenaw-bridge-partition.signal")
	}
	deviceID := strings.TrimSpace(os.Getenv("DEVICE_ID"))
	if deviceID == "" {
		deviceID = services.DefaultBridgeDeviceID
	}

	return config{
		HostedAPIURL:    strings.TrimRight(strings.TrimSpace(os.Getenv("HOSTED_API_URL")), "/"),
		BridgeToken:     strings.TrimSpace(os.Getenv("BRIDGE_TOKEN")),
		OrganizerPIN:    strings.TrimSpace(os.Getenv("ORGANIZER_PIN")),
		DeviceID:        deviceID,
		DataDir:         dataDir,
		EventID:         strings.TrimSpace(os.Getenv("EVENT_ID")),
		LocalAddr:       localAddr,
		PartitionSignal: partitionSignal,
		RFIDHardware:    strings.EqualFold(os.Getenv("RFID_HARDWARE"), "true"),
		BridgeMock:      strings.EqualFold(os.Getenv("BRIDGE_MOCK"), "true"),
		PollInterval:    poll,
	}
}

type app struct {
	cfg    config
	auth   *bridge.HostedAuth
	store  *bridge.LocalStore
	syncer *bridge.Syncer
	reader rfid.Reader
	pm3    *rfid.Proxmark3
	client *http.Client

	mu         sync.RWMutex
	conn       *websocket.Conn
	writeMu    sync.Mutex
	online     bool
	syncing    bool
	mode       bridge.ConnectionMode
	lastSyncAt *time.Time
	chipMemory string
	lastRead   string
	lastReadAt time.Time
}

func main() {
	cfg := loadConfig()
	if cfg.EventID == "" {
		log.Fatal("EVENT_ID is required")
	}

	auth, err := bridge.ResolveHostedAuth(cfg.HostedAPIURL, cfg.BridgeToken, cfg.OrganizerPIN, nil)
	if err != nil {
		log.Fatalf("auth: %v", err)
	}

	store, err := bridge.NewLocalStore(cfg.DataDir, cfg.EventID)
	if err != nil {
		log.Fatalf("local store: %v", err)
	}

	reader := openReader(cfg)
	a := &app{
		cfg:    cfg,
		auth:   auth,
		store:  store,
		syncer: bridge.NewSyncer(store),
		reader: reader,
		pm3:    rfid.NewProxmark3(reader),
		client: &http.Client{Timeout: 15 * time.Second},
		mode:   bridge.ModeOffline,
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	go a.runLocalHTTP(ctx)
	go a.runPollLoop(ctx)
	a.runBridgeLoop(ctx)
}

func openReader(cfg config) rfid.Reader {
	if cfg.BridgeMock {
		return rfid.NewMockReader()
	}
	if cfg.RFIDHardware {
		cli := strings.TrimSpace(os.Getenv("PROXMARK3_CLI"))
		if cli == "" {
			cli = "pm3"
		}
		return rfid.NewCLIProxmarkReader(rfid.CLIProxmarkConfig{
			CLIPath: cli,
			Port:    strings.TrimSpace(os.Getenv("PROXMARK3_PORT")),
			Enabled: true,
		})
	}
	return rfid.NewMockReader()
}

func (a *app) setMode(mode bridge.ConnectionMode) {
	a.mu.Lock()
	a.mode = mode
	a.mu.Unlock()
}

// partitioned is true when BRIDGE_PARTITION_SIGNAL points at an existing file.
// Dress-rehearsal chaos uses this to cut bridge→hosted while loopback HTTP + poll continue.
func (a *app) partitioned() bool {
	p := strings.TrimSpace(a.cfg.PartitionSignal)
	if p == "" {
		return false
	}
	_, err := os.Stat(p)
	return err == nil
}

func (a *app) snapshotStatus() map[string]any {
	a.mu.RLock()
	defer a.mu.RUnlock()

	out := map[string]any{
		"connected":     a.online,
		"pending_count": a.store.PendingCount(),
		"syncing":       a.syncing,
		"mode":          string(a.mode),
		"chip_memory":   a.chipMemory,
		"device_id":     a.cfg.DeviceID,
		"event_id":      a.cfg.EventID,
		"csv_path":      a.store.CSVPath(),
		"pending_path":  a.store.PendingPath(),
	}
	if a.lastSyncAt != nil {
		out["last_sync_at"] = a.lastSyncAt.UTC().Format(time.RFC3339)
	}
	return out
}

func (a *app) runLocalHTTP(ctx context.Context) {
	mux := http.NewServeMux()
	mux.HandleFunc("/status", a.handleStatus)
	mux.HandleFunc("/write-tag", a.handleWriteTag)

	srv := &http.Server{
		Addr:    a.cfg.LocalAddr,
		Handler: mux,
	}
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	}()

	log.Printf("local bridge HTTP listening on http://%s (status, write-tag)", a.cfg.LocalAddr)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Printf("local HTTP stopped: %v", err)
	}
}

func (a *app) handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(a.snapshotStatus())
}

type writeTagRequest struct {
	ParticipantID string `json:"participant_id"`
	RaceID        string `json:"race_id"`
	LogicalUUID   string `json:"logical_uuid"`
}

func (a *app) handleWriteTag(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		http.Error(w, "read body failed", http.StatusBadRequest)
		return
	}

	var req writeTagRequest
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	logicalUUID, err := a.resolveLogicalUUID(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := a.writeChip(logicalUUID); err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"logical_uuid": logicalUUID,
		"ok":           true,
	})
}

func (a *app) resolveLogicalUUID(req writeTagRequest) (string, error) {
	if uid := strings.TrimSpace(strings.ToLower(req.LogicalUUID)); uid != "" {
		return uid, nil
	}
	if req.ParticipantID == "" {
		return "", fmt.Errorf("logical_uuid or participant_id is required")
	}
	if req.RaceID == "" {
		return "", fmt.Errorf("race_id is required when using participant_id offline")
	}
	return a.auth.FetchLogicalUUID(a.client, req.RaceID, req.ParticipantID)
}

func (a *app) writeChip(logicalUUID string) error {
	if err := a.pm3.WriteLogicalUUID(logicalUUID); err != nil {
		return err
	}
	a.mu.Lock()
	a.chipMemory = logicalUUID
	a.mu.Unlock()
	return nil
}

func (a *app) runPollLoop(ctx context.Context) {
	ticker := time.NewTicker(a.cfg.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			a.pollOnce()
		}
	}
}

func (a *app) pollOnce() {
	logicalUUID, err := a.pm3.Poll()
	if err != nil || logicalUUID == "" {
		return
	}
	logicalUUID = strings.ToLower(strings.TrimSpace(logicalUUID))

	a.mu.Lock()
	a.chipMemory = logicalUUID
	// Match hosted finish cooldown so offline pending doesn't multiply one
	// physical presence into dozens of flush reads.
	debounce := 2 * time.Second
	if a.partitioned() || !a.online {
		debounce = 60 * time.Second
	}
	if logicalUUID == a.lastRead && time.Since(a.lastReadAt) < debounce {
		a.mu.Unlock()
		return
	}
	a.lastRead = logicalUUID
	a.lastReadAt = time.Now()
	online := a.online
	conn := a.conn
	a.mu.Unlock()

	// Partition signal must win over a stale online flag / half-closed WS —
	// otherwise in-flight polls keep scoring on hosted during dress-rehearsal outages.
	if a.partitioned() || !online || conn == nil {
		lap := bridge.PendingLap{
			LogicalUUID: logicalUUID,
			TS:          time.Now().UTC(),
			DeviceID:    a.cfg.DeviceID,
		}
		if err := a.store.EnqueueLap(lap); err != nil {
			log.Printf("offline enqueue failed: %v", err)
			return
		}
		a.setMode(bridge.ModeOffline)
		a.publishStatus()
		log.Printf("offline lap queued logical_uuid=%s pending=%d", logicalUUID, a.store.PendingCount())
		return
	}

	msg := services.BridgeMessage{
		Type:        "read",
		LogicalUUID: logicalUUID,
		TS:          time.Now().UTC().Format(time.RFC3339),
	}
	a.writeMu.Lock()
	err = conn.WriteJSON(msg)
	a.writeMu.Unlock()
	if err != nil {
		log.Printf("poll read send failed: %v", err)
		a.handleDisconnect()
	}
}

func (a *app) runBridgeLoop(ctx context.Context) {
	backoff := time.Second
	for {
		select {
		case <-ctx.Done():
			a.handleDisconnect()
			return
		default:
		}

		if a.partitioned() {
			a.handleDisconnect()
			a.setMode(bridge.ModeOffline)
			select {
			case <-ctx.Done():
				return
			case <-time.After(time.Second):
			}
			continue
		}

		if err := a.connectAndServe(ctx); err != nil {
			log.Printf("bridge disconnected: %v", err)
		}
		a.handleDisconnect()

		select {
		case <-ctx.Done():
			return
		case <-time.After(backoff):
		}
		if backoff < 30*time.Second {
			backoff *= 2
		}
	}
}

func (a *app) connectAndServe(ctx context.Context) error {
	wsURL, err := bridge.BridgeWebSocketURL(a.cfg.HostedAPIURL, a.cfg.DeviceID)
	if err != nil {
		return err
	}

	dialer := websocket.Dialer{HandshakeTimeout: 10 * time.Second}
	conn, resp, err := dialer.Dial(wsURL, a.auth.WSHeaders())
	if err != nil {
		if resp != nil {
			resp.Body.Close()
		}
		return err
	}
	if resp != nil {
		resp.Body.Close()
	}
	defer conn.Close()

	a.mu.Lock()
	a.conn = conn
	a.online = true
	a.mu.Unlock()

	log.Printf("bridge connected device_id=%s", a.cfg.DeviceID)

	if err := a.flushPending(conn); err != nil {
		log.Printf("initial flush failed: %v", err)
	}
	a.publishStatus()

	errCh := make(chan error, 1)
	go func() {
		for {
			var msg services.BridgeMessage
			if err := conn.ReadJSON(&msg); err != nil {
				errCh <- err
				return
			}
			if err := a.handleWSMessage(conn, &msg); err != nil {
				log.Printf("bridge message error: %v", err)
			}
		}
	}()

	partitionCh := make(chan struct{}, 1)
	go func() {
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if a.partitioned() {
					select {
					case partitionCh <- struct{}{}:
					default:
					}
					return
				}
			}
		}
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errCh:
		return err
	case <-partitionCh:
		return fmt.Errorf("partition signal active")
	}
}

func (a *app) handleWSMessage(conn *websocket.Conn, msg *services.BridgeMessage) error {
	if msg == nil {
		return nil
	}
	switch msg.Type {
	case "write":
		ok := true
		errMsg := ""
		if err := a.writeChip(strings.TrimSpace(msg.LogicalUUID)); err != nil {
			ok = false
			errMsg = err.Error()
		}
		return bridge.SendWriteAck(conn, &a.writeMu, msg.RequestID, ok, errMsg)
	default:
		return nil
	}
}

func (a *app) flushPending(conn *websocket.Conn) error {
	pending := a.store.PendingCount()
	if pending == 0 {
		a.mu.Lock()
		a.syncing = false
		a.mode = bridge.ModeOnlineSynced
		now := time.Now().UTC()
		a.lastSyncAt = &now
		a.mu.Unlock()
		return nil
	}

	a.mu.Lock()
	a.syncing = true
	a.mode = bridge.ModeSyncing
	a.mu.Unlock()
	_ = bridge.SendStatus(conn, &a.writeMu, pending, true, a.lastSyncAt)

	sender := bridge.NewWSReadSender(conn, &a.writeMu)
	n, err := a.syncer.Flush(sender)
	if err != nil {
		a.mu.Lock()
		a.syncing = false
		if a.store.PendingCount() > 0 {
			a.mode = bridge.ModeOffline
		}
		a.mu.Unlock()
		a.publishStatus()
		return err
	}

	now := time.Now().UTC()
	a.mu.Lock()
	a.syncing = false
	a.lastSyncAt = &now
	if a.store.PendingCount() == 0 {
		a.mode = bridge.ModeOnlineSynced
	} else {
		a.mode = bridge.ModeOffline
	}
	a.mu.Unlock()

	a.publishStatus()
	log.Printf("flushed %d pending laps", n)
	return nil
}

func (a *app) publishStatus() {
	a.mu.RLock()
	conn := a.conn
	online := a.online
	pending := a.store.PendingCount()
	syncing := a.syncing
	lastSync := a.lastSyncAt
	a.mu.RUnlock()

	if !online || conn == nil {
		return
	}
	_ = bridge.SendStatus(conn, &a.writeMu, pending, syncing, lastSync)
}

func (a *app) handleDisconnect() {
	a.mu.Lock()
	if a.conn != nil {
		_ = a.conn.Close()
		a.conn = nil
	}
	a.online = false
	a.syncing = false
	if a.store.PendingCount() > 0 {
		a.mode = bridge.ModeOffline
	}
	a.mu.Unlock()
	a.publishStatus()
}
