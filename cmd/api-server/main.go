package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/yourorg/spire-workload-mgmt/internal/repository"
	"github.com/yourorg/spire-workload-mgmt/internal/service"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	log.Println("Starting SPIRE Workload Management API Server...")

	// Load configuration
	grpcPort := getEnv("GRPC_PORT", "8080")
	httpPort := getEnv("HTTP_PORT", "8081")

	// Connect to database
	dbConfig := repository.ConfigFromEnv()
	db, err := repository.NewDB(dbConfig)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()
	log.Println("Connected to MySQL database")

	// Initialize repositories
	siteRepo := repository.NewSiteRepository(db)
	entryRepo := repository.NewEntryRepository(db)
	syncRepo := repository.NewSyncStatusRepository(db)
	auditRepo := repository.NewAuditRepository(db)

	// Initialize services
	workloadEntrySvc := service.NewWorkloadEntryService(entryRepo, siteRepo, syncRepo, auditRepo)
	siteAgentSvc := service.NewSiteAgentService(syncRepo, siteRepo, auditRepo)
	siteSvc := service.NewSiteService(siteRepo)
	auditSvc := service.NewAuditService(auditRepo)

	// Create gRPC server
	grpcServer := grpc.NewServer()

	// Register services with gRPC
	RegisterServices(grpcServer, workloadEntrySvc, siteAgentSvc, siteSvc, auditSvc)
	reflection.Register(grpcServer)

	// Start gRPC server
	grpcLis, err := net.Listen("tcp", ":"+grpcPort)
	if err != nil {
		log.Fatalf("Failed to listen on gRPC port %s: %v", grpcPort, err)
	}

	go func() {
		log.Printf("gRPC server listening on port %s", grpcPort)
		if err := grpcServer.Serve(grpcLis); err != nil {
			log.Fatalf("gRPC server failed: %v", err)
		}
	}()

	// Start HTTP server for REST API (simpler browser access)
	httpServer := &http.Server{
		Addr:    ":" + httpPort,
		Handler: newHTTPHandler(workloadEntrySvc, siteAgentSvc, siteSvc, auditSvc, db),
	}

	go func() {
		log.Printf("HTTP server listening on port %s", httpPort)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server failed: %v", err)
		}
	}()

	// Wait for shutdown signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	log.Println("Shutting down...")
	grpcServer.GracefulStop()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	httpServer.Shutdown(ctx)

	log.Println("Server stopped")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// RegisterServices registers all gRPC services
// This is a placeholder - will be replaced with generated code
func RegisterServices(s *grpc.Server, workloadEntrySvc *service.WorkloadEntryService,
	siteAgentSvc *service.SiteAgentService, siteSvc *service.SiteService, auditSvc *service.AuditService) {
	// Services will be registered once proto code is generated
	log.Println("Services registered with gRPC server")
}

