package bridgeapp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/keweenaw-endurance/backend/internal/bridge"
	"github.com/keweenaw-endurance/backend/internal/rfid"
	"github.com/keweenaw-endurance/backend/internal/services"
)

// Status is a snapshot for GUI / local HTTP.
type Status struct {
	Connected    bool       `json:"connected"`
	PendingCount int        `json:"pending_count"`
	Syncing      bool       `json:"syncing"`
	Mode         string     `json:"mode"`
	ChipMemory   string     `json:"chip_memory"`
	DeviceID     string     `json:"device_id"`
	EventID      string     `json:"event_id"`
	CSVPath      string     `json:"csv_path"`
	PendingPath  string     `json:"pending_path"`
	LastSyncAt   *time.Time `json:"last_sync_at,omitempty"`
	LastRead     string     `json:"last_read,omitempty"`
	LastReadAt   *time.Time `json:"last_read_at,omitempty"`
	LastTapUUID     string     `json:"last_tap_uuid,omitempty"`
	LastTapAt       *time.Time `json:"last_tap_at,omitempty"`
	LastTapResult   string     `json:"last_tap_result,omitempty"`
	LastTapName     string     `json:"last_tap_name,omitempty"`
	LastTapBib      string     `json:"last_tap_bib,omitempty"`
	LastTapRaceID   string     `json:"last_tap_race_id,omitempty"`
	LastTapRaceName string     `json:"last_tap_race_name,omitempty"`
	// Last write-tag outcome (website Program → local bridge). Separate from taps.
	LastWriteOK      bool       `json:"last_write_ok"`
	LastWriteAt      *time.Time `json:"last_write_at,omitempty"`
	LastWriteUUID    string     `json:"last_write_uuid,omitempty"`
	LastWriteBib     string     `json:"last_write_bib,omitempty"`
	LastWriteName    string     `json:"last_write_name,omitempty"`
	LastWriteMessage string     `json:"last_write_message,omitempty"`
	WriteOnly    bool       `json:"write_only"`
	Running      bool       `json:"running"`
	LastError    string     `json:"last_error,omitempty"`
}

// App owns Proxmark + hosted bridge loops.
type App struct {
	cfg    Config
	auth   *bridge.HostedAuth
	store  *bridge.LocalStore
	syncer *bridge.Syncer
	reader rfid.Reader
	pm3    *rfid.Proxmark3
	client *http.Client
	roster *bridge.RosterCache

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
	lastTapUUID     string
	lastTapAt       time.Time
	lastTapResult   string
	lastTapName     string
	lastTapBib      string
	lastTapRaceID   string
	lastTapRaceName string
	lastWriteOK      bool
	lastWriteAt      time.Time
	lastWriteUUID    string
	lastWriteBib     string
	lastWriteName    string
	lastWriteMessage string
	lastError  string
	running    bool

	runCancel context.CancelFunc
	runDone   chan struct{}
}

// New constructs an App (does not start loops).
func New(cfg Config) (*App, error) {
	normalizeConfig(&cfg)
	if cfg.EventID == "" {
		return nil, fmt.Errorf("EVENT_ID is required")
	}
	if cfg.HostedAPIURL == "" {
		return nil, fmt.Errorf("HOSTED_API_URL is required")
	}
	if cfg.BridgeToken == "" && cfg.OrganizerPIN == "" {
		return nil, fmt.Errorf("BRIDGE_TOKEN or ORGANIZER_PIN is required")
	}

	auth, err := bridge.ResolveHostedAuth(cfg.HostedAPIURL, cfg.BridgeToken, cfg.OrganizerPIN, nil)
	if err != nil {
		return nil, fmt.Errorf("auth: %w", err)
	}
	// Prefer PIN JWT for HTTP admin routes even when bridge token is set.
	if cfg.OrganizerPIN != "" {
		_ = bridge.EnsureBearer(auth, nil, cfg.OrganizerPIN)
	}

	store, err := bridge.NewLocalStore(cfg.DataDir, cfg.EventID)
	if err != nil {
		return nil, fmt.Errorf("local store: %w", err)
	}

	reader := openReader(cfg)
	app := &App{
		cfg:    cfg,
		auth:   auth,
		store:  store,
		syncer: bridge.NewSyncer(store),
		reader: reader,
		pm3:    rfid.NewProxmark3(reader),
		client: &http.Client{Timeout: 15 * time.Second},
		roster: bridge.NewRosterCache(),
		mode:   bridge.ModeOffline,
	}
	app.applyWriteOnlyToReader(cfg.WriteOnly)
	app.applyHFGainToReader(cfg.HFGain)
	return app, nil
}

