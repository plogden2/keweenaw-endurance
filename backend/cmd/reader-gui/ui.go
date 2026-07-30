package main

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/keweenaw-endurance/backend/internal/bridge"
	"github.com/keweenaw-endurance/backend/internal/bridgeapp"
	"github.com/keweenaw-endurance/backend/internal/rfid"
)

const allRacesLabel = "All races (event finish)"

type readerUI struct {
	app     fyne.App
	win     fyne.Window
	cfg     bridgeapp.Config
	cfgPath string
	bridge  *bridgeapp.App
	cancel  context.CancelFunc

	mu          sync.Mutex
	events      []bridgeapp.CatalogEvent
	races       []bridgeapp.CatalogRace
	checkpoints []bridgeapp.CatalogCheckpoint

	hostedURL        *widget.Entry
	bridgeToken      *widget.Entry
	organizerPIN     *widget.Entry
	deviceID         *widget.Entry
	eventSelect      *widget.Select
	raceSelect       *widget.Select
	checkpointSelect *widget.Select
	dataDir          *widget.Entry
	proxmarkCLI      *widget.Entry
	proxmarkPort     *widget.SelectEntry
	hwCheck          *widget.Check
	hfGainSlider     *widget.Slider
	hfGainLabel      *widget.Label
	mockCheck        *widget.Check
	writeOnlyCheck   *widget.Check
	autofillMsg      *widget.Label

	statusMode   *widget.Label
	statusDetail *widget.Label
	statusTap    *widget.Label
	statusError  *widget.Label
	startBtn     *widget.Button
	stopBtn      *widget.Button

	bibEntry         *widget.Entry
	manualRaceSelect *widget.Select
	manualMsg        *widget.Label
}

func newReaderWindow(a fyne.App, cfg bridgeapp.Config, cfgPath string) fyne.Window {
	w := a.NewWindow("Keweenaw Endurance — Reader")
	w.Resize(fyne.NewSize(760, 860))
	seedEv, seedRaces := bridgeapp.SeedCatalogFromBluffet()
	ui := &readerUI{
		app:     a,
		win:     w,
		cfg:     cfg,
		cfgPath: cfgPath,
		events:  []bridgeapp.CatalogEvent{seedEv},
		races:   seedRaces,
	}
	ui.build()
	w.SetContent(ui.layout())
	w.SetCloseIntercept(func() {
		ui.stopBridge()
		w.Close()
	})
	go ui.statusLoop()
	go ui.runAutofill()
	return w
}

