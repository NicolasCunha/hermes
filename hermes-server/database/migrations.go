package database

import (
	"log"
)

// migrate runs all database migrations
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
	}

	for _, migration := range migrations {
		log.Printf("Running migration: %s", migration.name)
		if _, err := db.Exec(migration.sql); err != nil {
			return err
		}
		log.Printf("Migration completed: %s", migration.name)
	}

	if len(migrations) == 0 {
		log.Println("No migrations to run")
	}

	return nil
}
