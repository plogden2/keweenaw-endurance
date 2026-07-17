package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/keweenaw-endurance/backend/internal/config"
	"github.com/keweenaw-endurance/backend/internal/database"
	"github.com/keweenaw-endurance/backend/internal/handlers"
	"github.com/keweenaw-endurance/backend/internal/middleware"
	"github.com/keweenaw-endurance/backend/internal/services"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize database
	db, err := database.Initialize(cfg.Database)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer database.Close(db)

	// Initialize services
	svc := services.NewServices(db, cfg)

	// Initialize handlers
	handlers := handlers.NewHandlers(svc)

	// Setup Gin router
	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(gin.Logger())
	router.Use(gin.Recovery())
	router.Use(middleware.CORS())
	router.Use(middleware.Security())

	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "healthy",
			"timestamp": time.Now().Unix(),
		})
	})

	auth := middleware.JWTAuth(svc.Auth)
	adminOnly := []gin.HandlerFunc{auth, middleware.RequireRoles(services.RoleAdmin, services.RoleOwner)}
	timerWrite := []gin.HandlerFunc{auth, middleware.RequireRoles(services.RoleAdmin, services.RoleOwner, services.RoleTimer)}

	// API routes
	api := router.Group("/api")
	{
		api.POST("/auth/login", handlers.Login)
		// Public PIN → admin JWT; existing adminOnly/timerWrite middleware then applies.
		api.POST("/auth/pin", handlers.ExchangePIN)

		// Event routes
		events := api.Group("/events")
		{
			events.GET("", handlers.GetEvents)
			events.POST("", append(adminOnly, handlers.CreateEvent)...)
			events.GET("/:id", handlers.GetEvent)
			events.PUT("/:id", append(adminOnly, handlers.UpdateEvent)...)
			events.DELETE("/:id", append(adminOnly, handlers.DeleteEvent)...)
			events.GET("/:id/live", handlers.GetEventLive)
			events.POST("/:id/scans", handlers.ProcessEventScan)
			events.GET("/:id/live-csv", append(adminOnly, handlers.GetLiveCSV)...)
			events.GET("/:id/live-csv/status", append(adminOnly, handlers.GetLiveCSVStatus)...)
			events.POST("/:id/import.csv", append(adminOnly, handlers.ImportCSV)...)
		}

		// Station routes (current reader laptop config)
		stations := api.Group("/stations")
		{
			stations.GET("/current", handlers.GetCurrentStation)
			stations.PUT("/current", append(adminOnly, handlers.PutCurrentStation)...)
		}

		// Race routes
		races := api.Group("/races")
		{
			races.GET("", handlers.GetRaces)
			races.POST("", append(adminOnly, handlers.CreateRace)...)
			races.GET("/:id/checkpoints", handlers.GetCheckpointsByRace)
			races.POST("/:id/checkpoints", append(adminOnly, handlers.CreateCheckpoint)...)
			races.GET("/:id/categories", handlers.GetCategoriesByRace)
			races.POST("/:id/categories", append(adminOnly, handlers.CreateCategory)...)
			races.GET("/:id/participants", handlers.GetRaceParticipants)
			races.POST("/:id/participants", append(adminOnly, handlers.CreateRaceParticipant)...)
			races.GET("/:id/participants/:participantId/tags", handlers.GetParticipantTags)
			races.POST("/:id/participants/:participantId/tags", append(adminOnly, handlers.PostParticipantTag)...)
			races.POST("/:id/start", append(adminOnly, handlers.StartRace)...)
			races.POST("/:id/finish", append(adminOnly, handlers.FinishRace)...)
			races.GET("/:id", handlers.GetRace)
			races.PUT("/:id", append(adminOnly, handlers.UpdateRace)...)
			races.DELETE("/:id", append(adminOnly, handlers.DeleteRace)...)
		}

		// Checkpoint routes
		checkpoints := api.Group("/checkpoints")
		{
			checkpoints.GET("/:id", handlers.GetCheckpoint)
			checkpoints.PUT("/:id", append(adminOnly, handlers.UpdateCheckpoint)...)
			checkpoints.DELETE("/:id", append(adminOnly, handlers.DeleteCheckpoint)...)
		}

		// Category routes
		categories := api.Group("/categories")
		{
			categories.GET("/:id", handlers.GetCategory)
			categories.PUT("/:id", append(adminOnly, handlers.UpdateCategory)...)
			categories.DELETE("/:id", append(adminOnly, handlers.DeleteCategory)...)
		}

		// Participant routes
		participants := api.Group("/participants")
		{
			participants.GET("", handlers.GetParticipants)
			participants.POST("", append(adminOnly, handlers.CreateParticipant)...)
			participants.GET("/:id", handlers.GetParticipant)
			participants.PUT("/:id", append(adminOnly, handlers.UpdateParticipant)...)
			participants.DELETE("/:id", append(adminOnly, handlers.DeleteParticipant)...)
		}

		// Timing routes
		timing := api.Group("/timing")
		{
			timing.GET("/live/:raceId", handlers.GetLiveTiming)
			timing.POST("/record", append(timerWrite, handlers.CreateTimingRecord)...)
			timing.PUT("/records/:id", append(timerWrite, handlers.UpdateTimingRecord)...)
			timing.GET("/results/:raceId", handlers.GetRaceResults)
			timing.GET("/leaderboard/:raceId", handlers.GetLeaderboard)
		}

		// Karaoke bonus (open like scans — no re-PIN on armed station)
		api.POST("/timing-records/:id/karaoke-bonus", handlers.CreateKaraokeBonus)

		// RFID routes (scan public; write-tag admin; inject gated by config)
		rfid := api.Group("/rfid")
		{
			rfid.POST("/write-tag", append(adminOnly, handlers.WriteRFIDTag)...)
			rfid.GET("/read-payload", append(adminOnly, handlers.ReadRFIDPayload)...)
			rfid.GET("/scan/:uid", handlers.ScanRFIDTag)
			rfid.GET("/stream", handlers.StreamRFIDTags)
			rfid.POST("/inject", handlers.InjectRFIDTag)
			rfid.POST("/manual-entry", append(timerWrite, handlers.ManualTimingEntry)...)
			rfid.GET("/sync-status", handlers.GetSyncStatus)
			rfid.POST("/sync-pending", append(timerWrite, handlers.SyncPendingRecords)...)
			rfid.GET("/bridge", handlers.BridgeWebSocket)
			rfid.GET("/bridge/status", handlers.GetBridgeStatus)
		}

		syncGroup := api.Group("/sync")
		{
			syncGroup.POST("/push", handlers.PushSync)
			syncGroup.POST("/pull", handlers.PullSync)
			syncGroup.POST("/ingest", handlers.IngestSync)
			syncGroup.GET("/export", handlers.ExportSync)
		}
	}

	// Background ticker: auto-start scheduled races when start_time is reached
	autoStartCtx, autoStartCancel := context.WithCancel(context.Background())
	defer autoStartCancel()
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-autoStartCtx.Done():
				return
			case now := <-ticker.C:
				if n, err := svc.Races.AutoStartDueRaces(now); err != nil {
					log.Printf("auto-start races: %v", err)
				} else if n > 0 {
					log.Printf("auto-started %d race(s)", n)
				}
			}
		}
	}()

	// Continuous Proxmark3 / mock reader poll → WebSocket fan-out.
	// Hardware CLI spawns a process per poll and needs exclusive COM access;
	// keep the interval long enough that Poll finishes before the next tick.
	pollInterval := 200 * time.Millisecond
	if cfg.RFID.Hardware {
		pollInterval = 1500 * time.Millisecond
	}
	pollCtx, pollCancel := context.WithCancel(context.Background())
	defer pollCancel()
	svc.RFID.StartPolling(pollCtx, pollInterval, func() string {
		return svc.Stations.CurrentDeviceID()
	})

	// Start server
	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: router,
	}

	// Graceful shutdown
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	log.Printf("Server started on port %s", cfg.Port)

	// Wait for interrupt signal to gracefully shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")
	autoStartCancel()
	pollCancel()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}