// Config returns a copy of the active config.
func (a *App) Config() Config {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.cfg
}

// SetWriteOnly enables/disables automatic tap recording without restarting.
// Reads still update last-tap display; writes and manual entry stay available.
func (a *App) SetWriteOnly(enabled bool) {
	if a == nil {
		return
	}
	a.mu.Lock()
	was := a.cfg.WriteOnly
	a.cfg.WriteOnly = enabled
	conn := a.conn
	online := a.online
	a.mu.Unlock()
	a.applyWriteOnlyToReader(enabled)
	a.publishStatus()
	// Turning write-only off: flush any laps that were held in the queue.
	if was && !enabled && online && conn != nil && !a.partitioned() {
		if err := a.flushPending(conn); err != nil {
			log.Printf("flush after leaving write-only: %v", err)
		}
	}
}

// SetHFGain updates the Proxmark HF antenna gain (1–63) without restarting.
func (a *App) SetHFGain(gain int) {
	if a == nil {
		return
	}
	gain = rfid.ClampHFGain(gain)
	a.mu.Lock()
	a.cfg.HFGain = gain
	a.mu.Unlock()
	a.applyHFGainToReader(gain)
	a.publishStatus()
}

func (a *App) applyHFGainToReader(gain int) {
	if cli, ok := a.reader.(*rfid.CLIProxmarkReader); ok {
		cli.SetHFGain(gain)
	}
}

func (a *App) applyWriteOnlyToReader(writeOnly bool) {
	if cli, ok := a.reader.(*rfid.CLIProxmarkReader); ok {
		cli.SetBeepEnabled(!writeOnly)
	}
}

// WriteOnly reports whether automatic tap recording is suppressed.
func (a *App) WriteOnly() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.cfg.WriteOnly
}

// Start begins local HTTP, poll, and bridge loops until Stop or ctx done.
func (a *App) Start(parent context.Context) error {
	a.mu.Lock()
	if a.running {
		a.mu.Unlock()
		return fmt.Errorf("bridge already running")
	}
	ctx, cancel := context.WithCancel(parent)
	a.runCancel = cancel
	a.runDone = make(chan struct{})
	a.running = true
	a.lastError = ""
	a.mu.Unlock()

	go func() {
		defer close(a.runDone)
		defer func() {
			a.mu.Lock()
			a.running = false
			a.runCancel = nil
			a.mu.Unlock()
		}()
		rfid.PrewarmTapBeep()
		go a.runLocalHTTP(ctx)
		go a.runPollLoop(ctx)
		a.runBridgeLoop(ctx)
	}()

	go func() {
		if err := a.RefreshRoster(); err != nil {
			log.Printf("roster refresh: %v", err)
		}
	}()
	return nil
}

// Stop cancels background loops and waits briefly for exit.
func (a *App) Stop() {
	a.mu.Lock()
	cancel := a.runCancel
	done := a.runDone
	a.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	if done != nil {
		select {
		case <-done:
		case <-time.After(5 * time.Second):
		}
	}
	a.handleDisconnect()
}

// Running reports whether Start loops are active.
func (a *App) Running() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.running
}

