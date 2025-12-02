// Package main is the entry point for the Hermes API Gateway server.
// It initializes all components, configures routes, and starts the HTTP server.
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"nfcunha/hermes/hermes-server/core"
	"nfcunha/hermes/hermes-server/core/bootstrap"
	"nfcunha/hermes/hermes-server/core/domain/healthlog"
	"nfcunha/hermes/hermes-server/database"
	"nfcunha/hermes/hermes-server/handler"
	"nfcunha/hermes/hermes-server/utils/config"
)

func main() {
	log.Println("Starting Hermes API Gateway...")

	// Load configuration from environment variables
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	log.Println("Configuration loaded successfully")
	// Initialize database
	if err := database.Initialize(); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	defer func() {
		if err := database.Close(); err != nil {
			log.Printf("Error closing database: %v", err)
		}
	}()

	// Initialize Aegis client
	aegisClient := core.NewAegisClient(cfg.Auth.AegisURL, cfg.Auth.AegisTimeout)
	log.Println("Testing Aegis connectivity...")
	if err := aegisClient.Health(); err != nil {
		log.Fatalf("Failed to connect to Aegis at %s: %v", cfg.Auth.AegisURL, err)
	}
	log.Println("Aegis connection successful")

	// Bootstrap admin user
	bootstrapper := bootstrap.NewAdminBootstrapper(
		cfg.Auth.AegisURL,
		cfg.Bootstrap.AdminUser,
		cfg.Bootstrap.AdminPassword,
	)
	if err := bootstrapper.EnsureAdminUser(); err != nil {
		log.Fatalf("Failed to bootstrap admin user: %v", err)
	}

	// Set Gin mode based on log level
	if config.IsDebugMode() {
		gin.SetMode(gin.DebugMode)
		log.Println("Running in DEBUG mode")
	} else {
		gin.SetMode(gin.ReleaseMode)
		log.Println("Running in RELEASE mode")
	}

	// Create Gin engine with logging middleware
	engine := gin.New()
	engine.Use(gin.Recovery())

	if config.IsDebugMode() {
		engine.Use(gin.Logger())
	}

	// Add CORS middleware to allow requests from React frontend
	engine.Use(handler.CORSMiddleware())

	// Create services
	prx := core.NewProxyService()
	reg := core.NewServiceRegistry(database.GetDB())

	// Create health log repository and health checker
	healthLogRepo := healthlog.NewRepository(database.GetDB())
	checker := core.NewHealthChecker(reg, healthLogRepo)
	go checker.Start()
	defer checker.Stop()

	// Register routes
	handler.RegisterRoutes(engine, prx, reg, aegisClient, cfg.Auth.AegisURL)

	// Create HTTP server
	addr := cfg.Server.Host + ":" + strconv.Itoa(cfg.Server.Port)
	server := &http.Server{
		Addr:           addr,
		Handler:        engine,
		ReadTimeout:    cfg.Server.ReadTimeout,
		WriteTimeout:   cfg.Server.WriteTimeout,
		IdleTimeout:    cfg.Server.IdleTimeout,
		MaxHeaderBytes: cfg.Server.MaxHeaderBytes,
	}

	// Start server in background
	go func() {
		log.Printf("Hermes API Gateway listening on %s", addr)
		log.Println("Management API available at: /hermes")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down gateway...")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Error during shutdown: %v", err)
	}

	log.Println("Gateway stopped gracefully")
}
