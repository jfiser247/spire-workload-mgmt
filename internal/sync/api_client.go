package sync

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// PendingEntry represents an entry pending sync
type PendingEntry struct {
	WorkloadEntryID string     `json:"workload_entry_id"`
	SpiffeID        string     `json:"spiffe_id"`
	ParentID        string     `json:"parent_id"`
	Selectors       []Selector `json:"selectors"`
	TTL             int        `json:"ttl"`
}

// Selector represents a workload selector
type Selector struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

// DeletionEntry represents an entry pending deletion
type DeletionEntry struct {
	WorkloadEntryID string `json:"workload_entry_id"`
	SpireEntryID    string `json:"spire_entry_id"`
}

// APIClient handles communication with the central API server
type APIClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewAPIClient creates a new API client
func NewAPIClient(address string) *APIClient {
	return &APIClient{
		baseURL: fmt.Sprintf("http://%s", address),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// PollEntries polls for entries pending sync
func (c *APIClient) PollEntries(ctx context.Context, siteID string, maxEntries int) ([]PendingEntry, error) {
	url := fmt.Sprintf("%s/api/v1/agent/poll?site_id=%s&max_entries=%d", c.baseURL, siteID, maxEntries)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to poll entries: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("poll entries failed with status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Entries []PendingEntry `json:"entries"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result.Entries, nil
}

// ReportSyncResult reports the result of syncing an entry
func (c *APIClient) ReportSyncResult(ctx context.Context, siteID, entryID string, success bool, spireEntryID, errorMsg string) error {
	url := fmt.Sprintf("%s/api/v1/agent/report", c.baseURL)

	payload := map[string]interface{}{
		"site_id":           siteID,
		"workload_entry_id": entryID,
		"success":           success,
		"spire_entry_id":    spireEntryID,
		"error_message":     errorMsg,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to report sync result: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("report sync result failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// PollDeletions polls for entries pending deletion
func (c *APIClient) PollDeletions(ctx context.Context, siteID string, maxEntries int) ([]DeletionEntry, error) {
	url := fmt.Sprintf("%s/api/v1/agent/poll-deletions?site_id=%s&max_entries=%d", c.baseURL, siteID, maxEntries)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to poll deletions: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("poll deletions failed with status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Entries []DeletionEntry `json:"entries"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result.Entries, nil
}

// ReportDeletionResult reports the result of deleting an entry
func (c *APIClient) ReportDeletionResult(ctx context.Context, siteID, entryID string, success bool, errorMsg string) error {
	url := fmt.Sprintf("%s/api/v1/agent/report-deletion", c.baseURL)

	payload := map[string]interface{}{
		"site_id":           siteID,
		"workload_entry_id": entryID,
		"success":           success,
		"error_message":     errorMsg,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to report deletion result: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("report deletion result failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}
