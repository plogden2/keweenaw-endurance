package main

import (
	"log"
	"os"
	"path/filepath"

	"fyne.io/fyne/v2/app"
	"github.com/keweenaw-endurance/backend/internal/bridgeapp"
)

func main() {
	defaultData := bridgeapp.PreferredGUIDataDir()
	cfgPath := bridgeapp.ConfigPath(defaultData)
	cfg, err := bridgeapp.LoadConfig(cfgPath)
	if err != nil {
		log.Printf("config load: %v — using defaults", err)
		cfg = bridgeapp.DefaultConfig()
		cfg.DataDir = defaultData
	}
	if cfg.DataDir == "" || cfg.DataDir == "./bridge-data" {
		if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
			cfg.DataDir = defaultData
		}
	}
	if exe, err := os.Executable(); err == nil {
		scriptsPM3 := filepath.Clean(filepath.Join(filepath.Dir(exe), "..", "scripts", "pm3.cmd"))
		if cfg.ProxmarkCLI == "" || cfg.ProxmarkCLI == "pm3" {
			if _, err := os.Stat(scriptsPM3); err == nil {
				cfg.ProxmarkCLI = scriptsPM3
			}
		}
		alt := filepath.Clean(filepath.Join(filepath.Dir(exe), "..", "..", "scripts", "pm3.cmd"))
		if cfg.ProxmarkCLI == "" || cfg.ProxmarkCLI == "pm3" {
			if _, err := os.Stat(alt); err == nil {
				cfg.ProxmarkCLI = alt
			}
		}
	}

	// Seed Bluffet fields immediately so the form is never empty while network runs.
	bridgeapp.ApplyBluffetDefaults(&cfg)

	a := app.NewWithID("com.keweenawendurance.reader")
	w := newReaderWindow(a, cfg, cfgPath)
	w.ShowAndRun()
}