// StatusSnapshot returns operator-facing state.
func (a *App) StatusSnapshot() Status {
	a.mu.RLock()
	defer a.mu.RUnlock()
	out := Status{
		Connected:    a.online,
		PendingCount: a.store.PendingCount(),
		Syncing:      a.syncing,
		Mode:         string(a.mode),
		ChipMemory:   a.chipMemory,
		DeviceID:     a.cfg.DeviceID,
		EventID:      a.cfg.EventID,
		CSVPath:      a.store.CSVPath(),
		PendingPath:  a.store.PendingPath(),
		LastRead:        a.lastRead,
		LastTapUUID:     a.lastTapUUID,
		LastTapResult:   a.lastTapResult,
		LastTapName:     a.lastTapName,
		LastTapBib:      a.lastTapBib,
		LastTapRaceID:   a.lastTapRaceID,
		LastTapRaceName: a.lastTapRaceName,
		LastWriteOK:      a.lastWriteOK,
		LastWriteUUID:    a.lastWriteUUID,
		LastWriteBib:     a.lastWriteBib,
		LastWriteName:    a.lastWriteName,
		LastWriteMessage: a.lastWriteMessage,
		WriteOnly:        a.cfg.WriteOnly,
		Running:          a.running,
		LastError:        a.lastError,
	}
	if a.lastSyncAt != nil {
		t := *a.lastSyncAt
		out.LastSyncAt = &t
	}
	if !a.lastReadAt.IsZero() {
		t := a.lastReadAt
		out.LastReadAt = &t
	}
	if !a.lastTapAt.IsZero() {
		t := a.lastTapAt
		out.LastTapAt = &t
	}
	if !a.lastWriteAt.IsZero() {
		t := a.lastWriteAt
		out.LastWriteAt = &t
	}
	return out
}

// RefreshRoster caches bib→UUID (+ names) for manual entry and tap labels.
// Empty RaceID refreshes the whole event (All races mode).
func (a *App) RefreshRoster() error {
	if err := bridge.EnsureBearer(a.auth, a.client, a.cfg.OrganizerPIN); err != nil {
		return err
	}
	if strings.TrimSpace(a.cfg.RaceID) == "" {
		return a.roster.RefreshEvent(a.auth, a.client, a.cfg.EventID)
	}
	return a.roster.RefreshRace(a.auth, a.client, a.cfg.RaceID, "", a.cfg.CheckpointID)
}

// ManualEntry records a lap online, or queues offline via roster UUID.
// raceOverride empty = auto-resolve across the event roster (All races).
func (a *App) ManualEntry(bib string, ts time.Time) error {
	return a.ManualEntryInRace(bib, ts, a.cfg.RaceID)
}

// ManualEntryInRace records a lap for an optional race override.
func (a *App) ManualEntryInRace(bib string, ts time.Time, raceOverride string) error {
	bib = strings.TrimSpace(bib)
	if bib == "" {
		return fmt.Errorf("bib number is required")
	}
	if ts.IsZero() {
		ts = time.Now().UTC()
	}
	raceOverride = strings.TrimSpace(raceOverride)

	entry, err := a.resolveManualEntry(bib, raceOverride)
	if err != nil {
		return err
	}

	a.mu.RLock()
	online := a.online && !a.partitioned()
	a.mu.RUnlock()

	if online {
		if err := bridge.EnsureBearer(a.auth, a.client, a.cfg.OrganizerPIN); err != nil {
			return err
		}
		checkpointID := entry.CheckpointID
		if checkpointID == "" {
			checkpointID = a.cfg.CheckpointID
		}
		if entry.RaceID == "" || checkpointID == "" {
			return fmt.Errorf("could not resolve race/checkpoint for bib %s", bib)
		}
		err := bridge.PostManualEntry(a.auth, a.client, bridge.ManualEntryRequest{
			RaceID:       entry.RaceID,
			CheckpointID: checkpointID,
			BibNumber:    bib,
			Timestamp:    ts,
			DeviceID:     a.cfg.DeviceID,
		})
		if err == nil {
			_ = a.RefreshRoster()
			a.setLastTapFromRoster(entry, "lap")
		}
		return err
	}

	if entry.LogicalUUID == "" {
		return fmt.Errorf("offline: no cached RFID UUID for bib %s (connect once to refresh roster)", bib)
	}
	lap := bridge.PendingLap{
		LogicalUUID: entry.LogicalUUID,
		TS:          ts.UTC(),
		DeviceID:    a.cfg.DeviceID,
	}
	if err := a.store.EnqueueLap(lap); err != nil {
		return err
	}
	a.setLastTapFromRoster(entry, "queued")
	a.setMode(bridge.ModeOffline)
	a.publishStatus()
	return nil
}

