package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/yourorg/spire-workload-mgmt/internal/repository"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// AuditService handles audit log operations
type AuditService struct {
	auditRepo *repository.AuditRepository
}

// NewAuditService creates a new AuditService
func NewAuditService(auditRepo *repository.AuditRepository) *AuditService {
	return &AuditService{auditRepo: auditRepo}
}

// AuditLogEntry represents an audit log entry response
type AuditLogEntry struct {
	ID           int64
	Timestamp    *timestamppb.Timestamp
	Actor        string
	Action       string
	ResourceType string
	ResourceID   string
	Details      string // JSON string
}

// ListAuditLogsResponse represents the response for listing audit logs
type ListAuditLogsResponse struct {
	Entries       []AuditLogEntry
	NextPageToken string
}

// ListAuditLogs returns audit log entries with pagination and filters
func (s *AuditService) ListAuditLogs(ctx context.Context, pageSize int, pageToken string,
	resourceType, resourceID, actor string, startTime, endTime *time.Time) (*ListAuditLogsResponse, error) {

	if pageSize <= 0 || pageSize > 100 {
		pageSize = 50
	}

	offset := 0
	if pageToken != "" {
		fmt.Sscanf(pageToken, "%d", &offset)
	}

	entries, err := s.auditRepo.List(ctx, pageSize, offset, resourceType, resourceID, actor, startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("failed to list audit logs: %w", err)
	}

	result := &ListAuditLogsResponse{
		Entries: make([]AuditLogEntry, len(entries)),
	}

	for i, e := range entries {
		detailsJSON := ""
		if e.Details != nil {
			if data, err := json.Marshal(e.Details); err == nil {
				detailsJSON = string(data)
			}
		}

		result.Entries[i] = AuditLogEntry{
			ID:           e.ID,
			Timestamp:    timestamppb.New(e.Timestamp),
			Actor:        e.Actor,
			Action:       e.Action,
			ResourceType: e.ResourceType,
			ResourceID:   e.ResourceID,
			Details:      detailsJSON,
		}
	}

	// Set next page token if there might be more results
	if len(entries) == pageSize {
		result.NextPageToken = fmt.Sprintf("%d", offset+pageSize)
	}

	return result, nil
}
