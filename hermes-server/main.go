package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"nfcunha/hermes/hermes-server/api"
	"nfcunha/hermes/hermes-server/database"
	"nfcunha/hermes/hermes-server/services/aegis"
	"nfcunha/hermes/hermes-server/services/health"
	"nfcunha/hermes/hermes-server/services/proxy"
	"nfcunha/hermes/hermes-server/services/registry"
	"nfcunha/hermes/hermes-server/services/router"
	"nfcunha/hermes/hermes-server/utils/config"
)

func main() {
	log.Println("Starting Hermes API Gateway...")

	// Load configuration from environment variables
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

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
	aegisClient := aegis.NewClient(cfg.Auth.AegisURL, cfg.Auth.AegisTimeout)
	log.Println("Testing Aegis connectivity...")
	if err := aegisClient.Health(); err != nil {
		log.Fatalf("Failed to connect to Aegis at %s: %v", cfg.Auth.AegisURL, err)
	}
	log.Println("Aegis connection successful")

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

	// Create services
	rtr := router.New()
	prx := proxy.New()
	reg := registry.New(database.GetDB())

	// Start health checker
	checker := health.New(reg)
	go checker.Start()
	defer checker.Stop()

	// Register routes
	api.RegisterRoutes(engine, rtr, prx, reg, aegisClient, cfg.Auth.AegisURL)

	// Create HTTP server
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
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