func (a *App) resolveManualEntry(bib, raceOverride string) (bridge.RosterEntry, error) {
	entries := a.roster.EntriesForBib(bib)
	if len(entries) == 0 {
		if err := a.RefreshRoster(); err == nil {
			entries = a.roster.EntriesForBib(bib)
		}
	}
	if raceOverride != "" {
		var filtered []bridge.RosterEntry
		for _, e := range entries {
			if e.RaceID == raceOverride || strings.HasSuffix(e.RaceID, raceOverride) || strings.HasSuffix(raceOverride, e.RaceID) {
				filtered = append(filtered, e)
			}
		}
		entries = filtered
		if len(entries) == 0 && a.cfg.RaceID == raceOverride && a.cfg.CheckpointID != "" {
			// Online path can still post with config when roster miss but race forced.
			return bridge.RosterEntry{Bib: bib, RaceID: raceOverride, CheckpointID: a.cfg.CheckpointID}, nil
		}
	}
	switch len(entries) {
	case 0:
		return bridge.RosterEntry{}, fmt.Errorf("unknown bib %s", bib)
	case 1:
		return entries[0], nil
	default:
		names := make([]string, 0, len(entries))
		for _, e := range entries {
			if e.RaceName != "" {
				names = append(names, e.RaceName)
			} else {
				names = append(names, e.RaceID)
			}
		}
		return bridge.RosterEntry{}, fmt.Errorf("ambiguous bib %s — select a race (%s)", bib, strings.Join(names, ", "))
	}
}

func (a *App) setLastTapFromRoster(entry bridge.RosterEntry, result string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.lastTapUUID = entry.LogicalUUID
	a.lastTapAt = time.Now().UTC()
	a.lastTapResult = result
	a.lastTapName = entry.Name
	a.lastTapBib = entry.Bib
	a.lastTapRaceID = entry.RaceID
	a.lastTapRaceName = entry.RaceName
}

func openReader(cfg Config) rfid.Reader {
	if cfg.BridgeMock {
		return rfid.NewMockReader()
	}
	if cfg.RFIDHardware {
		cli := cfg.ProxmarkCLI
		if cli == "" {
			cli = "pm3"
		}
		return rfid.NewCLIProxmarkReader(rfid.CLIProxmarkConfig{
			CLIPath: cli,
			Port:    cfg.ProxmarkPort,
			Enabled: true,
			HFGain:  cfg.HFGain,
		})
	}
	return rfid.NewMockReader()
}

func (a *App) setMode(mode bridge.ConnectionMode) {
	a.mu.Lock()
	a.mode = mode
	a.mu.Unlock()
}

func (a *App) setLastError(err error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if err == nil {
		a.lastError = ""
		return
	}
	a.lastError = err.Error()
}

func (a *App) partitioned() bool {
	p := strings.TrimSpace(a.cfg.PartitionSignal)
	if p == "" {
		return false
	}
	_, err := os.Stat(p)
	return err == nil
}

