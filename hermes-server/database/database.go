package database

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

var db *sql.DB

// Initialize opens SQLite connection and runs migrations
func Initialize() error {
	dbPath := getDBPath()
	log.Printf("Opening database at: %s", dbPath)

	// Create directory if path contains subdirectories
	dir := filepath.Dir(dbPath)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create database directory: %w", err)
		}
	}

	// Open database
	var err error
	db, err = sql.Open("sqlite3", dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	// Enable foreign keys
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		return fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	// Run migrations
	if err := migrate(); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	// Note: Users are managed by Aegis, not stored in Hermes database

	log.Println("Database initialized successfully")
	return nil
}

// GetDB returns the database connection
func GetDB() *sql.DB {
	return db
}

// Close closes the database connection
func Close() error {
	if db != nil {
		log.Println("Closing database connection")
		return db.Close()
	}
	return nil
}

// getDBPath returns the database file path from environment or default
func getDBPath() string {
	path := os.Getenv("HERMES_DB_PATH")
	if path == "" {
		// Default to root folder for both dev and container
		// In container, this will be mapped to a volume
		path = "./hermes.db"
	}
	return path
}
