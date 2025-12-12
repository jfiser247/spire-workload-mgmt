package service

import (
	"context"
	"fmt"
	"log"

	"github.com/yourorg/spire-workload-mgmt/internal/repository"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// WorkloadEntryService implements the WorkloadEntryService gRPC interface
type WorkloadEntryService struct {
	entryRepo  *repository.EntryRepository
	siteRepo   *repository.SiteRepository
	syncRepo   *repository.SyncStatusRepository
	auditRepo  *repository.AuditRepository
	actor      string // Hardcoded actor for demo
}

// NewWorkloadEntryService creates a new WorkloadEntryService
func NewWorkloadEntryService(entryRepo *repository.EntryRepository, siteRepo *repository.SiteRepository,
	syncRepo *repository.SyncStatusRepository, auditRepo *repository.AuditRepository) *WorkloadEntryService {
	return &WorkloadEntryService{
		entryRepo: entryRepo,
		siteRepo:  siteRepo,
		syncRepo:  syncRepo,
		auditRepo: auditRepo,
		actor:     "demo-user",
	}
}

// CreateWorkloadEntry creates a new workload entry
func (s *WorkloadEntryService) CreateWorkloadEntry(ctx context.Context, spiffeID, parentID string,
	selectors []Selector, siteIDs []string, ttl int, description string) (*WorkloadEntryResponse, error) {

	// Convert selectors to repository format
	repoSelectors := make([]repository.Selector, len(selectors))
	for i, sel := range selectors {
		repoSelectors[i] = repository.Selector{Type: sel.Type, Value: sel.Value}
	}

	entry := &repository.WorkloadEntry{
		SpiffeID:    spiffeID,
		ParentID:    parentID,
		Selectors:   repoSelectors,
		TTL:         ttl,
		Description: description,
		CreatedBy:   s.actor,
	}

	created, err := s.entryRepo.Create(ctx, entry, siteIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to create workload entry: %w", err)
	}

	// Audit log
	details := map[string]interface{}{
		"spiffe_id": spiffeID,
		"parent_id": parentID,
		"site_ids":  siteIDs,
	}
	if err := s.auditRepo.Log(ctx, s.actor, "create", "workload_entry", created.ID, details); err != nil {
		log.Printf("Failed to write audit log: %v", err)
	}

	return toWorkloadEntryResponse(created), nil
}

// GetWorkloadEntry gets a workload entry by ID
func (s *WorkloadEntryService) GetWorkloadEntry(ctx context.Context, id string) (*WorkloadEntryResponse, error) {
	entry, err := s.entryRepo.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get workload entry: %w", err)
	}
	if entry == nil {
		return nil, fmt.Errorf("workload entry not found")
	}

	return toWorkloadEntryResponse(entry), nil
}

// ListWorkloadEntries lists workload entries with pagination
func (s *WorkloadEntryService) ListWorkloadEntries(ctx context.Context, pageSize int, pageToken string,
	siteID, spiffeIDPrefix string) (*ListWorkloadEntriesResponse, error) {

	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}

	offset := 0
	if pageToken != "" {
		fmt.Sscanf(pageToken, "%d", &offset)
	}

	entries, totalCount, err := s.entryRepo.List(ctx, pageSize, offset, siteID, spiffeIDPrefix)
	if err != nil {
		return nil, fmt.Errorf("failed to list workload entries: %w", err)
	}

	result := &ListWorkloadEntriesResponse{
		Entries:    make([]*WorkloadEntryResponse, len(entries)),
		TotalCount: totalCount,
	}

	for i, e := range entries {
		result.Entries[i] = toWorkloadEntryResponse(&e)
	}

	if offset+pageSize < totalCount {
		result.NextPageToken = fmt.Sprintf("%d", offset+pageSize)
	}

	return result, nil
}

// DeleteWorkloadEntry deletes a workload entry
func (s *WorkloadEntryService) DeleteWorkloadEntry(ctx context.Context, id string) error {
	// Get entry first for audit log
	entry, err := s.entryRepo.Get(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get workload entry: %w", err)
	}
	if entry == nil {
		return fmt.Errorf("workload entry not found")
	}

	if err := s.entryRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete workload entry: %w", err)
	}

	// Audit log
	details := map[string]interface{}{
		"spiffe_id": entry.SpiffeID,
	}
	if err := s.auditRepo.Log(ctx, s.actor, "delete", "workload_entry", id, details); err != nil {
		log.Printf("Failed to write audit log: %v", err)
	}

	return nil
}

