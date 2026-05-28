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
			"status": "healthy",
			"timestamp": time.Now().Unix(),
		})
	})

	// API routes
	api := router.Group("/api")
	{
		// Event routes
		events := api.Group("/events")
		{
			events.GET("", handlers.GetEvents)
			events.POST("", handlers.CreateEvent)
			events.GET("/:id", handlers.GetEvent)
			events.PUT("/:id", handlers.UpdateEvent)
			events.DELETE("/:id", handlers.DeleteEvent)
		}

		// Race routes
		races := api.Group("/races")
		{
			races.GET("", handlers.GetRaces)
			races.POST("", handlers.CreateRace)
			races.GET("/:id", handlers.GetRace)
			races.PUT("/:id", handlers.UpdateRace)
			races.DELETE("/:id", handlers.DeleteRace)
		}

		// Participant routes
		participants := api.Group("/participants")
		{
			participants.GET("", handlers.GetParticipants)
			participants.POST("", handlers.CreateParticipant)
			participants.GET("/:id", handlers.GetParticipant)
			participants.PUT("/:id", handlers.UpdateParticipant)
			participants.DELETE("/:id", handlers.DeleteParticipant)
		}

		// Timing routes
		timing := api.Group("/timing")
		{
			timing.GET("/live/:raceId", handlers.GetLiveTiming)
			timing.POST("/record", handlers.CreateTimingRecord)
			timing.GET("/results/:raceId", handlers.GetRaceResults)
			timing.GET("/leaderboard/:raceId", handlers.GetLeaderboard)
		}

		// RFID routes
		rfid := api.Group("/rfid")
		{
			rfid.POST("/write-tag", handlers.WriteRFIDTag)
			rfid.GET("/scan/:uid", handlers.ScanRFIDTag)
			rfid.POST("/manual-entry", handlers.ManualTimingEntry)
			rfid.GET("/sync-status", handlers.GetSyncStatus)
		}
	}

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

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}