func (a *App) snapshotStatusMap() map[string]any {
	st := a.StatusSnapshot()
	out := map[string]any{
		"connected":     st.Connected,
		"pending_count": st.PendingCount,
		"syncing":       st.Syncing,
		"mode":          st.Mode,
		"chip_memory":   st.ChipMemory,
		"device_id":     st.DeviceID,
		"event_id":      st.EventID,
		"csv_path":      st.CSVPath,
		"pending_path":  st.PendingPath,
		"running":       st.Running,
		"write_only":    st.WriteOnly,
	}
	if st.LastSyncAt != nil {
		out["last_sync_at"] = st.LastSyncAt.UTC().Format(time.RFC3339)
	}
	if st.LastRead != "" {
		out["last_read"] = st.LastRead
	}
	if st.LastReadAt != nil {
		out["last_read_at"] = st.LastReadAt.UTC().Format(time.RFC3339)
	}
	if st.LastTapUUID != "" {
		out["last_tap_uuid"] = st.LastTapUUID
	}
	if st.LastTapAt != nil {
		out["last_tap_at"] = st.LastTapAt.UTC().Format(time.RFC3339)
	}
	if st.LastTapResult != "" {
		out["last_tap_result"] = st.LastTapResult
	}
	if st.LastTapName != "" {
		out["last_tap_name"] = st.LastTapName
	}
	if st.LastTapBib != "" {
		out["last_tap_bib"] = st.LastTapBib
	}
	if st.LastTapRaceID != "" {
		out["last_tap_race_id"] = st.LastTapRaceID
	}
	if st.LastTapRaceName != "" {
		out["last_tap_race_name"] = st.LastTapRaceName
	}
	if st.LastWriteAt != nil {
		out["last_write_ok"] = st.LastWriteOK
		out["last_write_at"] = st.LastWriteAt.UTC().Format(time.RFC3339)
		if st.LastWriteUUID != "" {
			out["last_write_uuid"] = st.LastWriteUUID
		}
		if st.LastWriteBib != "" {
			out["last_write_bib"] = st.LastWriteBib
		}
		if st.LastWriteName != "" {
			out["last_write_name"] = st.LastWriteName
		}
		if st.LastWriteMessage != "" {
			out["last_write_message"] = st.LastWriteMessage
		}
	}
	if st.LastError != "" {
		out["last_error"] = st.LastError
	}
	return out
}

func (a *App) runLocalHTTP(ctx context.Context) {
	mux := http.NewServeMux()
	mux.HandleFunc("/status", a.handleStatus)
	mux.HandleFunc("/write-tag", a.handleWriteTag)
	mux.HandleFunc("/beep", a.handleBeep)

	srv := &http.Server{
		Addr:    a.cfg.LocalAddr,
		Handler: withLocalCORS(mux),
	}
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	}()

	log.Printf("local bridge HTTP listening on http://%s (status, write-tag, beep)", a.cfg.LocalAddr)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Printf("local HTTP stopped: %v", err)
		a.setLastError(err)
	}
}

func (a *App) handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(a.snapshotStatusMap())
}

// handleBeep plays the tap tone for audio diagnostics (always on, even write-only).
func (a *App) handleBeep(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	rfid.PlayTapBeep()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]bool{"ok": true})
}

type writeTagRequest struct {
	ParticipantID string `json:"participant_id"`
	RaceID        string `json:"race_id"`
	LogicalUUID   string `json:"logical_uuid"`
}

func (a *App) handleWriteTag(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
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
		a.recordWriteResult("", false, err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := a.writeChip(logicalUUID); err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	st := a.StatusSnapshot()

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"logical_uuid": logicalUUID,
		"ok":           true,
		"bib":          st.LastWriteBib,
		"name":         st.LastWriteName,
	})
}

func withLocalCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := strings.TrimSpace(r.Header.Get("Origin"))
		if origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.Header().Set("Vary", "Origin")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (a *App) resolveLogicalUUID(req writeTagRequest) (string, error) {
	if uid := strings.TrimSpace(strings.ToLower(req.LogicalUUID)); uid != "" {
		if _, err := uuid.Parse(uid); err == nil {
			return uid, nil
		}
		// Event Bibs UI may send the short public id; resolve via hosted catalog.
		if a.auth != nil && strings.TrimSpace(a.cfg.EventID) != "" {
			full, err := a.auth.FetchBibLogicalUUID(a.client, a.cfg.EventID, uid)
			if err == nil {
				return full, nil
			}
			return "", fmt.Errorf("logical_uuid %q: %w", uid, err)
		}
		return "", fmt.Errorf("logical_uuid must be a full UUID, got %q", uid)
	}
	if req.ParticipantID == "" {
		return "", fmt.Errorf("logical_uuid or participant_id is required")
	}
	if req.RaceID == "" {
		return "", fmt.Errorf("race_id is required when using participant_id offline")
	}
	return a.auth.FetchLogicalUUID(a.client, req.RaceID, req.ParticipantID)
}

