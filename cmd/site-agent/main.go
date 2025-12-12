package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/yourorg/spire-workload-mgmt/internal/sync"
)

func main() {
	log.Println("Starting SPIRE Workload Management Site Agent...")

	// Load configuration from environment
	config := sync.Config{
		SiteID:              getEnv("SITE_ID", "site-a"),
		SiteName:            getEnv("SITE_NAME", "Default Site"),
		APIServerAddress:    getEnv("API_SERVER_ADDRESS", "localhost:8081"),
		SpireSocketPath:     getEnv("SPIRE_SOCKET_PATH", "/run/spire/agent-sockets/spire-agent.sock"),
		SyncIntervalSeconds: getEnvInt("SYNC_INTERVAL_SECONDS", 10),
		MaxEntries:          getEnvInt("MAX_ENTRIES", 10),
	}

	// Validate required config
	if config.SiteID == "" {
		log.Fatal("SITE_ID environment variable is required")
	}

	// Create agent
	agent, err := sync.NewAgent(config)
	if err != nil {
		log.Fatalf("Failed to create agent: %v", err)
	}
	defer agent.Close()

	// Create context that cancels on shutdown signal
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		log.Println("Received shutdown signal")
		cancel()
	}()

	// Run agent
	if err := agent.Run(ctx); err != nil && err != context.Canceled {
		log.Fatalf("Agent error: %v", err)
	}

	log.Println("Site agent stopped")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}
