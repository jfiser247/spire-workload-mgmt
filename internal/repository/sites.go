package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// Site represents a SPIRE server site
type Site struct {
	ID                 string
	Name               string
	Region             string
	SpireServerAddress string
	TrustDomain        string
	LastSyncAt         *time.Time
	Status             string
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

// SiteRepository handles site database operations
type SiteRepository struct {
	db *sql.DB
}

// NewSiteRepository creates a new SiteRepository
func NewSiteRepository(db *sql.DB) *SiteRepository {
	return &SiteRepository{db: db}
}

// List returns all sites, optionally filtered by status
func (r *SiteRepository) List(ctx context.Context, status string) ([]Site, error) {
	query := `SELECT id, name, region, spire_server_address, trust_domain, last_sync_at, status, created_at, updated_at
	          FROM sites`
	args := []interface{}{}

	if status != "" {
		query += " WHERE status = ?"
		args = append(args, status)
	}

	query += " ORDER BY name"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list sites: %w", err)
	}
	defer rows.Close()

	var sites []Site
	for rows.Next() {
		var s Site
		if err := rows.Scan(&s.ID, &s.Name, &s.Region, &s.SpireServerAddress, &s.TrustDomain,
			&s.LastSyncAt, &s.Status, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan site: %w", err)
		}
		sites = append(sites, s)
	}

	return sites, rows.Err()
}

// Get returns a site by ID
func (r *SiteRepository) Get(ctx context.Context, id string) (*Site, error) {
	query := `SELECT id, name, region, spire_server_address, trust_domain, last_sync_at, status, created_at, updated_at
	          FROM sites WHERE id = ?`

	var s Site
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&s.ID, &s.Name, &s.Region, &s.SpireServerAddress, &s.TrustDomain,
		&s.LastSyncAt, &s.Status, &s.CreatedAt, &s.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get site: %w", err)
	}

	return &s, nil
}

// UpdateLastSyncAt updates the last sync timestamp for a site
func (r *SiteRepository) UpdateLastSyncAt(ctx context.Context, id string) error {
	query := `UPDATE sites SET last_sync_at = NOW() WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to update last_sync_at: %w", err)
	}
	return nil
}
