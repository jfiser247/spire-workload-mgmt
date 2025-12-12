package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
)

// PendingEntry represents an entry pending sync to a site
type PendingEntry struct {
	WorkloadEntryID string
	SpiffeID        string
	ParentID        string
	Selectors       []Selector
	TTL             int
}

// DeletionEntry represents an entry pending deletion from a site
type DeletionEntry struct {
	WorkloadEntryID string
	SpireEntryID    string
}

// SyncStatusRepository handles sync status database operations
type SyncStatusRepository struct {
	db *sql.DB
}

// NewSyncStatusRepository creates a new SyncStatusRepository
func NewSyncStatusRepository(db *sql.DB) *SyncStatusRepository {
	return &SyncStatusRepository{db: db}
}

// GetPendingEntries returns entries that need to be synced to a site
func (r *SyncStatusRepository) GetPendingEntries(ctx context.Context, siteID string, maxEntries int) ([]PendingEntry, error) {
	query := `SELECT we.id, we.spiffe_id, we.parent_id, we.selectors, we.ttl
	          FROM workload_entries we
	          JOIN site_workload_entries swe ON we.id = swe.workload_entry_id
	          WHERE swe.site_id = ? AND swe.sync_status = 'pending'
	          ORDER BY we.created_at ASC
	          LIMIT ?`

	rows, err := r.db.QueryContext(ctx, query, siteID, maxEntries)
	if err != nil {
		return nil, fmt.Errorf("failed to get pending entries: %w", err)
	}
	defer rows.Close()

	var entries []PendingEntry
	for rows.Next() {
		var e PendingEntry
		var selectorsJSON []byte
		if err := rows.Scan(&e.WorkloadEntryID, &e.SpiffeID, &e.ParentID, &selectorsJSON, &e.TTL); err != nil {
			return nil, fmt.Errorf("failed to scan pending entry: %w", err)
		}

		if err := json.Unmarshal(selectorsJSON, &e.Selectors); err != nil {
			return nil, fmt.Errorf("failed to unmarshal selectors: %w", err)
		}

		entries = append(entries, e)
	}

	return entries, rows.Err()
}

// GetDeletionEntries returns entries marked for deletion from a site
func (r *SyncStatusRepository) GetDeletionEntries(ctx context.Context, siteID string, maxEntries int) ([]DeletionEntry, error) {
	query := `SELECT swe.workload_entry_id, swe.spire_entry_id
	          FROM site_workload_entries swe
	          WHERE swe.site_id = ? AND swe.sync_status = 'deleting' AND swe.spire_entry_id IS NOT NULL
	          LIMIT ?`

	rows, err := r.db.QueryContext(ctx, query, siteID, maxEntries)
	if err != nil {
		return nil, fmt.Errorf("failed to get deletion entries: %w", err)
	}
	defer rows.Close()

	var entries []DeletionEntry
	for rows.Next() {
		var e DeletionEntry
		var spireEntryID sql.NullString
		if err := rows.Scan(&e.WorkloadEntryID, &spireEntryID); err != nil {
			return nil, fmt.Errorf("failed to scan deletion entry: %w", err)
		}
		if spireEntryID.Valid {
			e.SpireEntryID = spireEntryID.String
		}
		entries = append(entries, e)
	}

	return entries, rows.Err()
}

// UpdateSyncStatus updates the sync status for an entry at a site
func (r *SyncStatusRepository) UpdateSyncStatus(ctx context.Context, siteID, entryID string, status string, spireEntryID string, errorMsg string) error {
	var query string
	var args []interface{}

	if status == "synced" {
		query = `UPDATE site_workload_entries
		         SET sync_status = ?, spire_entry_id = ?, last_sync_at = NOW(), sync_error = NULL
		         WHERE site_id = ? AND workload_entry_id = ?`
		args = []interface{}{status, spireEntryID, siteID, entryID}
	} else if status == "failed" {
		query = `UPDATE site_workload_entries
		         SET sync_status = ?, sync_error = ?, last_sync_at = NOW()
		         WHERE site_id = ? AND workload_entry_id = ?`
		args = []interface{}{status, errorMsg, siteID, entryID}
	} else {
		query = `UPDATE site_workload_entries
		         SET sync_status = ?
		         WHERE site_id = ? AND workload_entry_id = ?`
		args = []interface{}{status, siteID, entryID}
	}

	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update sync status: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("site entry not found")
	}

	return nil
}

// RemoveSiteEntry removes the site assignment after successful deletion
func (r *SyncStatusRepository) RemoveSiteEntry(ctx context.Context, siteID, entryID string) error {
	query := `DELETE FROM site_workload_entries WHERE site_id = ? AND workload_entry_id = ?`
	_, err := r.db.ExecContext(ctx, query, siteID, entryID)
	if err != nil {
		return fmt.Errorf("failed to remove site entry: %w", err)
	}
	return nil
}

// GetSyncStatuses returns sync statuses for an entry across all sites
func (r *SyncStatusRepository) GetSyncStatuses(ctx context.Context, entryID string) ([]SiteWorkloadEntry, error) {
	query := `SELECT swe.site_id, s.name, swe.workload_entry_id, swe.sync_status,
	                 swe.spire_entry_id, swe.last_sync_at, swe.sync_error
	          FROM site_workload_entries swe
	          JOIN sites s ON swe.site_id = s.id
	          WHERE swe.workload_entry_id = ?
	          ORDER BY s.name`

	rows, err := r.db.QueryContext(ctx, query, entryID)
	if err != nil {
		return nil, fmt.Errorf("failed to get sync statuses: %w", err)
	}
	defer rows.Close()

	var statuses []SiteWorkloadEntry
	for rows.Next() {
		var s SiteWorkloadEntry
		if err := rows.Scan(&s.SiteID, &s.SiteName, &s.WorkloadEntryID, &s.SyncStatus,
			&s.SpireEntryID, &s.LastSyncAt, &s.SyncError); err != nil {
			return nil, fmt.Errorf("failed to scan sync status: %w", err)
		}
		statuses = append(statuses, s)
	}

	return statuses, rows.Err()
}
