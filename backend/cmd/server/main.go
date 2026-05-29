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

		// Event routes
		events := api.Group("/events")
		{
			events.GET("", handlers.GetEvents)
			events.POST("", append(adminOnly, handlers.CreateEvent)...)
			events.GET("/:id", handlers.GetEvent)
			events.PUT("/:id", append(adminOnly, handlers.UpdateEvent)...)
			events.DELETE("/:id", append(adminOnly, handlers.DeleteEvent)...)
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

		// RFID routes
		rfid := api.Group("/rfid")
		{
			rfid.POST("/write-tag", append(adminOnly, handlers.WriteRFIDTag)...)
			rfid.GET("/scan/:uid", handlers.ScanRFIDTag)
			rfid.POST("/manual-entry", append(timerWrite, handlers.ManualTimingEntry)...)
			rfid.GET("/sync-status", handlers.GetSyncStatus)
			rfid.POST("/sync-pending", append(timerWrite, handlers.SyncPendingRecords)...)
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
