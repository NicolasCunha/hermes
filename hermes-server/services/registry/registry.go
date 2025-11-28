package registry

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"nfcunha/hermes/hermes-server/domain/service"
)

// Registry manages registered services with database persistence
type Registry struct {
	services map[string]*service.Service   // Key: service ID
	byName   map[string][]*service.Service // Key: service name
	mu       sync.RWMutex
	db       *sql.DB
}

// New creates a new service registry
func New(db *sql.DB) *Registry {
	r := &Registry{
		services: make(map[string]*service.Service),
		byName:   make(map[string][]*service.Service),
		db:       db,
	}

	// Load existing services from database
	if err := r.loadFromDatabase(); err != nil {
		log.Printf("Warning: failed to load services from database: %v", err)
	}

	return r
}

// Register adds a new service to the registry
func (r *Registry) Register(svc *service.Service) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.services[svc.ID]; exists {
		return fmt.Errorf("service already registered: %s", svc.ID)
	}

	// Check for duplicate (name, host, port) combination
	for _, existing := range r.services {
		if existing.Name == svc.Name && existing.Host == svc.Host && existing.Port == svc.Port {
			return fmt.Errorf("service with name '%s' already registered at %s:%d", svc.Name, svc.Host, svc.Port)
		}
	}

	r.services[svc.ID] = svc
	r.byName[svc.Name] = append(r.byName[svc.Name], svc)

	// Persist to database
	if err := r.saveToDatabase(svc); err != nil {
		log.Printf("Warning: failed to persist service to database: %v", err)
	}

	log.Printf("Service registered: %s (%s) at %s", svc.Name, svc.ID, svc.BaseURL())
	return nil
}

// Deregister removes a service from the registry
func (r *Registry) Deregister(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	svc, exists := r.services[id]
	if !exists {
		return fmt.Errorf("service not found: %s", id)
	}

	// Remove from services map
	delete(r.services, id)

	// Remove from byName map
	instances := r.byName[svc.Name]
	for i, instance := range instances {
		if instance.ID == id {
			r.byName[svc.Name] = append(instances[:i], instances[i+1:]...)
			break
		}
	}

	// Clean up empty name entry
	if len(r.byName[svc.Name]) == 0 {
		delete(r.byName, svc.Name)
	}

	// Remove from database
	if err := r.deleteFromDatabase(id); err != nil {
		log.Printf("Warning: failed to delete service from database: %v", err)
	}

	log.Printf("Service deregistered: %s (%s)", svc.Name, svc.ID)
	return nil
}

// GetByID retrieves service by ID
func (r *Registry) GetByID(id string) (*service.Service, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	svc, exists := r.services[id]
	if !exists {
		return nil, fmt.Errorf("service not found: %s", id)
	}

	return svc, nil
}

// GetByName retrieves all instances of a service by name
func (r *Registry) GetByName(name string) ([]*service.Service, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	instances, exists := r.byName[name]
	if !exists || len(instances) == 0 {
		return nil, fmt.Errorf("no instances found for service: %s", name)
	}

	return instances, nil
}

// GetHealthy retrieves all healthy instances of a service
func (r *Registry) GetHealthy(name string) []*service.Service {
	r.mu.RLock()
	defer r.mu.RUnlock()

	instances := r.byName[name]
	healthy := make([]*service.Service, 0)

	for _, svc := range instances {
		if svc.Status == service.StatusHealthy {
			healthy = append(healthy, svc)
		}
	}

	return healthy
}

// List retrieves all registered services
func (r *Registry) List() []*service.Service {
	r.mu.RLock()
	defer r.mu.RUnlock()

	services := make([]*service.Service, 0, len(r.services))
	for _, svc := range r.services {
		services = append(services, svc)
	}

	return services
}

// UpdateStatus updates service status (called by health checker)
func (r *Registry) UpdateStatus(id string, status service.Status) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	svc, exists := r.services[id]
	if !exists {
		return fmt.Errorf("service not found: %s", id)
	}

	svc.Status = status

	// Update database
	if err := r.updateStatusInDatabase(id, status); err != nil {
		log.Printf("Warning: failed to update service status in database: %v", err)
	}

	return nil
}

// loadFromDatabase loads all services from the database on startup
func (r *Registry) loadFromDatabase() error {
	rows, err := r.db.Query(`
		SELECT id, name, host, port, protocol, health_check_path, status, 
		       metadata, registered_at, last_checked_at, failure_count
		FROM services
	`)
	if err != nil {
		return fmt.Errorf("failed to query services: %w", err)
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		svc := &service.Service{
			Metadata: make(map[string]string),
		}
		var metadataJSON sql.NullString
		var registeredAt, lastCheckedAt string

		err := rows.Scan(
			&svc.ID, &svc.Name, &svc.Host, &svc.Port, &svc.Protocol,
			&svc.HealthCheckPath, &svc.Status, &metadataJSON,
			&registeredAt, &lastCheckedAt, &svc.FailureCount,
		)
		if err != nil {
			log.Printf("Warning: failed to scan service row: %v", err)
			continue
		}

		// Parse timestamps
		if svc.RegisteredAt, err = time.Parse(time.RFC3339, registeredAt); err != nil {
			svc.RegisteredAt = time.Now()
		}
		if svc.LastCheckedAt, err = time.Parse(time.RFC3339, lastCheckedAt); err != nil {
			svc.LastCheckedAt = time.Now()
		}

		// Parse metadata JSON
		if metadataJSON.Valid && metadataJSON.String != "" {
			if err := json.Unmarshal([]byte(metadataJSON.String), &svc.Metadata); err != nil {
				log.Printf("Warning: failed to parse metadata for service %s: %v", svc.ID, err)
			}
		}

		// Add to in-memory registry
		r.services[svc.ID] = svc
		r.byName[svc.Name] = append(r.byName[svc.Name], svc)
		count++
	}

	if count > 0 {
		log.Printf("Loaded %d services from database", count)
	}

	return rows.Err()
}

// saveToDatabase persists a service to the database
func (r *Registry) saveToDatabase(svc *service.Service) error {
	metadataJSON, err := json.Marshal(svc.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	_, err = r.db.Exec(`
		INSERT INTO services (
			id, name, host, port, protocol, health_check_path, status,
			metadata, registered_at, last_checked_at, failure_count
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		svc.ID, svc.Name, svc.Host, svc.Port, svc.Protocol,
		svc.HealthCheckPath, svc.Status, string(metadataJSON),
		svc.RegisteredAt.Format(time.RFC3339),
		svc.LastCheckedAt.Format(time.RFC3339),
		svc.FailureCount,
	)

	return err
}

// deleteFromDatabase removes a service from the database
func (r *Registry) deleteFromDatabase(id string) error {
	_, err := r.db.Exec("DELETE FROM services WHERE id = ?", id)
	return err
}

// updateStatusInDatabase updates service status and health check info in the database
func (r *Registry) updateStatusInDatabase(id string, status service.Status) error {
	svc, exists := r.services[id]
	if !exists {
		return fmt.Errorf("service not found in memory: %s", id)
	}

	_, err := r.db.Exec(`
		UPDATE services 
		SET status = ?, last_checked_at = ?, failure_count = ?
		WHERE id = ?
	`,
		status,
		svc.LastCheckedAt.Format(time.RFC3339),
		svc.FailureCount,
		id,
	)

	return err
}