// AssignToSites assigns an entry to additional sites
func (s *WorkloadEntryService) AssignToSites(ctx context.Context, entryID string, siteIDs []string) ([]SiteSyncStatus, error) {
	// Verify entry exists
	entry, err := s.entryRepo.Get(ctx, entryID)
	if err != nil {
		return nil, fmt.Errorf("failed to get workload entry: %w", err)
	}
	if entry == nil {
		return nil, fmt.Errorf("workload entry not found")
	}

	if err := s.entryRepo.AssignToSites(ctx, entryID, siteIDs); err != nil {
		return nil, fmt.Errorf("failed to assign to sites: %w", err)
	}

	// Audit log
	details := map[string]interface{}{
		"site_ids": siteIDs,
	}
	if err := s.auditRepo.Log(ctx, s.actor, "assign", "workload_entry", entryID, details); err != nil {
		log.Printf("Failed to write audit log: %v", err)
	}

	// Return updated sync statuses
	return s.GetSyncStatus(ctx, entryID)
}

// GetSyncStatus returns sync status for an entry
func (s *WorkloadEntryService) GetSyncStatus(ctx context.Context, entryID string) ([]SiteSyncStatus, error) {
	statuses, err := s.syncRepo.GetSyncStatuses(ctx, entryID)
	if err != nil {
		return nil, fmt.Errorf("failed to get sync status: %w", err)
	}

	result := make([]SiteSyncStatus, len(statuses))
	for i, st := range statuses {
		result[i] = SiteSyncStatus{
			SiteID:   st.SiteID,
			SiteName: st.SiteName,
			Status:   st.SyncStatus,
		}
		if st.SpireEntryID != nil {
			result[i].SpireEntryID = *st.SpireEntryID
		}
		if st.LastSyncAt != nil {
			result[i].LastSyncAt = timestamppb.New(*st.LastSyncAt)
		}
		if st.SyncError != nil {
			result[i].SyncError = *st.SyncError
		}
	}

	return result, nil
}

// Helper types for service layer

type Selector struct {
	Type  string
	Value string
}

type SiteSyncStatus struct {
	SiteID       string
	SiteName     string
	Status       string
	SpireEntryID string
	LastSyncAt   *timestamppb.Timestamp
	SyncError    string
}

type WorkloadEntryResponse struct {
	ID           string
	SpiffeID     string
	ParentID     string
	Selectors    []Selector
	TTL          int
	Description  string
	CreatedBy    string
	CreatedAt    *timestamppb.Timestamp
	UpdatedAt    *timestamppb.Timestamp
	SiteStatuses []SiteSyncStatus
}

type ListWorkloadEntriesResponse struct {
	Entries       []*WorkloadEntryResponse
	NextPageToken string
	TotalCount    int
}

func toWorkloadEntryResponse(entry *repository.WorkloadEntryWithSites) *WorkloadEntryResponse {
	selectors := make([]Selector, len(entry.Selectors))
	for i, s := range entry.Selectors {
		selectors[i] = Selector{Type: s.Type, Value: s.Value}
	}

	siteStatuses := make([]SiteSyncStatus, len(entry.SiteStatuses))
	for i, st := range entry.SiteStatuses {
		siteStatuses[i] = SiteSyncStatus{
			SiteID:   st.SiteID,
			SiteName: st.SiteName,
			Status:   st.SyncStatus,
		}
		if st.SpireEntryID != nil {
			siteStatuses[i].SpireEntryID = *st.SpireEntryID
		}
		if st.LastSyncAt != nil {
			siteStatuses[i].LastSyncAt = timestamppb.New(*st.LastSyncAt)
		}
		if st.SyncError != nil {
			siteStatuses[i].SyncError = *st.SyncError
		}
	}

	return &WorkloadEntryResponse{
		ID:           entry.ID,
		SpiffeID:     entry.SpiffeID,
		ParentID:     entry.ParentID,
		Selectors:    selectors,
		TTL:          entry.TTL,
		Description:  entry.Description,
		CreatedBy:    entry.CreatedBy,
		CreatedAt:    timestamppb.New(entry.CreatedAt),
		UpdatedAt:    timestamppb.New(entry.UpdatedAt),
		SiteStatuses: siteStatuses,
	}
}
