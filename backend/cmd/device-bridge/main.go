package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/keweenaw-endurance/backend/internal/bridgeapp"
)

func main() {
	cfg, err := bridgeapp.LoadConfig("")
	if err != nil {
		log.Fatalf("config: %v", err)
	}
	// Preserve prior behavior: env is the primary source for headless; empty path
	// still applies DefaultConfig + env. Require EVENT_ID explicitly.
	if cfg.EventID == "" {
		log.Fatal("EVENT_ID is required")
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	if err := bridgeapp.RunHeadless(ctx, cfg); err != nil {
		log.Fatal(err)
	}
}
