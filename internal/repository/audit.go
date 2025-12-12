package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// AuditLogEntry represents an audit log record
type AuditLogEntry struct {
	ID           int64
	Timestamp    time.Time
	Actor        string
	Action       string
	ResourceType string
	ResourceID   string
	Details      map[string]interface{}
}

// AuditRepository handles audit log database operations
type AuditRepository struct {
	db *sql.DB
}

// NewAuditRepository creates a new AuditRepository
func NewAuditRepository(db *sql.DB) *AuditRepository {
	return &AuditRepository{db: db}
}

// Log creates a new audit log entry
func (r *AuditRepository) Log(ctx context.Context, actor, action, resourceType, resourceID string, details map[string]interface{}) error {
	detailsJSON, err := json.Marshal(details)
	if err != nil {
		return fmt.Errorf("failed to marshal details: %w", err)
	}

	query := `INSERT INTO audit_log (actor, action, resource_type, resource_id, details) VALUES (?, ?, ?, ?, ?)`
	_, err = r.db.ExecContext(ctx, query, actor, action, resourceType, resourceID, detailsJSON)
	if err != nil {
		return fmt.Errorf("failed to insert audit log: %w", err)
	}

	return nil
}

// List returns audit log entries with pagination and optional filters
func (r *AuditRepository) List(ctx context.Context, pageSize, offset int, resourceType, resourceID, actor string, startTime, endTime *time.Time) ([]AuditLogEntry, error) {
	query := `SELECT id, timestamp, actor, action, resource_type, resource_id, details
	          FROM audit_log WHERE 1=1`
	args := []interface{}{}

	if resourceType != "" {
		query += " AND resource_type = ?"
		args = append(args, resourceType)
	}

	if resourceID != "" {
		query += " AND resource_id = ?"
		args = append(args, resourceID)
	}

	if actor != "" {
		query += " AND actor = ?"
		args = append(args, actor)
	}

	if startTime != nil {
		query += " AND timestamp >= ?"
		args = append(args, startTime)
	}

	if endTime != nil {
		query += " AND timestamp <= ?"
		args = append(args, endTime)
	}

	query += " ORDER BY timestamp DESC LIMIT ? OFFSET ?"
	args = append(args, pageSize, offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list audit logs: %w", err)
	}
	defer rows.Close()

	var entries []AuditLogEntry
	for rows.Next() {
		var e AuditLogEntry
		var detailsJSON []byte
		if err := rows.Scan(&e.ID, &e.Timestamp, &e.Actor, &e.Action, &e.ResourceType, &e.ResourceID, &detailsJSON); err != nil {
			return nil, fmt.Errorf("failed to scan audit log: %w", err)
		}

		if len(detailsJSON) > 0 {
			if err := json.Unmarshal(detailsJSON, &e.Details); err != nil {
				// Log but don't fail on JSON parse errors
				e.Details = map[string]interface{}{"raw": string(detailsJSON)}
			}
		}

		entries = append(entries, e)
	}

	return entries, rows.Err()
}
