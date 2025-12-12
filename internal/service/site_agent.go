package service

import (
	"context"
	"fmt"
	"log"

	"github.com/yourorg/spire-workload-mgmt/internal/repository"
)

// SiteAgentService handles site agent sync operations
type SiteAgentService struct {
	syncRepo  *repository.SyncStatusRepository
	siteRepo  *repository.SiteRepository
	auditRepo *repository.AuditRepository
}

// NewSiteAgentService creates a new SiteAgentService
func NewSiteAgentService(syncRepo *repository.SyncStatusRepository, siteRepo *repository.SiteRepository,
	auditRepo *repository.AuditRepository) *SiteAgentService {
	return &SiteAgentService{
		syncRepo:  syncRepo,
		siteRepo:  siteRepo,
		auditRepo: auditRepo,
	}
}

// PendingEntry represents an entry pending sync
type PendingEntry struct {
	WorkloadEntryID string
	SpiffeID        string
	ParentID        string
	Selectors       []Selector
	TTL             int
}

// DeletionEntry represents an entry pending deletion
type DeletionEntry struct {
	WorkloadEntryID string
	SpireEntryID    string
}

// PollEntries returns entries pending sync for a site
func (s *SiteAgentService) PollEntries(ctx context.Context, siteID string, maxEntries int) ([]PendingEntry, error) {
	// Verify site exists
	site, err := s.siteRepo.Get(ctx, siteID)
	if err != nil {
		return nil, fmt.Errorf("failed to get site: %w", err)
	}
	if site == nil {
		return nil, fmt.Errorf("site not found: %s", siteID)
	}

	if maxEntries <= 0 || maxEntries > 100 {
		maxEntries = 10
	}

	entries, err := s.syncRepo.GetPendingEntries(ctx, siteID, maxEntries)
	if err != nil {
		return nil, fmt.Errorf("failed to get pending entries: %w", err)
	}

	result := make([]PendingEntry, len(entries))
	for i, e := range entries {
		selectors := make([]Selector, len(e.Selectors))
		for j, sel := range e.Selectors {
			selectors[j] = Selector{Type: sel.Type, Value: sel.Value}
		}
		result[i] = PendingEntry{
			WorkloadEntryID: e.WorkloadEntryID,
			SpiffeID:        e.SpiffeID,
			ParentID:        e.ParentID,
			Selectors:       selectors,
			TTL:             e.TTL,
		}
	}

	return result, nil
}

// ReportSyncResult reports the result of syncing an entry
func (s *SiteAgentService) ReportSyncResult(ctx context.Context, siteID, entryID string, success bool, spireEntryID, errorMsg string) error {
	var status string
	if success {
		status = "synced"
	} else {
		status = "failed"
	}

	if err := s.syncRepo.UpdateSyncStatus(ctx, siteID, entryID, status, spireEntryID, errorMsg); err != nil {
		return fmt.Errorf("failed to update sync status: %w", err)
	}

	// Update site last sync time
	if err := s.siteRepo.UpdateLastSyncAt(ctx, siteID); err != nil {
		log.Printf("Failed to update site last_sync_at: %v", err)
	}

	// Audit log for sync events
	details := map[string]interface{}{
		"site_id":        siteID,
		"success":        success,
		"spire_entry_id": spireEntryID,
	}
	if errorMsg != "" {
		details["error"] = errorMsg
	}
	if err := s.auditRepo.Log(ctx, "site-agent-"+siteID, "sync", "workload_entry", entryID, details); err != nil {
		log.Printf("Failed to write audit log: %v", err)
	}

	return nil
}

// PollDeletions returns entries pending deletion from a site
func (s *SiteAgentService) PollDeletions(ctx context.Context, siteID string, maxEntries int) ([]DeletionEntry, error) {
	// Verify site exists
	site, err := s.siteRepo.Get(ctx, siteID)
	if err != nil {
		return nil, fmt.Errorf("failed to get site: %w", err)
	}
	if site == nil {
		return nil, fmt.Errorf("site not found: %s", siteID)
	}

	if maxEntries <= 0 || maxEntries > 100 {
		maxEntries = 10
	}

	entries, err := s.syncRepo.GetDeletionEntries(ctx, siteID, maxEntries)
	if err != nil {
		return nil, fmt.Errorf("failed to get deletion entries: %w", err)
	}

	result := make([]DeletionEntry, len(entries))
	for i, e := range entries {
		result[i] = DeletionEntry{
			WorkloadEntryID: e.WorkloadEntryID,
			SpireEntryID:    e.SpireEntryID,
		}
	}

	return result, nil
}

// ReportDeletionResult reports the result of deleting an entry from SPIRE
func (s *SiteAgentService) ReportDeletionResult(ctx context.Context, siteID, entryID string, success bool, errorMsg string) error {
	if success {
		// Remove the site entry on successful deletion
		if err := s.syncRepo.RemoveSiteEntry(ctx, siteID, entryID); err != nil {
			return fmt.Errorf("failed to remove site entry: %w", err)
		}
	} else {
		// Update status to failed
		if err := s.syncRepo.UpdateSyncStatus(ctx, siteID, entryID, "failed", "", errorMsg); err != nil {
			return fmt.Errorf("failed to update sync status: %w", err)
		}
	}

	// Audit log for deletion events
	details := map[string]interface{}{
		"site_id": siteID,
		"success": success,
	}
	if errorMsg != "" {
		details["error"] = errorMsg
	}
	if err := s.auditRepo.Log(ctx, "site-agent-"+siteID, "delete_sync", "workload_entry", entryID, details); err != nil {
		log.Printf("Failed to write audit log: %v", err)
	}

	return nil
}
