package sync

import (
	"context"
	"log"
	"time"
)

// Config holds the site agent configuration
type Config struct {
	SiteID              string
	SiteName            string
	APIServerAddress    string
	SpireSocketPath     string
	SyncIntervalSeconds int
	MaxEntries          int
}

// Agent handles syncing workload entries to the local SPIRE server
type Agent struct {
	config      Config
	apiClient   *APIClient
	spireClient *SpireClient
}

// NewAgent creates a new sync agent
func NewAgent(config Config) (*Agent, error) {
	apiClient := NewAPIClient(config.APIServerAddress)

	spireClient, err := NewSpireClient(config.SpireSocketPath)
	if err != nil {
		return nil, err
	}

	return &Agent{
		config:      config,
		apiClient:   apiClient,
		spireClient: spireClient,
	}, nil
}

// Run starts the sync loop
func (a *Agent) Run(ctx context.Context) error {
	log.Printf("Starting sync agent for site %s (%s)", a.config.SiteID, a.config.SiteName)
	log.Printf("API server: %s", a.config.APIServerAddress)
	log.Printf("SPIRE socket: %s", a.config.SpireSocketPath)
	log.Printf("Sync interval: %d seconds", a.config.SyncIntervalSeconds)

	ticker := time.NewTicker(time.Duration(a.config.SyncIntervalSeconds) * time.Second)
	defer ticker.Stop()

	// Run immediately on start
	a.syncCycle(ctx)

	for {
		select {
		case <-ctx.Done():
			log.Println("Sync agent shutting down...")
			return ctx.Err()
		case <-ticker.C:
			a.syncCycle(ctx)
		}
	}
}

// syncCycle runs one sync cycle
func (a *Agent) syncCycle(ctx context.Context) {
	log.Printf("[%s] Starting sync cycle...", a.config.SiteID)

	// 1. Poll for pending entries
	a.syncPendingEntries(ctx)

	// 2. Poll for deletions
	a.syncDeletions(ctx)

	log.Printf("[%s] Sync cycle complete", a.config.SiteID)
}

// syncPendingEntries syncs pending entries to SPIRE
func (a *Agent) syncPendingEntries(ctx context.Context) {
	entries, err := a.apiClient.PollEntries(ctx, a.config.SiteID, a.config.MaxEntries)
	if err != nil {
		log.Printf("[%s] Error polling entries: %v", a.config.SiteID, err)
		return
	}

	if len(entries) == 0 {
		log.Printf("[%s] No pending entries to sync", a.config.SiteID)
		return
	}

	log.Printf("[%s] Found %d pending entries to sync", a.config.SiteID, len(entries))

	for _, entry := range entries {
		a.syncEntry(ctx, entry)
	}
}

// syncEntry syncs a single entry to SPIRE
func (a *Agent) syncEntry(ctx context.Context, entry PendingEntry) {
	log.Printf("[%s] Syncing entry %s (SPIFFE ID: %s)", a.config.SiteID, entry.WorkloadEntryID, entry.SpiffeID)

	// Create the entry in SPIRE
	spireEntryID, err := a.spireClient.CreateEntry(ctx, entry)
	if err != nil {
		log.Printf("[%s] Error creating SPIRE entry for %s: %v", a.config.SiteID, entry.WorkloadEntryID, err)

		// Report failure
		if reportErr := a.apiClient.ReportSyncResult(ctx, a.config.SiteID, entry.WorkloadEntryID, false, "", err.Error()); reportErr != nil {
			log.Printf("[%s] Error reporting sync failure: %v", a.config.SiteID, reportErr)
		}
		return
	}

	log.Printf("[%s] Created SPIRE entry %s for %s", a.config.SiteID, spireEntryID, entry.WorkloadEntryID)

	// Report success
	if err := a.apiClient.ReportSyncResult(ctx, a.config.SiteID, entry.WorkloadEntryID, true, spireEntryID, ""); err != nil {
		log.Printf("[%s] Error reporting sync success: %v", a.config.SiteID, err)
	}
}

// syncDeletions handles pending deletions
func (a *Agent) syncDeletions(ctx context.Context) {
	entries, err := a.apiClient.PollDeletions(ctx, a.config.SiteID, a.config.MaxEntries)
	if err != nil {
		log.Printf("[%s] Error polling deletions: %v", a.config.SiteID, err)
		return
	}

	if len(entries) == 0 {
		return
	}

	log.Printf("[%s] Found %d entries to delete", a.config.SiteID, len(entries))

	for _, entry := range entries {
		a.deleteEntry(ctx, entry)
	}
}

// deleteEntry deletes an entry from SPIRE
func (a *Agent) deleteEntry(ctx context.Context, entry DeletionEntry) {
	log.Printf("[%s] Deleting SPIRE entry %s", a.config.SiteID, entry.SpireEntryID)

	err := a.spireClient.DeleteEntry(ctx, entry.SpireEntryID)
	if err != nil {
		log.Printf("[%s] Error deleting SPIRE entry %s: %v", a.config.SiteID, entry.SpireEntryID, err)

		// Report failure
		if reportErr := a.apiClient.ReportDeletionResult(ctx, a.config.SiteID, entry.WorkloadEntryID, false, err.Error()); reportErr != nil {
			log.Printf("[%s] Error reporting deletion failure: %v", a.config.SiteID, reportErr)
		}
		return
	}

	log.Printf("[%s] Deleted SPIRE entry %s", a.config.SiteID, entry.SpireEntryID)

	// Report success
	if err := a.apiClient.ReportDeletionResult(ctx, a.config.SiteID, entry.WorkloadEntryID, true, ""); err != nil {
		log.Printf("[%s] Error reporting deletion success: %v", a.config.SiteID, err)
	}
}

// Close cleans up resources
func (a *Agent) Close() error {
	if a.spireClient != nil {
		return a.spireClient.Close()
	}
	return nil
}
