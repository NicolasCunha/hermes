package registry

import (
	"database/sql"
	"os"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"nfcunha/hermes/hermes-server/domain/service"
)

// setupTestDB creates an in-memory SQLite database for testing
func setupTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	// Create services table
	_, err = db.Exec(`
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
		t.Fatalf("Failed to create test table: %v", err)
	}

	return db
}

func TestMain(m *testing.M) {
	// Run tests
	code := m.Run()
	os.Exit(code)
}

func TestRegistry_Register(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	reg := New(db)
	svc := service.NewService("test-service", "localhost", 8080, "/health")

	err := reg.Register(svc)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Verify service is registered
	retrieved, err := reg.GetByID(svc.ID)
	if err != nil {
		t.Fatalf("Expected to retrieve service, got error: %v", err)
	}

	if retrieved.ID != svc.ID {
		t.Errorf("Expected ID %s, got %s", svc.ID, retrieved.ID)
	}
}

func TestRegistry_Register_Duplicate(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	reg := New(db)
	svc := service.NewService("test-service", "localhost", 8080, "/health")

	reg.Register(svc)
	err := reg.Register(svc)

	if err == nil {
		t.Error("Expected error when registering duplicate service")
	}

	expectedErr := "service already registered"
	if err != nil && err.Error()[:len(expectedErr)] != expectedErr {
		t.Errorf("Expected error message to start with '%s', got '%s'", expectedErr, err.Error())
	}
}

func TestRegistry_Register_DuplicateNameHostPort(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	reg := New(db)
	svc1 := service.NewService("api-service", "localhost", 8080, "/health")
	svc2 := service.NewService("api-service", "localhost", 8080, "/health")

	err := reg.Register(svc1)
	if err != nil {
		t.Fatalf("Expected no error registering first service, got %v", err)
	}

	err = reg.Register(svc2)
	if err == nil {
		t.Error("Expected error when registering service with duplicate name/host/port")
	}

	expectedErr := "service with name 'api-service' already registered at localhost:8080"
	if err != nil && err.Error() != expectedErr {
		t.Errorf("Expected error '%s', got '%s'", expectedErr, err.Error())
	}
}

func TestRegistry_Register_SameNameDifferentHost(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	reg := New(db)
	svc1 := service.NewService("api-service", "host1.com", 8080, "/health")
	svc2 := service.NewService("api-service", "host2.com", 8080, "/health")

	err := reg.Register(svc1)
	if err != nil {
		t.Fatalf("Expected no error registering first service, got %v", err)
	}

	err = reg.Register(svc2)
	if err != nil {
		t.Fatalf("Expected no error registering service with same name but different host, got %v", err)
	}

	// Verify both are registered
	services, err := reg.GetByName("api-service")
	if err != nil {
		t.Fatalf("Expected to retrieve services by name, got error: %v", err)
	}

	if len(services) != 2 {
		t.Errorf("Expected 2 services with name 'api-service', got %d", len(services))
	}
}

func TestRegistry_Register_DifferentNameSameHostPort(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	reg := New(db)
	svc1 := service.NewService("service-a", "localhost", 9000, "/health")
	svc2 := service.NewService("service-b", "localhost", 9000, "/health")

	err := reg.Register(svc1)
	if err != nil {
		t.Fatalf("Expected no error registering first service, got %v", err)
	}

	err = reg.Register(svc2)
	if err != nil {
		t.Fatalf("Expected no error registering different service on same host:port, got %v", err)
	}

	// Verify both are registered
	allServices := reg.List()
	if len(allServices) != 2 {
		t.Errorf("Expected 2 services, got %d", len(allServices))
	}
}

func TestRegistry_Deregister(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	reg := New(db)
	svc := service.NewService("test-service", "localhost", 8080, "/health")
	reg.Register(svc)

	err := reg.Deregister(svc.ID)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Verify service is deregistered
	_, err = reg.GetByID(svc.ID)
	if err == nil {
		t.Error("Expected error when retrieving deregistered service")
	}
}

func TestRegistry_Deregister_NotFound(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	reg := New(db)

	err := reg.Deregister("non-existent-id")
	if err == nil {
		t.Error("Expected error when deregistering non-existent service")
	}
}

func TestRegistry_GetByName(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	reg := New(db)
	
	svc1 := service.NewService("test-service", "localhost", 8080, "/health")
	svc2 := service.NewService("test-service", "localhost", 8081, "/health")
	svc3 := service.NewService("other-service", "localhost", 8082, "/health")

	reg.Register(svc1)
	reg.Register(svc2)
	reg.Register(svc3)

	instances, err := reg.GetByName("test-service")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(instances) != 2 {
		t.Errorf("Expected 2 instances, got %d", len(instances))
	}
}

func TestRegistry_GetByName_NotFound(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	reg := New(db)

	_, err := reg.GetByName("non-existent")
	if err == nil {
		t.Error("Expected error when getting non-existent service by name")
	}
}

func TestRegistry_GetHealthy(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	reg := New(db)
	
	svc1 := service.NewService("test-service", "localhost", 8080, "/health")
	svc2 := service.NewService("test-service", "localhost", 8081, "/health")
	
	reg.Register(svc1)
	reg.Register(svc2)

	// Mark one unhealthy
	svc2.MarkUnhealthy(1)

	healthy := reg.GetHealthy("test-service")
	if len(healthy) != 1 {
		t.Errorf("Expected 1 healthy instance, got %d", len(healthy))
	}

	if healthy[0].ID != svc1.ID {
		t.Errorf("Expected healthy service to be %s, got %s", svc1.ID, healthy[0].ID)
	}
}

func TestRegistry_List(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	reg := New(db)
	
	svc1 := service.NewService("service-1", "localhost", 8080, "/health")
	svc2 := service.NewService("service-2", "localhost", 8081, "/health")
	
	reg.Register(svc1)
	reg.Register(svc2)

	services := reg.List()
	if len(services) != 2 {
		t.Errorf("Expected 2 services, got %d", len(services))
	}
}

func TestRegistry_UpdateStatus(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	reg := New(db)
	svc := service.NewService("test-service", "localhost", 8080, "/health")
	reg.Register(svc)

	err := reg.UpdateStatus(svc.ID, service.StatusUnhealthy)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	retrieved, _ := reg.GetByID(svc.ID)
	if retrieved.Status != service.StatusUnhealthy {
		t.Errorf("Expected status %s, got %s", service.StatusUnhealthy, retrieved.Status)
	}
}

func TestRegistry_ConcurrentAccess(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	reg := New(db)
	done := make(chan bool)

	// Concurrent writes
	for i := 0; i < 10; i++ {
		go func(n int) {
			svc := service.NewService("test-service", "localhost", 8080+n, "/health")
			reg.Register(svc)
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify all services registered
	services := reg.List()
	if len(services) != 10 {
		t.Errorf("Expected 10 services, got %d", len(services))
	}
}

func TestRegistry_DatabasePersistence(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Create first registry and register services
	reg1 := New(db)
	svc1 := service.NewService("test-service-1", "localhost", 8080, "/health")
	svc2 := service.NewService("test-service-2", "localhost", 8081, "/health")

	if err := reg1.Register(svc1); err != nil {
		t.Fatalf("Failed to register service 1: %v", err)
	}
	if err := reg1.Register(svc2); err != nil {
		t.Fatalf("Failed to register service 2: %v", err)
	}

	// Update status of one service
	if err := reg1.UpdateStatus(svc2.ID, service.StatusUnhealthy); err != nil {
		t.Fatalf("Failed to update status: %v", err)
	}

	// Create second registry (simulates restart)
	reg2 := New(db)

	// Verify services were loaded from database
	loaded := reg2.List()
	if len(loaded) != 2 {
		t.Fatalf("Expected 2 services after reload, got %d", len(loaded))
	}

	// Verify service details
	retrieved1, err := reg2.GetByID(svc1.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve service 1: %v", err)
	}
	if retrieved1.Name != "test-service-1" || retrieved1.Port != 8080 {
		t.Errorf("Service 1 details incorrect after reload")
	}

	retrieved2, err := reg2.GetByID(svc2.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve service 2: %v", err)
	}
	if retrieved2.Status != service.StatusUnhealthy {
		t.Errorf("Expected status %s, got %s", service.StatusUnhealthy, retrieved2.Status)
	}

	// Verify deregister persists
	if err := reg2.Deregister(svc1.ID); err != nil {
		t.Fatalf("Failed to deregister: %v", err)
	}

	// Create third registry and verify deletion persisted
	reg3 := New(db)
	loaded = reg3.List()
	if len(loaded) != 1 {
		t.Fatalf("Expected 1 service after deregister, got %d", len(loaded))
	}
	if loaded[0].ID != svc2.ID {
		t.Errorf("Wrong service remained after deregister")
	}
}