func (ui *readerUI) build() {
	ui.hostedURL = widget.NewEntry()
	ui.hostedURL.SetText(ui.cfg.HostedAPIURL)
	ui.bridgeToken = widget.NewPasswordEntry()
	ui.bridgeToken.SetText(ui.cfg.BridgeToken)
	ui.organizerPIN = widget.NewPasswordEntry()
	ui.organizerPIN.SetText(ui.cfg.OrganizerPIN)
	ui.deviceID = widget.NewEntry()
	ui.deviceID.SetText(ui.cfg.DeviceID)
	ui.dataDir = widget.NewEntry()
	ui.dataDir.SetText(ui.cfg.DataDir)
	ui.proxmarkCLI = widget.NewEntry()
	ui.proxmarkCLI.SetText(ui.cfg.ProxmarkCLI)

	ui.eventSelect = widget.NewSelect(ui.eventOptionLabels(), func(name string) {
		ui.onEventSelected(name)
	})
	ui.raceSelect = widget.NewSelect(ui.raceOptionLabels(), func(name string) {
		ui.onRaceSelected(name)
	})
	ui.checkpointSelect = widget.NewSelect([]string{}, func(name string) {
		ui.onCheckpointSelected(name)
	})
	ui.selectEventMatchingConfig()
	ui.selectRaceMatchingConfig()

	ports, _ := bridgeapp.ListSerialPorts()
	if len(ports) == 0 {
		ports = []string{"COM3", "COM4", "COM5"}
	}
	ui.proxmarkPort = widget.NewSelectEntry(ports)
	if ui.cfg.ProxmarkPort != "" {
		ui.proxmarkPort.SetText(ui.cfg.ProxmarkPort)
	} else if len(ports) > 0 {
		ui.proxmarkPort.SetText(ports[0])
	}

	ui.hwCheck = widget.NewCheck("Use Proxmark hardware", nil)
	ui.hwCheck.SetChecked(ui.cfg.RFIDHardware)

	hfGain := ui.cfg.HFGain
	if hfGain == 0 {
		hfGain = rfid.HFGainDefault
	}
	ui.hfGainSlider = widget.NewSlider(rfid.HFGainMin, rfid.HFGainMax)
	ui.hfGainSlider.Step = 1
	ui.hfGainSlider.SetValue(float64(hfGain))
	ui.hfGainLabel = widget.NewLabel(hfGainLabelText(hfGain))
	ui.hfGainSlider.OnChanged = func(value float64) {
		gain := int(value)
		ui.cfg.HFGain = gain
		ui.hfGainLabel.SetText(hfGainLabelText(gain))
		if ui.bridge != nil && ui.bridge.Running() {
			ui.bridge.SetHFGain(gain)
		}
		_ = bridgeapp.SaveConfig(ui.cfgPath, ui.readForm())
	}

	ui.mockCheck = widget.NewCheck("Mock reader (no hardware)", nil)
	ui.mockCheck.SetChecked(ui.cfg.BridgeMock)
	ui.writeOnlyCheck = widget.NewCheck("Write-only mode (show taps, do not record)", func(on bool) {
		ui.cfg.WriteOnly = on
		if ui.bridge != nil && ui.bridge.Running() {
			ui.bridge.SetWriteOnly(on)
		}
		_ = bridgeapp.SaveConfig(ui.cfgPath, ui.readForm())
	})
	ui.writeOnlyCheck.SetChecked(ui.cfg.WriteOnly)
	ui.autofillMsg = widget.NewLabel("Loading events & races…")
	ui.autofillMsg.Wrapping = fyne.TextWrapWord

	ui.statusMode = widget.NewLabel("Stopped")
	ui.statusMode.TextStyle = fyne.TextStyle{Bold: true}
	ui.statusDetail = widget.NewLabel("Start the bridge when ready.")
	ui.statusDetail.Wrapping = fyne.TextWrapWord
	ui.statusTap = widget.NewLabel("Last tap: —")
	ui.statusTap.Wrapping = fyne.TextWrapWord
	ui.statusError = widget.NewLabel("")
	ui.statusError.Wrapping = fyne.TextWrapWord

	ui.startBtn = widget.NewButtonWithIcon("Start bridge", theme.MediaPlayIcon(), ui.onStart)
	ui.startBtn.Importance = widget.HighImportance
	ui.stopBtn = widget.NewButtonWithIcon("Stop", theme.MediaStopIcon(), ui.onStop)
	ui.stopBtn.Disable()

	ui.bibEntry = widget.NewEntry()
	ui.bibEntry.SetPlaceHolder("Bib number")
	ui.manualRaceSelect = widget.NewSelect(ui.manualRaceOptionLabels(), nil)
	ui.manualRaceSelect.SetSelected(allRacesLabel)
	ui.manualMsg = widget.NewLabel("")
	ui.manualMsg.Wrapping = fyne.TextWrapWord
}

func (ui *readerUI) eventOptionLabels() []string {
	ui.mu.Lock()
	defer ui.mu.Unlock()
	labels := make([]string, 0, len(ui.events))
	for _, e := range ui.events {
		labels = append(labels, e.Name)
	}
	return labels
}

func (ui *readerUI) raceOptionLabels() []string {
	ui.mu.Lock()
	defer ui.mu.Unlock()
	labels := []string{allRacesLabel}
	for _, r := range ui.races {
		labels = append(labels, r.Name)
	}
	return labels
}

func (ui *readerUI) manualRaceOptionLabels() []string {
	return ui.raceOptionLabels()
}

func (ui *readerUI) checkpointOptionLabels() []string {
	ui.mu.Lock()
	defer ui.mu.Unlock()
	labels := make([]string, 0, len(ui.checkpoints))
	for _, cp := range ui.checkpoints {
		label := cp.Name
		if cp.Type != "" && !strings.EqualFold(cp.Name, cp.Type) {
			label = fmt.Sprintf("%s (%s)", cp.Name, cp.Type)
		}
		labels = append(labels, label)
	}
	return labels
}

func (ui *readerUI) selectEventMatchingConfig() {
	ui.mu.Lock()
	events := append([]bridgeapp.CatalogEvent(nil), ui.events...)
	eventID := ui.cfg.EventID
	ui.mu.Unlock()
	for _, e := range events {
		if e.ID == eventID || strings.HasSuffix(e.ID, eventID) || strings.HasSuffix(eventID, e.ID) {
			ui.eventSelect.SetSelected(e.Name)
			return
		}
	}
	if len(events) > 0 {
		ui.eventSelect.SetSelected(events[0].Name)
	}
}