// HTTP Handler for REST API
func newHTTPHandler(workloadEntrySvc *service.WorkloadEntryService, siteAgentSvc *service.SiteAgentService,
	siteSvc *service.SiteService, auditSvc *service.AuditService, db *sql.DB) http.Handler {

	mux := http.NewServeMux()

	// CORS middleware
	cors := func(h http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}
			h(w, r)
		}
	}

	// Health check
	mux.HandleFunc("/health", cors(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
	}))

	// Sites endpoints
	mux.HandleFunc("/api/v1/sites", cors(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		w.Header().Set("Content-Type", "application/json")

		if r.Method == "GET" {
			sites, err := siteSvc.ListSites(ctx, "")
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			json.NewEncoder(w).Encode(map[string]interface{}{"sites": sites})
		}
	}))

	// Entries endpoints
	mux.HandleFunc("/api/v1/entries", cors(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		w.Header().Set("Content-Type", "application/json")

		switch r.Method {
		case "GET":
			entries, err := workloadEntrySvc.ListWorkloadEntries(ctx, 100, "", "", "")
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			json.NewEncoder(w).Encode(map[string]interface{}{
				"entries":     entries.Entries,
				"total_count": entries.TotalCount,
			})

		case "POST":
			var req struct {
				SpiffeID    string              `json:"spiffe_id"`
				ParentID    string              `json:"parent_id"`
				Selectors   []service.Selector  `json:"selectors"`
				SiteIDs     []string            `json:"site_ids"`
				TTL         int                 `json:"ttl"`
				Description string              `json:"description"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, "Invalid request body", http.StatusBadRequest)
				return
			}

			if req.TTL == 0 {
				req.TTL = 3600
			}

			entry, err := workloadEntrySvc.CreateWorkloadEntry(ctx, req.SpiffeID, req.ParentID,
				req.Selectors, req.SiteIDs, req.TTL, req.Description)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(entry)
		}
	}))

	// Single entry endpoint
	mux.HandleFunc("/api/v1/entries/", cors(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		w.Header().Set("Content-Type", "application/json")

		// Extract ID from path
		id := r.URL.Path[len("/api/v1/entries/"):]
		if id == "" {
			http.Error(w, "Entry ID required", http.StatusBadRequest)
			return
		}

		switch r.Method {
		case "GET":
			entry, err := workloadEntrySvc.GetWorkloadEntry(ctx, id)
			if err != nil {
				http.Error(w, err.Error(), http.StatusNotFound)
				return
			}
			json.NewEncoder(w).Encode(entry)

		case "DELETE":
			if err := workloadEntrySvc.DeleteWorkloadEntry(ctx, id); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			json.NewEncoder(w).Encode(map[string]bool{"success": true})
		}
	}))

	// Sync status endpoint
	mux.HandleFunc("/api/v1/entries/", cors(func(w http.ResponseWriter, r *http.Request) {
		// Already handled above
	}))

	// Site agent endpoints
	mux.HandleFunc("/api/v1/agent/poll", cors(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		w.Header().Set("Content-Type", "application/json")

		siteID := r.URL.Query().Get("site_id")
		if siteID == "" {
			http.Error(w, "site_id required", http.StatusBadRequest)
			return
		}

		entries, err := siteAgentSvc.PollEntries(ctx, siteID, 10)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(map[string]interface{}{"entries": entries})
	}))

	mux.HandleFunc("/api/v1/agent/report", cors(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		w.Header().Set("Content-Type", "application/json")

		var req struct {
			SiteID          string `json:"site_id"`
			WorkloadEntryID string `json:"workload_entry_id"`
			Success         bool   `json:"success"`
			SpireEntryID    string `json:"spire_entry_id"`
			ErrorMessage    string `json:"error_message"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		err := siteAgentSvc.ReportSyncResult(ctx, req.SiteID, req.WorkloadEntryID, req.Success, req.SpireEntryID, req.ErrorMessage)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(map[string]bool{"acknowledged": true})
	}))

	mux.HandleFunc("/api/v1/agent/poll-deletions", cors(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		w.Header().Set("Content-Type", "application/json")

		siteID := r.URL.Query().Get("site_id")
		if siteID == "" {
			http.Error(w, "site_id required", http.StatusBadRequest)
			return
		}

		entries, err := siteAgentSvc.PollDeletions(ctx, siteID, 10)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(map[string]interface{}{"entries": entries})
	}))

	mux.HandleFunc("/api/v1/agent/report-deletion", cors(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		w.Header().Set("Content-Type", "application/json")

		var req struct {
			SiteID          string `json:"site_id"`
			WorkloadEntryID string `json:"workload_entry_id"`
			Success         bool   `json:"success"`
			ErrorMessage    string `json:"error_message"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		err := siteAgentSvc.ReportDeletionResult(ctx, req.SiteID, req.WorkloadEntryID, req.Success, req.ErrorMessage)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(map[string]bool{"acknowledged": true})
	}))

	// Audit log endpoint
	mux.HandleFunc("/api/v1/audit", cors(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		w.Header().Set("Content-Type", "application/json")

		resourceType := r.URL.Query().Get("resource_type")
		resourceID := r.URL.Query().Get("resource_id")

		logs, err := auditSvc.ListAuditLogs(ctx, 100, "", resourceType, resourceID, "", nil, nil)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(map[string]interface{}{"entries": logs.Entries})
	}))

	return mux
}
