package core

import (
	"database/sql"
	"path/filepath"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"nfcunha/hermes/hermes-server/core/domain/service"
)

// TestPersistenceWithRealDatabase tests persistence using a file-based SQLite database
// This simulates a real scenario where Hermes restarts
func TestPersistenceWithRealDatabase(t *testing.T) {
	// Create temporary database file
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	// First run: Create database and register services
	db1, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	// Create table
	_, err = db1.Exec(`
		CREATE TABLE IF NOT EXISTS services (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			host TEXT NOT NULL,
			port INTEGER NOT NULL,
			protocol TEXT NOT NULL DEFAULT 'http',
			health_check_path TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'healthy',
			metadata TEXT,
			registered_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			last_checked_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			failure_count INTEGER DEFAULT 0,
			UNIQUE(name, host, port)
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	// Register services
	reg1 := NewServiceRegistry(db1)
	svc1 := service.NewService("api-service", "api.example.com", 8080, "/health")
	svc1.Metadata["version"] = "1.0.0"
	svc1.Metadata["env"] = "production"

	svc2 := service.NewService("auth-service", "auth.example.com", 8081, "/api/health")
	svc2.Metadata["version"] = "2.0.0"

	if err := reg1.Register(svc1); err != nil {
		t.Fatalf("Failed to register service 1: %v", err)
	}
	if err := reg1.Register(svc2); err != nil {
		t.Fatalf("Failed to register service 2: %v", err)
	}

	// Mark one unhealthy
	if err := reg1.UpdateStatus(svc2.ID, service.StatusUnhealthy); err != nil {
		t.Fatalf("Failed to update status: %v", err)
	}

	// Close first database connection (simulate shutdown)
	db1.Close()

	// Second run: Reopen database and verify services are loaded
	db2, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("Failed to reopen database: %v", err)
	}
	defer db2.Close()

	reg2 := NewServiceRegistry(db2)

	// Verify count
	services := reg2.List()
	if len(services) != 2 {
		t.Fatalf("Expected 2 services after restart, got %d", len(services))
	}

	// Verify service 1 details and metadata
	retrieved1, err := reg2.GetByID(svc1.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve service 1: %v", err)
	}
	if retrieved1.Name != "api-service" {
		t.Errorf("Expected name 'api-service', got '%s'", retrieved1.Name)
	}
	if retrieved1.Host != "api.example.com" {
		t.Errorf("Expected host 'api.example.com', got '%s'", retrieved1.Host)
	}
	if retrieved1.Port != 8080 {
		t.Errorf("Expected port 8080, got %d", retrieved1.Port)
	}
	if retrieved1.Metadata["version"] != "1.0.0" {
		t.Errorf("Expected version '1.0.0', got '%s'", retrieved1.Metadata["version"])
	}
	if retrieved1.Metadata["env"] != "production" {
		t.Errorf("Expected env 'production', got '%s'", retrieved1.Metadata["env"])
	}
	if retrieved1.Status != service.StatusHealthy {
		t.Errorf("Expected status healthy, got %s", retrieved1.Status)
	}

	// Verify service 2 status was persisted
	retrieved2, err := reg2.GetByID(svc2.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve service 2: %v", err)
	}
	if retrieved2.Status != service.StatusUnhealthy {
		t.Errorf("Expected status unhealthy, got %s", retrieved2.Status)
	}
	if retrieved2.Metadata["version"] != "2.0.0" {
		t.Errorf("Expected version '2.0.0', got '%s'", retrieved2.Metadata["version"])
	}

	// Verify GetByName still works after reload
	apiServices, err := reg2.GetByName("api-service")
	if err != nil {
		t.Fatalf("Failed to get by name: %v", err)
	}
	if len(apiServices) != 1 {
		t.Errorf("Expected 1 api-service, got %d", len(apiServices))
	}

	// Verify GetHealthy respects persisted status
	healthyAuth := reg2.GetHealthy("auth-service")
	if len(healthyAuth) != 0 {
		t.Errorf("Expected 0 healthy auth-service instances, got %d", len(healthyAuth))
	}

	// Test deregister persists
	if err := reg2.Deregister(svc1.ID); err != nil {
		t.Fatalf("Failed to deregister: %v", err)
	}

	// Verify in database
	var count int
	err = db2.QueryRow("SELECT COUNT(*) FROM services").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query database: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected 1 service in database after deregister, got %d", count)
	}

	// Third run: Verify deregister persisted
	db2.Close()
	db3, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("Failed to reopen database again: %v", err)
	}
	defer db3.Close()

	reg3 := NewServiceRegistry(db3)
	finalServices := reg3.List()
	if len(finalServices) != 1 {
		t.Fatalf("Expected 1 service after final restart, got %d", len(finalServices))
	}
	if finalServices[0].Name != "auth-service" {
		t.Errorf("Wrong service remained, expected auth-service, got %s", finalServices[0].Name)
	}
}

// TestDatabaseMigration verifies that the registry works with the actual migration system
func TestDatabaseMigration(t *testing.T) {
	// This test ensures our schema matches what the migration system expects
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Run the actual migration SQL (copy from migrations.go)
	migration := `
		CREATE TABLE IF NOT EXISTS services (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			host TEXT NOT NULL,
			port INTEGER NOT NULL,
			protocol TEXT NOT NULL DEFAULT 'http',
			health_check_path TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'healthy',
			metadata TEXT,
			registered_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			last_checked_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			failure_count INTEGER DEFAULT 0,
			UNIQUE(name, host, port)
		);

		CREATE INDEX IF NOT EXISTS idx_services_name ON services(name);
		CREATE INDEX IF NOT EXISTS idx_services_status ON services(status);
	`

	_, err = db.Exec(migration)
	if err != nil {
		t.Fatalf("Migration failed: %v", err)
	}

	// Verify registry works with migrated database
	reg := NewServiceRegistry(db)
	svc := service.NewService("test", "localhost", 8080, "/health")

	if err := reg.Register(svc); err != nil {
		t.Fatalf("Failed to register with migrated database: %v", err)
	}

	retrieved, err := reg.GetByID(svc.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve from migrated database: %v", err)
	}

	if retrieved.Name != "test" {
		t.Errorf("Service data incorrect after migration")
	}
}