func (ui *readerUI) selectRaceMatchingConfig() {
	if strings.TrimSpace(ui.cfg.RaceID) == "" {
		ui.raceSelect.SetSelected(allRacesLabel)
		ui.checkpointSelect.Disable()
		ui.checkpointSelect.SetOptions([]string{})
		ui.checkpointSelect.ClearSelected()
		return
	}
	ui.mu.Lock()
	races := append([]bridgeapp.CatalogRace(nil), ui.races...)
	raceID := ui.cfg.RaceID
	ui.mu.Unlock()
	for _, r := range races {
		if r.ID == raceID || strings.HasSuffix(r.ID, raceID) || strings.HasSuffix(raceID, r.ID) {
			ui.raceSelect.SetSelected(r.Name)
			ui.checkpointSelect.Enable()
			return
		}
	}
	ui.raceSelect.SetSelected(allRacesLabel)
}

func (ui *readerUI) onEventSelected(name string) {
	ui.mu.Lock()
	var eventID string
	for _, e := range ui.events {
		if e.Name == name {
			eventID = e.ID
			break
		}
	}
	ui.cfg.EventID = eventID
	ui.mu.Unlock()
	go ui.reloadRacesForEvent(eventID)
}

func (ui *readerUI) onRaceSelected(name string) {
	if name == allRacesLabel || name == "" {
		ui.cfg.RaceID = ""
		ui.cfg.CheckpointID = ""
		ui.checkpointSelect.SetOptions([]string{})
		ui.checkpointSelect.ClearSelected()
		ui.checkpointSelect.Disable()
		return
	}
	ui.mu.Lock()
	var race bridgeapp.CatalogRace
	for _, r := range ui.races {
		if r.Name == name {
			race = r
			break
		}
	}
	ui.cfg.RaceID = race.ID
	if race.FinishCheckpointID != "" {
		ui.cfg.CheckpointID = race.FinishCheckpointID
	}
	ui.mu.Unlock()
	ui.checkpointSelect.Enable()
	go ui.reloadCheckpoints(race.ID, race.FinishCheckpointID)
}

func (ui *readerUI) onCheckpointSelected(name string) {
	ui.mu.Lock()
	defer ui.mu.Unlock()
	for _, cp := range ui.checkpoints {
		label := cp.Name
		if cp.Type != "" && !strings.EqualFold(cp.Name, cp.Type) {
			label = fmt.Sprintf("%s (%s)", cp.Name, cp.Type)
		}
		if label == name {
			ui.cfg.CheckpointID = cp.ID
			return
		}
	}
}