func (a *App) writeChip(logicalUUID string) error {
	// WriteLogicalUUID cancels the continuous arm and (via killAllProxmarkHook)
	// terminates any proxmark3 still holding COM — including our own Lua arm.
	// It also does a family-specific readback before returning.
	normalized := strings.ToLower(strings.TrimSpace(logicalUUID))
	if err := a.pm3.WriteLogicalUUID(normalized); err != nil {
		// Hosted WS writes and local /write-tag both use this path — always
		// update the operator WRITE OK/FAILED banner.
		a.recordWriteResult(normalized, false, err)
		return err
	}
	// Independent quick Poll before operator success — never report WRITE OK
	// from write-exit alone (partial page writes have slipped past CLI exit 0).
	got, err := a.verifyWriteByPoll(normalized)
	if err != nil {
		a.recordWriteResult(normalized, false, err)
		return err
	}
	a.mu.Lock()
	a.chipMemory = got
	// Dress rehearsal rewrites the same chip every lap — clear debounce so the
	// post-write emit always scores.
	a.lastReadAt = time.Time{}
	a.mu.Unlock()
	a.emitRead(got, true)
	a.recordWriteResult(got, true, nil)
	return nil
}

// verifyWriteByPoll polls the antenna a few times until the chip returns want.
func (a *App) verifyWriteByPoll(want string) (string, error) {
	want = strings.ToLower(strings.TrimSpace(want))
	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		if attempt == 0 {
			time.Sleep(150 * time.Millisecond)
		} else {
			time.Sleep(250 * time.Millisecond)
		}
		got, err := a.pm3.Poll()
		got = strings.ToLower(strings.TrimSpace(got))
		if err != nil {
			lastErr = fmt.Errorf("post-write read failed: %w", err)
			continue
		}
		if got == "" {
			lastErr = fmt.Errorf("post-write read empty — hold chip on antenna and retry")
			continue
		}
		if !strings.EqualFold(got, want) {
			lastErr = fmt.Errorf("post-write read mismatch: got %s want %s", got, want)
			continue
		}
		return got, nil
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("post-write read failed")
	}
	return "", lastErr
}

// recordWriteResult updates the operator-facing write banner (bib + ok/fail).
func (a *App) recordWriteResult(logicalUUID string, ok bool, writeErr error) {
	normalized := strings.ToLower(strings.TrimSpace(logicalUUID))
	bib, name := a.lookupWriteSubject(normalized)
	msg := ""
	if ok {
		msg = "Verified read " + normalized
		a.setLastError(nil)
	} else if writeErr != nil {
		msg = writeErr.Error()
		a.setLastError(writeErr)
	}

	a.mu.Lock()
	a.lastWriteOK = ok
	a.lastWriteAt = time.Now()
	a.lastWriteUUID = normalized
	a.lastWriteBib = bib
	a.lastWriteName = name
	a.lastWriteMessage = msg
	a.mu.Unlock()
	a.publishStatus()
}

func (a *App) lookupWriteSubject(logicalUUID string) (bib, name string) {
	if logicalUUID == "" || a.roster == nil {
		return "", ""
	}
	if e, found := a.roster.EntryForUUID(logicalUUID); found {
		return e.Bib, e.Name
	}
	// Best-effort refresh so Program shows "Bib N" even if catalog wasn't loaded yet.
	if a.auth != nil && strings.TrimSpace(a.cfg.EventID) != "" {
		_ = a.roster.RefreshEvent(a.auth, a.client, a.cfg.EventID)
		if e, found := a.roster.EntryForUUID(logicalUUID); found {
			return e.Bib, e.Name
		}
	}
	return "", ""
}

