package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/anunay/mentis/internal/api/handlers"
	"github.com/anunay/mentis/internal/api/middleware"
	"github.com/anunay/mentis/internal/config"
	"github.com/anunay/mentis/internal/core/services"
	"github.com/anunay/mentis/internal/core/services/embedding"
	"github.com/anunay/mentis/internal/storage/postgres"
	"github.com/anunay/mentis/internal/storage/vector"
	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
	"github.com/sirupsen/logrus"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatal("Failed to load config:", err)
	}

	// Setup logging
	config.SetupLogging(cfg.Log.Level)

	// Connect to PostgreSQL
	db, err := sql.Open("postgres", cfg.Database.URL)
	if err != nil {
		logrus.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	// Test database connection
	if err := db.Ping(); err != nil {
		logrus.Fatal("Failed to ping database:", err)
	}
	logrus.Info("Connected to PostgreSQL")

	// Connect to vector database using factory pattern
	vectorRepo, err := vector.NewVectorRepository(&cfg.Vector)
	if err != nil {
		logrus.Fatal("Failed to create vector repository:", err)
	}
	logrus.Infof("Connected to vector database via provider: %s", cfg.Vector.Provider)

	// Initialize repositories
	artifactRepo := postgres.NewArtifactRepository(db)
	workflowRepo := postgres.NewWorkflowRepository(db)

	// Initialize services
	hashService := services.NewHashService()
	embeddingService, err := embedding.NewService(cfg.Embedding)
	if err != nil {
		logrus.Fatal("Failed to create embedding service:", err)
	}
	logrus.Infof("Using embedding provider: %s", cfg.Embedding.Provider)
	
	cacheService := services.NewCacheService(artifactRepo, vectorRepo, hashService)
	workflowService := services.NewWorkflowService(
		workflowRepo,
		artifactRepo,
		vectorRepo,
		embeddingService,
		hashService,
	)

	// Initialize handlers
	cacheHandler := handlers.NewCacheHandler(cacheService)
	workflowHandler := handlers.NewWorkflowHandler(workflowService)

	// Setup Gin router
	if cfg.Log.Level != "debug" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(middleware.ErrorHandlingMiddleware())
	router.Use(middleware.LoggingMiddleware())
	router.Use(middleware.CORSMiddleware())
	router.Use(middleware.RequestIDMiddleware())

	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "healthy",
			"timestamp": time.Now().UTC(),
			"service":   "mentis",
		})
	})

	// API routes
	v1 := router.Group("/v1")
	{
		cacheHandler.RegisterRoutes(v1)
		workflowHandler.RegisterRoutes(v1)

		// Quick lookup endpoints
		v1.GET("/lookup", cacheHandler.QuickLookup)
		v1.GET("/workflow/lookup", workflowHandler.QuickStepLookup)
	}

	// Create HTTP server
	server := &http.Server{
		Addr:    ":" + cfg.Server.Port,
		Handler: router,
	}

	// Start server in a goroutine
	go func() {
		logrus.Infof("Starting server on port %s", cfg.Server.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logrus.Fatal("Failed to start server:", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logrus.Info("Shutting down server...")

	// Give outstanding requests 30 seconds to complete
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logrus.Fatal("Server forced to shutdown:", err)
	}

	logrus.Info("Server exited")
}