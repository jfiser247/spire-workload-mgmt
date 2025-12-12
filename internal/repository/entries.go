package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Selector represents a workload selector
type Selector struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

// WorkloadEntry represents a workload entry
type WorkloadEntry struct {
	ID          string
	SpiffeID    string
	ParentID    string
	Selectors   []Selector
	TTL         int
	Description string
	CreatedBy   string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// SiteWorkloadEntry represents the sync status of an entry at a site
type SiteWorkloadEntry struct {
	SiteID          string
	SiteName        string
	WorkloadEntryID string
	SyncStatus      string
	SpireEntryID    *string
	LastSyncAt      *time.Time
	SyncError       *string
}

// WorkloadEntryWithSites combines an entry with its site statuses
type WorkloadEntryWithSites struct {
	WorkloadEntry
	SiteStatuses []SiteWorkloadEntry
}

// EntryRepository handles workload entry database operations
type EntryRepository struct {
	db *sql.DB
}

// NewEntryRepository creates a new EntryRepository
func NewEntryRepository(db *sql.DB) *EntryRepository {
	return &EntryRepository{db: db}
}

// Create creates a new workload entry with site assignments
func (r *EntryRepository) Create(ctx context.Context, entry *WorkloadEntry, siteIDs []string) (*WorkloadEntry, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Generate UUID if not set
	if entry.ID == "" {
		entry.ID = uuid.New().String()
	}

	// Serialize selectors to JSON
	selectorsJSON, err := json.Marshal(entry.Selectors)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal selectors: %w", err)
	}

	// Insert workload entry
	query := `INSERT INTO workload_entries (id, spiffe_id, parent_id, selectors, ttl, description, created_by)
	          VALUES (?, ?, ?, ?, ?, ?, ?)`
	_, err = tx.ExecContext(ctx, query, entry.ID, entry.SpiffeID, entry.ParentID,
		selectorsJSON, entry.TTL, entry.Description, entry.CreatedBy)
	if err != nil {
		return nil, fmt.Errorf("failed to insert workload entry: %w", err)
	}

	// Create site assignments with pending status
	if len(siteIDs) > 0 {
		assignQuery := `INSERT INTO site_workload_entries (site_id, workload_entry_id, sync_status) VALUES (?, ?, 'pending')`
		for _, siteID := range siteIDs {
			_, err = tx.ExecContext(ctx, assignQuery, siteID, entry.ID)
			if err != nil {
				return nil, fmt.Errorf("failed to assign entry to site %s: %w", siteID, err)
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Return the created entry
	return r.Get(ctx, entry.ID)
}

// Get returns a workload entry by ID with its site statuses
func (r *EntryRepository) Get(ctx context.Context, id string) (*WorkloadEntryWithSites, error) {
	// Get entry
	query := `SELECT id, spiffe_id, parent_id, selectors, ttl, description, created_by, created_at, updated_at
	          FROM workload_entries WHERE id = ?`

	var entry WorkloadEntry
	var selectorsJSON []byte
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&entry.ID, &entry.SpiffeID, &entry.ParentID, &selectorsJSON,
		&entry.TTL, &entry.Description, &entry.CreatedBy, &entry.CreatedAt, &entry.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get workload entry: %w", err)
	}

	// Parse selectors
	if err := json.Unmarshal(selectorsJSON, &entry.Selectors); err != nil {
		return nil, fmt.Errorf("failed to unmarshal selectors: %w", err)
	}

	// Get site statuses
	siteStatuses, err := r.getSiteStatuses(ctx, id)
	if err != nil {
		return nil, err
	}

	return &WorkloadEntryWithSites{
		WorkloadEntry: entry,
		SiteStatuses:  siteStatuses,
	}, nil
}

// List returns workload entries with pagination
func (r *EntryRepository) List(ctx context.Context, pageSize int, offset int, siteID string, spiffeIDPrefix string) ([]WorkloadEntryWithSites, int, error) {
	// Build query with optional filters
	baseQuery := `FROM workload_entries we`
	whereClause := " WHERE 1=1"
	args := []interface{}{}

	if siteID != "" {
		baseQuery += ` JOIN site_workload_entries swe ON we.id = swe.workload_entry_id`
		whereClause += " AND swe.site_id = ?"
		args = append(args, siteID)
	}

	if spiffeIDPrefix != "" {
		whereClause += " AND we.spiffe_id LIKE ?"
		args = append(args, spiffeIDPrefix+"%")
	}

	// Get total count
	countQuery := "SELECT COUNT(DISTINCT we.id) " + baseQuery + whereClause
	var totalCount int
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&totalCount); err != nil {
		return nil, 0, fmt.Errorf("failed to count entries: %w", err)
	}

	// Get entries
	selectQuery := `SELECT DISTINCT we.id, we.spiffe_id, we.parent_id, we.selectors, we.ttl,
	                we.description, we.created_by, we.created_at, we.updated_at ` +
		baseQuery + whereClause + " ORDER BY we.created_at DESC LIMIT ? OFFSET ?"
	args = append(args, pageSize, offset)

	rows, err := r.db.QueryContext(ctx, selectQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list entries: %w", err)
	}
	defer rows.Close()

	var entries []WorkloadEntryWithSites
	for rows.Next() {
		var entry WorkloadEntry
		var selectorsJSON []byte
		if err := rows.Scan(&entry.ID, &entry.SpiffeID, &entry.ParentID, &selectorsJSON,
			&entry.TTL, &entry.Description, &entry.CreatedBy, &entry.CreatedAt, &entry.UpdatedAt); err != nil {
			return nil, 0, fmt.Errorf("failed to scan entry: %w", err)
		}

		if err := json.Unmarshal(selectorsJSON, &entry.Selectors); err != nil {
			return nil, 0, fmt.Errorf("failed to unmarshal selectors: %w", err)
		}

		// Get site statuses for this entry
		siteStatuses, err := r.getSiteStatuses(ctx, entry.ID)
		if err != nil {
			return nil, 0, err
		}

		entries = append(entries, WorkloadEntryWithSites{
			WorkloadEntry: entry,
			SiteStatuses:  siteStatuses,
		})
	}

	return entries, totalCount, rows.Err()
}

// Delete deletes a workload entry (site assignments cascade delete)
func (r *EntryRepository) Delete(ctx context.Context, id string) error {
	// First mark all site entries as deleting
	updateQuery := `UPDATE site_workload_entries SET sync_status = 'deleting' WHERE workload_entry_id = ?`
	_, err := r.db.ExecContext(ctx, updateQuery, id)
	if err != nil {
		return fmt.Errorf("failed to mark entries for deletion: %w", err)
	}

	// Delete the entry (cascade will handle site_workload_entries)
	deleteQuery := `DELETE FROM workload_entries WHERE id = ?`
	result, err := r.db.ExecContext(ctx, deleteQuery, id)
	if err != nil {
		return fmt.Errorf("failed to delete entry: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("entry not found")
	}

	return nil
}

// AssignToSites assigns an entry to additional sites
func (r *EntryRepository) AssignToSites(ctx context.Context, entryID string, siteIDs []string) error {
	query := `INSERT IGNORE INTO site_workload_entries (site_id, workload_entry_id, sync_status) VALUES (?, ?, 'pending')`

	for _, siteID := range siteIDs {
		_, err := r.db.ExecContext(ctx, query, siteID, entryID)
		if err != nil {
			return fmt.Errorf("failed to assign entry to site %s: %w", siteID, err)
		}
	}

	return nil
}

// getSiteStatuses returns site sync statuses for an entry
func (r *EntryRepository) getSiteStatuses(ctx context.Context, entryID string) ([]SiteWorkloadEntry, error) {
	query := `SELECT swe.site_id, s.name, swe.workload_entry_id, swe.sync_status,
	                 swe.spire_entry_id, swe.last_sync_at, swe.sync_error
	          FROM site_workload_entries swe
	          JOIN sites s ON swe.site_id = s.id
	          WHERE swe.workload_entry_id = ?
	          ORDER BY s.name`

	rows, err := r.db.QueryContext(ctx, query, entryID)
	if err != nil {
		return nil, fmt.Errorf("failed to get site statuses: %w", err)
	}
	defer rows.Close()

	var statuses []SiteWorkloadEntry
	for rows.Next() {
		var s SiteWorkloadEntry
		if err := rows.Scan(&s.SiteID, &s.SiteName, &s.WorkloadEntryID, &s.SyncStatus,
			&s.SpireEntryID, &s.LastSyncAt, &s.SyncError); err != nil {
			return nil, fmt.Errorf("failed to scan site status: %w", err)
		}
		statuses = append(statuses, s)
	}

	return statuses, rows.Err()
}