func (a *App) runPollLoop(ctx context.Context) {
	// Hardware finish line: arm Proxmark continuous wait-for-card (hf 14a reader -w)
	// instead of spawn-per-tick one-shot polls that miss sub-second waves.
	if a.cfg.RFIDHardware && !a.cfg.BridgeMock {
		a.runContinuousArm(ctx)
		return
	}
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

func (a *App) runContinuousArm(ctx context.Context) {
	log.Printf("rfid: continuous arm scan (hf 14a reader -w)")
	for {
		if err := ctx.Err(); err != nil {
			return
		}
		logicalUUID, err := a.pm3.ArmScan(ctx)
		if ctx.Err() != nil {
			return
		}
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return
			}
			log.Printf("rfid arm scan: %v", err)
			select {
			case <-ctx.Done():
				return
			case <-time.After(500 * time.Millisecond):
			}
			continue
		}
		if logicalUUID == "" {
			continue
		}
		// Arm path already beeped after a successful UUID decode.
		a.emitRead(strings.ToLower(strings.TrimSpace(logicalUUID)), false)
	}
}

func (a *App) pollOnce() {
	logicalUUID, err := a.pm3.Poll()
	if err != nil || logicalUUID == "" {
		return
	}
	a.emitRead(strings.ToLower(strings.TrimSpace(logicalUUID)), true)
}

// ReadSuccessCooldown is how long the finish reader ignores the same UUID
// after a successful scan (online). Prevents double-fires while a chip rests
// on the mat without feeling sluggish between racers.
const ReadSuccessCooldown = time.Second

// readDebounce returns how long to suppress duplicate UUID taps.
// BRIDGE_READ_DEBOUNCE_MS overrides (0 = accept every successful read).
// Default: 1s online / 60s offline (offline avoids flooding the pending queue).
func (a *App) readDebounce() time.Duration {
	if v := strings.TrimSpace(os.Getenv("BRIDGE_READ_DEBOUNCE_MS")); v != "" {
		if ms, err := strconv.Atoi(v); err == nil && ms >= 0 {
			return time.Duration(ms) * time.Millisecond
		}
	}
	if a.partitioned() || !a.online {
		return 60 * time.Second
	}
	return ReadSuccessCooldown
}

