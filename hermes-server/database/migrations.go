// Package database provides schema migrations for the Hermes database.
package database

import (
	"log"
)

// migrate runs all database migrations to create the schema.
// Creates two tables:
//   - services: stores registered service information
//   - health_check_logs: stores health check history
//
// Returns an error if any migration fails.
func migrate() error {
	migrations := []struct {
		name string
		sql  string
	}{
		{
			name: "create_services_table",
			sql: `
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
			`,
		},
		{
			name: "create_health_check_logs_table",
			sql: `
CREATE TABLE IF NOT EXISTS health_check_logs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    service_id TEXT NOT NULL,
    checked_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    status TEXT NOT NULL,
    error_message TEXT,
    response_time_ms INTEGER,
    response_body TEXT,
    FOREIGN KEY (service_id) REFERENCES services(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_health_logs_service ON health_check_logs(service_id);
CREATE INDEX IF NOT EXISTS idx_health_logs_checked_at ON health_check_logs(checked_at);
			`,
		},
	}

	for _, migration := range migrations {
		log.Printf("Running migration: %s", migration.name)
		if _, err := db.Exec(migration.sql); err != nil {
			log.Printf("Migration failed for %s: %v", migration.name, err)
			return err
		}
		log.Printf("Migration completed: %s", migration.name)
	}

	if len(migrations) == 0 {
		log.Println("No migrations to run")
	}

	return nil
}
