// Package healthlog defines the domain model for health check logs.
// It provides persistence for historical health check data.
package healthlog

import (
	"database/sql"
	"time"
)

// HealthLog represents a health check log entry.
// Each entry records the result of a single health check operation.
type HealthLog struct {
	ID             int64     `json:"id"`
	ServiceID      string    `json:"service_id"`
	CheckedAt      time.Time `json:"checked_at"`
	Status         string    `json:"status"`
	ErrorMessage   string    `json:"error_message,omitempty"`
	ResponseTimeMs int64     `json:"response_time_ms"`
	ResponseBody   string    `json:"response_body,omitempty"`
}

// Repository handles persistence of health check logs to the database.
type Repository struct {
	db *sql.DB
}

// NewRepository creates a new health log repository with the given database connection.
func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// Create stores a health check result in the database.
// Parameters:
//   - serviceID: unique ID of the service being checked
//   - status: "healthy", "unhealthy", or "error"
//   - errorMsg: error message if check failed (empty string if successful)
//   - responseBody: HTTP response body from the health endpoint
//   - responseTimeMs: response time in milliseconds
//
// Returns an error if the database operation fails.
func (r *Repository) Create(serviceID, status, errorMsg, responseBody string, responseTimeMs int64) error {
	if r.db == nil {
		return nil
	}

	query := `
		INSERT INTO health_check_logs (service_id, status, error_message, response_body, response_time_ms)
		VALUES (?, ?, ?, ?, ?)
	`

	var errorMsgPtr *string
	if errorMsg != "" {
		errorMsgPtr = &errorMsg
	}

	var responseBodyPtr *string
	if responseBody != "" {
		responseBodyPtr = &responseBody
	}

	_, err := r.db.Exec(query, serviceID, status, errorMsgPtr, responseBodyPtr, responseTimeMs)
	return err
}

// GetByServiceID retrieves health check logs for a specific service.
// Logs are returned in descending order (most recent first).
// The limit parameter controls the maximum number of logs to return.
func (r *Repository) GetByServiceID(serviceID string, limit int) ([]HealthLog, error) {
	if r.db == nil {
		return nil, nil
	}

	query := `
		SELECT id, service_id, checked_at, status, error_message, response_body, response_time_ms
		FROM health_check_logs
		WHERE service_id = ?
		ORDER BY checked_at DESC
		LIMIT ?
	`

	rows, err := r.db.Query(query, serviceID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []HealthLog
	for rows.Next() {
		var log HealthLog
		var errorMsg sql.NullString
		var responseBody sql.NullString
		err := rows.Scan(&log.ID, &log.ServiceID, &log.CheckedAt, &log.Status, &errorMsg, &responseBody, &log.ResponseTimeMs)
		if err != nil {
			return nil, err
		}
		if errorMsg.Valid {
			log.ErrorMessage = errorMsg.String
		}
		if responseBody.Valid {
			log.ResponseBody = responseBody.String
		}
		logs = append(logs, log)
	}

	return logs, rows.Err()
}
