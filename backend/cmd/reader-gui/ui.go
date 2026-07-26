package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/keweenaw-endurance/backend/internal/bridgeapp"
)

type readerUI struct {
	app     fyne.App
	win     fyne.Window
	cfg     bridgeapp.Config
	cfgPath string
	bridge  *bridgeapp.App
	cancel  context.CancelFunc
	races   []bridgeapp.BluffetRace

	hostedURL    *widget.Entry
	bridgeToken  *widget.Entry
	organizerPIN *widget.Entry
	deviceID     *widget.Entry
	eventID      *widget.Entry
	raceSelect   *widget.Select
	raceID       *widget.Entry
	checkpointID *widget.Entry
	dataDir      *widget.Entry
	proxmarkCLI  *widget.Entry
	proxmarkPort *widget.SelectEntry
	hwCheck      *widget.Check
	mockCheck    *widget.Check
	autofillMsg  *widget.Label

	statusMode   *widget.Label
	statusDetail *widget.Label
	statusError  *widget.Label
	startBtn     *widget.Button
	stopBtn      *widget.Button

	bibEntry  *widget.Entry
	manualMsg *widget.Label
}

func newReaderWindow(a fyne.App, cfg bridgeapp.Config, cfgPath string) fyne.Window {
	w := a.NewWindow("Keweenaw Endurance — Reader")
	w.Resize(fyne.NewSize(740, 820))
	ui := &readerUI{
		app:     a,
		win:     w,
		cfg:     cfg,
		cfgPath: cfgPath,
		races:   bridgeapp.SeedBluffetDetails().Races,
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
	ui.eventID = widget.NewEntry()
	ui.eventID.SetText(ui.cfg.EventID)
	ui.raceID = widget.NewEntry()
	ui.raceID.SetText(ui.cfg.RaceID)
	ui.checkpointID = widget.NewEntry()
	ui.checkpointID.SetText(ui.cfg.CheckpointID)
	ui.dataDir = widget.NewEntry()
	ui.dataDir.SetText(ui.cfg.DataDir)
	ui.proxmarkCLI = widget.NewEntry()
	ui.proxmarkCLI.SetText(ui.cfg.ProxmarkCLI)

	ui.raceSelect = widget.NewSelect(ui.raceOptionLabels(), func(name string) {
		ui.applyRaceSelection(name)
	})
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
	ui.mockCheck = widget.NewCheck("Mock reader (no hardware)", nil)
	ui.mockCheck.SetChecked(ui.cfg.BridgeMock)
	ui.autofillMsg = widget.NewLabel("Autofill: loading Bluffet + bridge token…")
	ui.autofillMsg.Wrapping = fyne.TextWrapWord

	ui.statusMode = widget.NewLabel("Stopped")
	ui.statusMode.TextStyle = fyne.TextStyle{Bold: true}
	ui.statusDetail = widget.NewLabel("Start the bridge when ready.")
	ui.statusDetail.Wrapping = fyne.TextWrapWord
	ui.statusError = widget.NewLabel("")
	ui.statusError.Wrapping = fyne.TextWrapWord

	ui.startBtn = widget.NewButtonWithIcon("Start bridge", theme.MediaPlayIcon(), ui.onStart)
	ui.startBtn.Importance = widget.HighImportance
	ui.stopBtn = widget.NewButtonWithIcon("Stop", theme.MediaStopIcon(), ui.onStop)
	ui.stopBtn.Disable()

	ui.bibEntry = widget.NewEntry()
	ui.bibEntry.SetPlaceHolder("Bib number")
	ui.manualMsg = widget.NewLabel("")
	ui.manualMsg.Wrapping = fyne.TextWrapWord
}

func (ui *readerUI) raceOptionLabels() []string {
	labels := make([]string, 0, len(ui.races))
	for _, r := range ui.races {
		labels = append(labels, r.Name)
	}
	return labels
}

func (ui *readerUI) selectRaceMatchingConfig() {
	if len(ui.races) == 0 {
		return
	}
	for _, r := range ui.races {
		if r.RaceID == ui.cfg.RaceID || strings.HasSuffix(r.RaceID, ui.cfg.RaceID) || strings.HasSuffix(ui.cfg.RaceID, r.RaceID) {
			ui.raceSelect.SetSelected(r.Name)
			return
		}
	}
	ui.raceSelect.SetSelected(ui.races[0].Name)
	ui.applyRaceSelection(ui.races[0].Name)
}

func (ui *readerUI) applyRaceSelection(name string) {
	for _, r := range ui.races {
		if r.Name != name {
			continue
		}
		ui.raceID.SetText(r.RaceID)
		ui.checkpointID.SetText(r.FinishCheckpointID)
		ui.cfg.RaceID = r.RaceID
		ui.cfg.CheckpointID = r.FinishCheckpointID
		return
	}
}

func (ui *readerUI) layout() fyne.CanvasObject {
	header := widget.NewRichTextFromMarkdown(
		"## Keweenaw Endurance — Reader\nAll You Can East Bluffet · Proxmark bridge + manual lap entry",
	)

	configForm := widget.NewForm(
		widget.NewFormItem("Hosted API URL", ui.hostedURL),
		widget.NewFormItem("Bridge token", ui.bridgeToken),
		widget.NewFormItem("Organizer PIN", ui.organizerPIN),
		widget.NewFormItem("Device ID", ui.deviceID),
		widget.NewFormItem("Event ID", ui.eventID),
		widget.NewFormItem("Bluffet race", ui.raceSelect),
		widget.NewFormItem("Race ID", ui.raceID),
		widget.NewFormItem("Finish checkpoint ID", ui.checkpointID),
		widget.NewFormItem("Data directory", ui.dataDir),
		widget.NewFormItem("Proxmark CLI", ui.proxmarkCLI),
		widget.NewFormItem("COM port", ui.proxmarkPort),
	)

	saveBtn := widget.NewButton("Save config", ui.onSave)
	testBtn := widget.NewButton("Test Proxmark", ui.onTestProxmark)
	reloadBtn := widget.NewButton("Reload Bluffet autofill", func() { go ui.runAutofill() })
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
		ui.statusError,
	)

	manualSubmit := widget.NewButtonWithIcon("Record lap", theme.ConfirmIcon(), ui.onManualEntry)
	manualSubmit.Importance = widget.WarningImportance
	manualBox := container.NewVBox(
		widget.NewSeparator(),
		widget.NewLabelWithStyle("Manual entry (Proxmark fallback)", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewLabel("Enter bib and record a lap when the reader misses a tap. Online uses PIN auth; offline uses the roster cache (connect once first)."),
		ui.bibEntry,
		manualSubmit,
		ui.manualMsg,
	)

	return container.NewVScroll(container.NewPadded(container.NewVBox(
		header,
		ui.autofillMsg,
		configForm,
		ui.hwCheck,
		ui.mockCheck,
		controls,
		statusBox,
		manualBox,
	)))
}

func (ui *readerUI) runAutofill() {
	fyne.Do(func() {
		ui.autofillMsg.SetText("Autofill: fetching bridge token + Bluffet races…")
	})

	cfg := ui.readFormSafe()
	details, fetchErr := bridgeapp.AutofillConfig(&cfg, true)

	fyne.Do(func() {
		ui.cfg = cfg
		ui.races = details.Races
		ui.hostedURL.SetText(cfg.HostedAPIURL)
		ui.bridgeToken.SetText(cfg.BridgeToken)
		ui.organizerPIN.SetText(cfg.OrganizerPIN)
		ui.deviceID.SetText(cfg.DeviceID)
		ui.eventID.SetText(cfg.EventID)
		ui.raceID.SetText(cfg.RaceID)
		ui.checkpointID.SetText(cfg.CheckpointID)
		ui.dataDir.SetText(cfg.DataDir)
		ui.proxmarkCLI.SetText(cfg.ProxmarkCLI)
		ui.proxmarkPort.SetText(cfg.ProxmarkPort)
		ui.hwCheck.SetChecked(cfg.RFIDHardware)
		ui.mockCheck.SetChecked(cfg.BridgeMock)
		ui.raceSelect.Options = ui.raceOptionLabels()
		ui.selectRaceMatchingConfig()

		tokenOK := cfg.BridgeToken != ""
		msg := fmt.Sprintf("Autofill ready · event %s · %d races", shortID(cfg.EventID), len(details.Races))
		if tokenOK {
			msg += " · bridge token loaded"
		} else {
			msg += " · bridge token missing (set BRIDGE_TOKEN or gcloud auth)"
		}
		if fetchErr != nil && !tokenOK {
			msg += " · " + fetchErr.Error()
		} else if fetchErr != nil {
			msg += " · hosted races: seed fallback (" + fetchErr.Error() + ")"
		}
		ui.autofillMsg.SetText(msg)
		_ = bridgeapp.SaveConfig(ui.cfgPath, cfg)
	})
}

func (ui *readerUI) readFormSafe() bridgeapp.Config {
	// Called from background goroutine before widgets exist is OK via cfg copy;
	// after build, prefer widget values on UI thread only. Here we start from cfg.
	cfg := ui.cfg
	if ui.hostedURL != nil {
		// May race; AutofillConfig only fills empties when force=false.
		// We pass force=true and rebuild from current cfg snapshot taken on UI thread first.
	}
	return cfg
}

func shortID(id string) string {
	id = strings.TrimSpace(id)
	if len(id) <= 8 {
		return id
	}
	return id[len(id)-6:]
}

func (ui *readerUI) readForm() bridgeapp.Config {
	cfg := ui.cfg
	cfg.HostedAPIURL = strings.TrimSpace(ui.hostedURL.Text)
	cfg.BridgeToken = strings.TrimSpace(ui.bridgeToken.Text)
	cfg.OrganizerPIN = strings.TrimSpace(ui.organizerPIN.Text)
	cfg.DeviceID = strings.TrimSpace(ui.deviceID.Text)
	cfg.EventID = strings.TrimSpace(ui.eventID.Text)
	cfg.RaceID = strings.TrimSpace(ui.raceID.Text)
	cfg.CheckpointID = strings.TrimSpace(ui.checkpointID.Text)
	cfg.DataDir = strings.TrimSpace(ui.dataDir.Text)
	cfg.ProxmarkCLI = strings.TrimSpace(ui.proxmarkCLI.Text)
	cfg.ProxmarkPort = strings.TrimSpace(ui.proxmarkPort.Text)
	cfg.RFIDHardware = ui.hwCheck.Checked
	cfg.BridgeMock = ui.mockCheck.Checked
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
	err := ui.bridge.ManualEntry(bib, time.Now().UTC())
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
		detail := fmt.Sprintf(
			"mode=%s  connected=%v  pending=%d  last_read=%s",
			mode, st.Connected, st.PendingCount, st.LastRead,
		)
		if st.LastSyncAt != nil {
			detail += "  last_sync=" + st.LastSyncAt.Local().Format(time.Kitchen)
		}
		errText := st.LastError
		fyne.Do(func() {
			ui.statusMode.SetText(strings.ToUpper(mode))
			ui.statusDetail.SetText(detail)
			ui.statusError.SetText(errText)
		})
	}
}