func (a *App) emitRead(logicalUUID string, playBeep bool) {
	if logicalUUID == "" {
		return
	}

	a.mu.Lock()
	a.chipMemory = logicalUUID
	debounce := a.readDebounce()
	if logicalUUID == a.lastRead && time.Since(a.lastReadAt) < debounce {
		a.mu.Unlock()
		return
	}
	a.lastRead = logicalUUID
	a.lastReadAt = time.Now()
	tapAt := a.lastReadAt
	a.mu.Unlock()

	// Resolve bib outside a.mu — may refresh event bibs (includes unassigned).
	e, ok := a.roster.EntryForUUID(logicalUUID)
	if !ok {
		_, _ = a.lookupWriteSubject(logicalUUID)
		e, ok = a.roster.EntryForUUID(logicalUUID)
	}

	a.mu.Lock()
	a.lastTapUUID = logicalUUID
	a.lastTapAt = tapAt
	if ok {
		a.lastTapName = e.Name
		a.lastTapBib = e.Bib
		a.lastTapRaceID = e.RaceID
		a.lastTapRaceName = e.RaceName
	} else {
		a.lastTapName = ""
		a.lastTapBib = ""
		a.lastTapRaceID = ""
		a.lastTapRaceName = ""
	}
	// Reset until hosted scan_result confirms a scored lap (avoids stale "(lap)"
	// when the station is unarmed / server rejects the read).
	a.lastTapResult = "read"
	writeOnly := a.cfg.WriteOnly
	if writeOnly {
		a.lastTapResult = "write_only"
	}
	online := a.online
	conn := a.conn
	a.mu.Unlock()

	// write-tag / mock poll: beep here. Continuous Lua arm beeps earlier (raw READ).
	if playBeep && !writeOnly {
		rfid.PlayTapBeep()
	}

	if writeOnly {
		a.publishStatus()
		return
	}

	if a.partitioned() || !online || conn == nil {
		lap := bridge.PendingLap{
			LogicalUUID: logicalUUID,
			TS:          time.Now().UTC(),
			DeviceID:    a.cfg.DeviceID,
		}
		if err := a.store.EnqueueLap(lap); err != nil {
			log.Printf("offline enqueue failed: %v", err)
			a.setLastError(err)
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
	err := conn.WriteJSON(msg)
	a.writeMu.Unlock()
	if err != nil {
		log.Printf("poll read send failed: %v", err)
		a.setLastError(err)
		a.handleDisconnect()
	}
}

func (a *App) runBridgeLoop(ctx context.Context) {
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
			a.setLastError(err)
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

func (a *App) connectAndServe(ctx context.Context) error {
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
	a.lastError = ""
	a.mu.Unlock()

	log.Printf("bridge connected device_id=%s", a.cfg.DeviceID)

	if err := a.flushPending(conn); err != nil {
		log.Printf("initial flush failed: %v", err)
	}
	a.publishStatus()
	if err := a.RefreshRoster(); err != nil {
		log.Printf("roster refresh on connect: %v", err)
	}

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

func (a *App) handleWSMessage(conn *websocket.Conn, msg *services.BridgeMessage) error {
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
		a.publishStatus()
		return bridge.SendWriteAck(conn, &a.writeMu, msg.RequestID, ok, errMsg)
	case "scan_result":
		a.applyScanResultMessage(msg)
		return nil
	default:
		return nil
	}
}

func (a *App) applyScanResultMessage(msg *services.BridgeMessage) {
	if msg == nil {
		return
	}
	uid := strings.ToLower(strings.TrimSpace(msg.LogicalUUID))
	name, bib, raceID, raceName, result := "", "", "", "", ""
	if msg.Scan != nil {
		raw, err := json.Marshal(msg.Scan)
		if err == nil {
			var scan struct {
				Result          string `json:"result"`
				ParticipantName string `json:"participant_name"`
				BibNumber       string `json:"bib_number"`
				RaceName        string `json:"race_name"`
				RaceID          string `json:"race_id"`
			}
			if json.Unmarshal(raw, &scan) == nil {
				result = scan.Result
				name = scan.ParticipantName
				bib = scan.BibNumber
				raceName = scan.RaceName
				raceID = scan.RaceID
			}
		}
	}
	if (name == "" || bib == "") && uid != "" {
		if e, ok := a.roster.EntryForUUID(uid); ok {
			if name == "" {
				name = e.Name
			}
			if bib == "" {
				bib = e.Bib
			}
			if raceID == "" {
				raceID = e.RaceID
			}
			if raceName == "" {
				raceName = e.RaceName
			}
		}
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if uid != "" {
		a.lastTapUUID = uid
		a.lastRead = uid
		a.lastReadAt = time.Now().UTC()
	}
	a.lastTapAt = time.Now().UTC()
	a.lastTapResult = result
	a.lastTapName = name
	a.lastTapBib = bib
	a.lastTapRaceID = raceID
	a.lastTapRaceName = raceName
}

func (a *App) flushPending(conn *websocket.Conn) error {
	a.mu.RLock()
	writeOnly := a.cfg.WriteOnly
	a.mu.RUnlock()
	if writeOnly {
		// Hold queued automatic reads until write-only is turned off.
		// Still mark online_synced so operators/UI see a healthy bridge while programming.
		log.Printf("write-only: skipping pending flush (%d queued)", a.store.PendingCount())
		a.mu.Lock()
		a.syncing = false
		a.mode = bridge.ModeOnlineSynced
		now := time.Now().UTC()
		a.lastSyncAt = &now
		a.mu.Unlock()
		return nil
	}

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

func (a *App) publishStatus() {
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

func (a *App) handleDisconnect() {
	a.mu.Lock()
	if a.conn != nil {
		_ = a.conn.Close()
		a.conn = nil
	}
	a.online = false
	a.syncing = false
	if a.store != nil && a.store.PendingCount() > 0 {
		a.mode = bridge.ModeOffline
	}
	a.mu.Unlock()
	a.publishStatus()
}

// RunHeadless blocks until ctx is cancelled (CLI device-bridge entrypoint).
func RunHeadless(ctx context.Context, cfg Config) error {
	app, err := New(cfg)
	if err != nil {
		return err
	}
	if err := app.Start(ctx); err != nil {
		return err
	}
	<-ctx.Done()
	app.Stop()
	return nil
}
