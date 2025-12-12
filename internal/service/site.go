package service

import (
	"context"
	"fmt"

	"github.com/yourorg/spire-workload-mgmt/internal/repository"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// SiteService handles site operations
type SiteService struct {
	siteRepo *repository.SiteRepository
}

// NewSiteService creates a new SiteService
func NewSiteService(siteRepo *repository.SiteRepository) *SiteService {
	return &SiteService{siteRepo: siteRepo}
}

// Site represents a site response
type Site struct {
	ID                 string
	Name               string
	Region             string
	SpireServerAddress string
	TrustDomain        string
	Status             string
	LastSyncAt         *timestamppb.Timestamp
}

// ListSites returns all sites, optionally filtered by status
func (s *SiteService) ListSites(ctx context.Context, status string) ([]Site, error) {
	sites, err := s.siteRepo.List(ctx, status)
	if err != nil {
		return nil, fmt.Errorf("failed to list sites: %w", err)
	}

	result := make([]Site, len(sites))
	for i, site := range sites {
		result[i] = Site{
			ID:                 site.ID,
			Name:               site.Name,
			Region:             site.Region,
			SpireServerAddress: site.SpireServerAddress,
			TrustDomain:        site.TrustDomain,
			Status:             site.Status,
		}
		if site.LastSyncAt != nil {
			result[i].LastSyncAt = timestamppb.New(*site.LastSyncAt)
		}
	}

	return result, nil
}

// GetSite returns a site by ID
func (s *SiteService) GetSite(ctx context.Context, id string) (*Site, error) {
	site, err := s.siteRepo.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get site: %w", err)
	}
	if site == nil {
		return nil, fmt.Errorf("site not found")
	}

	result := &Site{
		ID:                 site.ID,
		Name:               site.Name,
		Region:             site.Region,
		SpireServerAddress: site.SpireServerAddress,
		TrustDomain:        site.TrustDomain,
		Status:             site.Status,
	}
	if site.LastSyncAt != nil {
		result.LastSyncAt = timestamppb.New(*site.LastSyncAt)
	}

	return result, nil
}