func (ui *readerUI) layout() fyne.CanvasObject {
	header := widget.NewRichTextFromMarkdown(
		"## Keweenaw Endurance — Reader\nProxmark bridge · event finish scores all races · manual lap fallback",
	)

	configForm := widget.NewForm(
		widget.NewFormItem("Hosted API URL", ui.hostedURL),
		widget.NewFormItem("Bridge token", ui.bridgeToken),
		widget.NewFormItem("Organizer PIN", ui.organizerPIN),
		widget.NewFormItem("Device ID", ui.deviceID),
		widget.NewFormItem("Event", ui.eventSelect),
		widget.NewFormItem("Race", ui.raceSelect),
		widget.NewFormItem("Checkpoint (manual)", ui.checkpointSelect),
		widget.NewFormItem("Data directory", ui.dataDir),
		widget.NewFormItem("Proxmark CLI", ui.proxmarkCLI),
		widget.NewFormItem("COM port", ui.proxmarkPort),
	)

	saveBtn := widget.NewButton("Save config", ui.onSave)
	testBtn := widget.NewButton("Test Proxmark", ui.onTestProxmark)
	reloadBtn := widget.NewButton("Reload catalog", func() { go ui.runAutofill() })
	refreshPorts := widget.NewButton("Refresh COM ports", func() {
		ports, err := bridgeapp.ListSerialPorts()
		if err != nil {
			dialog.ShowError(err, ui.win)
			return
		}
		if len(ports) == 0 {
			ports = []string{"COM3"}
		}
		ui.proxmarkPort.SetOptions(ports)
		dialog.ShowInformation("COM ports", strings.Join(ports, ", "), ui.win)
	})

	controls := container.NewHBox(ui.startBtn, ui.stopBtn, saveBtn, testBtn, reloadBtn, refreshPorts)

	statusBox := container.NewVBox(
		widget.NewSeparator(),
		widget.NewLabelWithStyle("Status", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		ui.statusMode,
		ui.statusDetail,
		ui.statusTap,
		ui.statusError,
	)

	manualSubmit := widget.NewButtonWithIcon("Record lap", theme.ConfirmIcon(), ui.onManualEntry)
	manualSubmit.Importance = widget.WarningImportance
	manualBox := container.NewVBox(
		widget.NewSeparator(),
		widget.NewLabelWithStyle("Manual entry (Proxmark fallback)", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewLabel("Bib auto-resolves across all races; pick a race only if the bib is ambiguous."),
		ui.bibEntry,
		widget.NewForm(widget.NewFormItem("Race override", ui.manualRaceSelect)),
		manualSubmit,
		ui.manualMsg,
	)

	return container.NewVScroll(container.NewPadded(container.NewVBox(
		header,
		ui.autofillMsg,
		configForm,
		ui.hwCheck,
		container.NewBorder(nil, nil, ui.hfGainLabel, nil, ui.hfGainSlider),
		ui.mockCheck,
		ui.writeOnlyCheck,
		controls,
		statusBox,
		manualBox,
	)))
}

func (ui *readerUI) runAutofill() {
	fyne.Do(func() {
		ui.autofillMsg.SetText("Loading catalog (events / races)…")
	})

	var cfg bridgeapp.Config
	fyne.DoAndWait(func() {
		cfg = ui.readForm()
	})

	details, fetchErr := bridgeapp.AutofillConfig(&cfg, true)
	auth, _ := bridge.ResolveHostedAuth(cfg.HostedAPIURL, cfg.BridgeToken, cfg.OrganizerPIN, nil)
	_ = bridge.EnsureBearer(auth, nil, cfg.OrganizerPIN)

	events, evErr := bridgeapp.FetchCatalogEvents(auth, nil)
	if evErr != nil || len(events) == 0 {
		seedEv, _ := bridgeapp.SeedCatalogFromBluffet()
		events = []bridgeapp.CatalogEvent{seedEv}
	}
	races, raceErr := bridgeapp.FetchCatalogRaces(auth, nil, cfg.EventID)
	if raceErr != nil || len(races) == 0 {
		_, seedRaces := bridgeapp.SeedCatalogFromBluffet()
		races = seedRaces
		if len(details.Races) > 0 {
			races = nil
			for _, r := range details.Races {
				races = append(races, bridgeapp.CatalogRace{
					ID: r.RaceID, Name: r.Name, FinishCheckpointID: r.FinishCheckpointID,
				})
			}
		}
	}

	fyne.Do(func() {
		ui.cfg = cfg
		ui.mu.Lock()
		ui.events = events
		ui.races = races
		ui.mu.Unlock()

		ui.hostedURL.SetText(cfg.HostedAPIURL)
		ui.bridgeToken.SetText(cfg.BridgeToken)
		ui.organizerPIN.SetText(cfg.OrganizerPIN)
		ui.deviceID.SetText(cfg.DeviceID)
		ui.dataDir.SetText(cfg.DataDir)
		ui.proxmarkCLI.SetText(cfg.ProxmarkCLI)
		ui.proxmarkPort.SetText(cfg.ProxmarkPort)
		ui.hwCheck.SetChecked(cfg.RFIDHardware)
		hfGain := cfg.HFGain
		if hfGain == 0 {
			hfGain = rfid.HFGainDefault
		}
		ui.hfGainSlider.SetValue(float64(hfGain))
		ui.hfGainLabel.SetText(hfGainLabelText(hfGain))
		ui.mockCheck.SetChecked(cfg.BridgeMock)
		ui.writeOnlyCheck.SetChecked(cfg.WriteOnly)

		ui.eventSelect.Options = ui.eventOptionLabels()
		ui.raceSelect.Options = ui.raceOptionLabels()
		ui.manualRaceSelect.Options = ui.manualRaceOptionLabels()
		ui.selectEventMatchingConfig()
		ui.selectRaceMatchingConfig()
		if ui.cfg.RaceID != "" {
			go ui.reloadCheckpoints(ui.cfg.RaceID, ui.cfg.CheckpointID)
		}

		msg := fmt.Sprintf("Catalog ready · %d events · %d races", len(events), len(races))
		if cfg.BridgeToken != "" {
			msg += " · bridge token loaded"
		}
		if fetchErr != nil {
			msg += " · " + fetchErr.Error()
		}
		if evErr != nil {
			msg += " · events: seed/fallback"
		}
		ui.autofillMsg.SetText(msg)
		_ = bridgeapp.SaveConfig(ui.cfgPath, cfg)
	})
}

func (ui *readerUI) reloadRacesForEvent(eventID string) {
	cfg := ui.cfg
	auth, err := bridge.ResolveHostedAuth(cfg.HostedAPIURL, cfg.BridgeToken, cfg.OrganizerPIN, nil)
	if err != nil {
		return
	}
	_ = bridge.EnsureBearer(auth, &http.Client{Timeout: 15 * time.Second}, cfg.OrganizerPIN)
	races, err := bridgeapp.FetchCatalogRaces(auth, nil, eventID)
	if err != nil || len(races) == 0 {
		return
	}
	fyne.Do(func() {
		ui.mu.Lock()
		ui.races = races
		ui.mu.Unlock()
		ui.raceSelect.Options = ui.raceOptionLabels()
		ui.manualRaceSelect.Options = ui.manualRaceOptionLabels()
		// Keep All races selected when switching events unless a race still matches.
		ui.selectRaceMatchingConfig()
	})
}

func (ui *readerUI) reloadCheckpoints(raceID, preferID string) {
	cfg := ui.cfg
	auth, err := bridge.ResolveHostedAuth(cfg.HostedAPIURL, cfg.BridgeToken, cfg.OrganizerPIN, nil)
	if err != nil {
		return
	}
	_ = bridge.EnsureBearer(auth, nil, cfg.OrganizerPIN)
	cps, err := bridgeapp.FetchCatalogCheckpoints(auth, nil, raceID)
	if err != nil {
		return
	}
	fyne.Do(func() {
		ui.mu.Lock()
		ui.checkpoints = cps
		ui.mu.Unlock()
		labels := ui.checkpointOptionLabels()
		ui.checkpointSelect.SetOptions(labels)
		selected := ""
		for _, cp := range cps {
			label := cp.Name
			if cp.Type != "" && !strings.EqualFold(cp.Name, cp.Type) {
				label = fmt.Sprintf("%s (%s)", cp.Name, cp.Type)
			}
			if preferID != "" && (cp.ID == preferID || strings.EqualFold(cp.Type, "finish")) {
				selected = label
				if cp.ID == preferID {
					break
				}
			}
		}
		if selected == "" && len(labels) > 0 {
			selected = labels[0]
		}
		if selected != "" {
			ui.checkpointSelect.SetSelected(selected)
			ui.onCheckpointSelected(selected)
		}
	})
}

func (ui *readerUI) readForm() bridgeapp.Config {
	cfg := ui.cfg
	cfg.HostedAPIURL = strings.TrimSpace(ui.hostedURL.Text)
	cfg.BridgeToken = strings.TrimSpace(ui.bridgeToken.Text)
	cfg.OrganizerPIN = strings.TrimSpace(ui.organizerPIN.Text)
	cfg.DeviceID = strings.TrimSpace(ui.deviceID.Text)
	cfg.DataDir = strings.TrimSpace(ui.dataDir.Text)
	cfg.ProxmarkCLI = strings.TrimSpace(ui.proxmarkCLI.Text)
	cfg.ProxmarkPort = strings.TrimSpace(ui.proxmarkPort.Text)
	cfg.RFIDHardware = ui.hwCheck.Checked
	cfg.HFGain = int(ui.hfGainSlider.Value)
	cfg.BridgeMock = ui.mockCheck.Checked
	cfg.WriteOnly = ui.writeOnlyCheck.Checked
	// Event/race/checkpoint kept in ui.cfg via select handlers.
	cfg.EventID = ui.cfg.EventID
	cfg.RaceID = ui.cfg.RaceID
	cfg.CheckpointID = ui.cfg.CheckpointID
	if cfg.PollMS <= 0 {
		cfg.PollMS = 500
	}
	return cfg
}

func (ui *readerUI) onSave() {
	cfg := ui.readForm()
	ui.cfg = cfg
	path := ui.cfgPath
	if cfg.DataDir != "" {
		path = bridgeapp.ConfigPath(cfg.DataDir)
		ui.cfgPath = path
	}
	if err := bridgeapp.SaveConfig(path, cfg); err != nil {
		dialog.ShowError(err, ui.win)
		return
	}
	dialog.ShowInformation("Saved", "Config written to:\n"+path, ui.win)
}

func (ui *readerUI) onTestProxmark() {
	cfg := ui.readForm()
	msg, err := bridgeapp.TestProxmark(cfg)
	if err != nil {
		dialog.ShowError(err, ui.win)
		return
	}
	dialog.ShowInformation("Proxmark", msg, ui.win)
}

func (ui *readerUI) onStart() {
	if ui.bridge != nil && ui.bridge.Running() {
		return
	}
	cfg := ui.readForm()
	ui.cfg = cfg
	_ = bridgeapp.SaveConfig(ui.cfgPath, cfg)

	appInst, err := bridgeapp.New(cfg)
	if err != nil {
		dialog.ShowError(err, ui.win)
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	ui.cancel = cancel
	if err := appInst.Start(ctx); err != nil {
		cancel()
		dialog.ShowError(err, ui.win)
		return
	}
	ui.bridge = appInst
	ui.startBtn.Disable()
	ui.stopBtn.Enable()
	ui.statusMode.SetText("Starting…")
}

func (ui *readerUI) onStop() {
	ui.stopBridge()
	ui.startBtn.Enable()
	ui.stopBtn.Disable()
	ui.statusMode.SetText("Stopped")
	ui.statusDetail.SetText("Bridge stopped.")
}

func (ui *readerUI) stopBridge() {
	if ui.cancel != nil {
		ui.cancel()
		ui.cancel = nil
	}
	if ui.bridge != nil {
		ui.bridge.Stop()
		ui.bridge = nil
	}
}

func (ui *readerUI) onManualEntry() {
	bib := strings.TrimSpace(ui.bibEntry.Text)
	if bib == "" {
		ui.manualMsg.SetText("Enter a bib number.")
		return
	}
	if ui.bridge == nil || !ui.bridge.Running() {
		ui.manualMsg.SetText("Start the bridge first (needed for auth + offline queue).")
		return
	}
	raceOverride := ""
	sel := ui.manualRaceSelect.Selected
	if sel != "" && sel != allRacesLabel {
		ui.mu.Lock()
		for _, r := range ui.races {
			if r.Name == sel {
				raceOverride = r.ID
				break
			}
		}
		ui.mu.Unlock()
	}
	err := ui.bridge.ManualEntryInRace(bib, time.Now().UTC(), raceOverride)
	if err != nil {
		ui.manualMsg.SetText("Error: " + err.Error())
		return
	}
	ui.manualMsg.SetText(fmt.Sprintf("Recorded lap for bib %s at %s", bib, time.Now().Format(time.Kitchen)))
	ui.bibEntry.SetText("")
}

func (ui *readerUI) statusLoop() {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	for range ticker.C {
		b := ui.bridge
		if b == nil {
			continue
		}
		st := b.StatusSnapshot()
		mode := st.Mode
		if mode == "" {
			mode = "unknown"
		}
		modeLabel := strings.ToUpper(mode)
		if st.WriteOnly {
			modeLabel = "WRITE ONLY"
		}
		detail := fmt.Sprintf(
			"mode=%s  connected=%v  pending=%d",
			mode, st.Connected, st.PendingCount,
		)
		if st.WriteOnly {
			detail += "  (taps shown, not recorded)"
		}
		if st.LastSyncAt != nil {
			detail += "  last_sync=" + st.LastSyncAt.Local().Format(time.Kitchen)
		}
		tap := formatLastTap(st)
		errText := st.LastError
		fyne.Do(func() {
			ui.statusMode.SetText(modeLabel)
			ui.statusDetail.SetText(detail)
			ui.statusTap.SetText(tap)
			ui.statusError.SetText(errText)
		})
	}
}

func hfGainLabelText(gain int) string {
	if gain == rfid.HFGainMax {
		return fmt.Sprintf("HF gain: %d (max sensitivity)", gain)
	}
	return fmt.Sprintf("HF gain: %d", gain)
}

func formatLastTap(st bridgeapp.Status) string {
	if st.LastTapUUID == "" && st.LastRead == "" {
		return "Last tap: —"
	}
	uid := st.LastTapUUID
	if uid == "" {
		uid = st.LastRead
	}
	parts := []string{}
	if st.LastTapName != "" {
		parts = append(parts, st.LastTapName)
	}
	if st.LastTapBib != "" {
		parts = append(parts, "bib "+st.LastTapBib)
	}
	if st.LastTapRaceName != "" {
		parts = append(parts, st.LastTapRaceName)
	}
	parts = append(parts, uid)
	line := "Last tap: " + strings.Join(parts, " · ")
	if st.LastTapResult != "" {
		line += " (" + st.LastTapResult + ")"
	}
	return line
}
