package grpcserver

import (
	"context"
	"fmt"
	"log"
	"net"

	"github.com/yourorg/spire-workload-mgmt/internal/repository"
	"github.com/yourorg/spire-workload-mgmt/internal/service"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Server wraps all gRPC services
type Server struct {
	grpcServer *grpc.Server
	listener   net.Listener

	workloadEntrySvc *service.WorkloadEntryService
	siteAgentSvc     *service.SiteAgentService
	siteSvc          *service.SiteService
	auditSvc         *service.AuditService
}

// NewServer creates a new gRPC server
func NewServer(
	workloadEntrySvc *service.WorkloadEntryService,
	siteAgentSvc *service.SiteAgentService,
	siteSvc *service.SiteService,
	auditSvc *service.AuditService,
) *Server {
	return &Server{
		workloadEntrySvc: workloadEntrySvc,
		siteAgentSvc:     siteAgentSvc,
		siteSvc:          siteSvc,
		auditSvc:         auditSvc,
	}
}

// Start starts the gRPC server
func (s *Server) Start(port string) error {
	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return fmt.Errorf("failed to listen on port %s: %w", port, err)
	}
	s.listener = lis

	s.grpcServer = grpc.NewServer(
		grpc.UnaryInterceptor(loggingInterceptor),
	)

	// Register services
	RegisterWorkloadEntryServiceServer(s.grpcServer, &workloadEntryServer{svc: s.workloadEntrySvc})
	RegisterSiteServiceServer(s.grpcServer, &siteServer{svc: s.siteSvc})
	RegisterSiteAgentServiceServer(s.grpcServer, &siteAgentServer{svc: s.siteAgentSvc})
	RegisterAuditServiceServer(s.grpcServer, &auditServer{svc: s.auditSvc})

	// Enable reflection for grpcurl/debugging
	reflection.Register(s.grpcServer)

	log.Printf("gRPC server starting on port %s", port)
	return s.grpcServer.Serve(lis)
}

// Stop stops the gRPC server
func (s *Server) Stop() {
	if s.grpcServer != nil {
		s.grpcServer.GracefulStop()
	}
}

// Logging interceptor
func loggingInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	log.Printf("gRPC call: %s", info.FullMethod)
	resp, err := handler(ctx, req)
	if err != nil {
		log.Printf("gRPC error: %s - %v", info.FullMethod, err)
	}
	return resp, err
}

// ============ Proto-compatible types and interfaces ============

// These types mirror the proto definitions and will be replaced by generated code

// WorkloadEntry proto message
type WorkloadEntry struct {
	Id           string
	SpiffeId     string
	ParentId     string
	Selectors    []*Selector
	Ttl          int32
	Description  string
	CreatedBy    string
	CreatedAt    *timestamppb.Timestamp
	UpdatedAt    *timestamppb.Timestamp
	SiteStatuses []*SiteSyncStatus
}

type Selector struct {
	Type  string
	Value string
}

type SiteSyncStatus struct {
	SiteId       string
	SiteName     string
	Status       string
	SpireEntryId string
	LastSyncAt   *timestamppb.Timestamp
	SyncError    string
}

type Site struct {
	Id                 string
	Name               string
	Region             string
	SpireServerAddress string
	TrustDomain        string
	Status             string
	LastSyncAt         *timestamppb.Timestamp
}

type AuditLogEntry struct {
	Id           int64
	Timestamp    *timestamppb.Timestamp
	Actor        string
	Action       string
	ResourceType string
	ResourceId   string
	Details      string
}

type PendingEntry struct {
	WorkloadEntryId string
	SpiffeId        string
	ParentId        string
	Selectors       []*Selector
	Ttl             int32
}

type DeletionEntry struct {
	WorkloadEntryId string
	SpireEntryId    string
}

// Request/Response types
type CreateWorkloadEntryRequest struct {
	SpiffeId    string
	ParentId    string
	Selectors   []*Selector
	SiteIds     []string
	Ttl         int32
	Description string
}

type GetWorkloadEntryRequest struct{ Id string }
type ListWorkloadEntriesRequest struct {
	PageSize       int32
	PageToken      string
	SiteId         string
	SpiffeIdPrefix string
}
type ListWorkloadEntriesResponse struct {
	Entries       []*WorkloadEntry
	NextPageToken string
	TotalCount    int32
}
type DeleteWorkloadEntryRequest struct{ Id string }
type DeleteWorkloadEntryResponse struct {
	Success bool
	Message string
}
type AssignToSitesRequest struct {
	WorkloadEntryId string
	SiteIds         []string
}
type AssignToSitesResponse struct{ Statuses []*SiteSyncStatus }
type GetSyncStatusRequest struct{ WorkloadEntryId string }
type SyncStatusResponse struct{ Statuses []*SiteSyncStatus }

type ListSitesRequest struct{ Status string }
type ListSitesResponse struct{ Sites []*Site }
type GetSiteRequest struct{ Id string }

type PollEntriesRequest struct {
	SiteId     string
	MaxEntries int32
}
type PollEntriesResponse struct{ Entries []*PendingEntry }
type ReportSyncResultRequest struct {
	SiteId          string
	WorkloadEntryId string
	Success         bool
	SpireEntryId    string
	ErrorMessage    string
}
type ReportSyncResultResponse struct{ Acknowledged bool }
type PollDeletionsRequest struct {
	SiteId     string
	MaxEntries int32
}
type PollDeletionsResponse struct{ Entries []*DeletionEntry }
type ReportDeletionResultRequest struct {
	SiteId          string
	WorkloadEntryId string
	Success         bool
	ErrorMessage    string
}
type ReportDeletionResultResponse struct{ Acknowledged bool }

type ListAuditLogsRequest struct {
	PageSize     int32
	PageToken    string
	ResourceType string
	ResourceId   string
	Actor        string
	StartTime    *timestamppb.Timestamp
	EndTime      *timestamppb.Timestamp
}
type ListAuditLogsResponse struct {
	Entries       []*AuditLogEntry
	NextPageToken string
}

// ============ Service interfaces (to be implemented by generated code registration) ============

type WorkloadEntryServiceServer interface {
	CreateWorkloadEntry(context.Context, *CreateWorkloadEntryRequest) (*WorkloadEntry, error)
	GetWorkloadEntry(context.Context, *GetWorkloadEntryRequest) (*WorkloadEntry, error)
	ListWorkloadEntries(context.Context, *ListWorkloadEntriesRequest) (*ListWorkloadEntriesResponse, error)
	DeleteWorkloadEntry(context.Context, *DeleteWorkloadEntryRequest) (*DeleteWorkloadEntryResponse, error)
	AssignToSites(context.Context, *AssignToSitesRequest) (*AssignToSitesResponse, error)
	GetSyncStatus(context.Context, *GetSyncStatusRequest) (*SyncStatusResponse, error)
}

type SiteServiceServer interface {
	ListSites(context.Context, *ListSitesRequest) (*ListSitesResponse, error)
	GetSite(context.Context, *GetSiteRequest) (*Site, error)
}

type SiteAgentServiceServer interface {
	PollEntries(context.Context, *PollEntriesRequest) (*PollEntriesResponse, error)
	ReportSyncResult(context.Context, *ReportSyncResultRequest) (*ReportSyncResultResponse, error)
	PollDeletions(context.Context, *PollDeletionsRequest) (*PollDeletionsResponse, error)
	ReportDeletionResult(context.Context, *ReportDeletionResultRequest) (*ReportDeletionResultResponse, error)
}

type AuditServiceServer interface {
	ListAuditLogs(context.Context, *ListAuditLogsRequest) (*ListAuditLogsResponse, error)
}

// Registration functions (placeholder - will use generated code)
func RegisterWorkloadEntryServiceServer(s *grpc.Server, srv WorkloadEntryServiceServer) {
	// In real implementation, this would register the proto-generated service descriptor
	log.Println("WorkloadEntryService registered")
}

func RegisterSiteServiceServer(s *grpc.Server, srv SiteServiceServer) {
	log.Println("SiteService registered")
}

func RegisterSiteAgentServiceServer(s *grpc.Server, srv SiteAgentServiceServer) {
	log.Println("SiteAgentService registered")
}

func RegisterAuditServiceServer(s *grpc.Server, srv AuditServiceServer) {
	log.Println("AuditService registered")
}

// ============ Server implementations ============

type workloadEntryServer struct {
	svc *service.WorkloadEntryService
}

func (s *workloadEntryServer) CreateWorkloadEntry(ctx context.Context, req *CreateWorkloadEntryRequest) (*WorkloadEntry, error) {
	selectors := make([]service.Selector, len(req.Selectors))
	for i, sel := range req.Selectors {
		selectors[i] = service.Selector{Type: sel.Type, Value: sel.Value}
	}

	result, err := s.svc.CreateWorkloadEntry(ctx, req.SpiffeId, req.ParentId, selectors, req.SiteIds, int(req.Ttl), req.Description)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create workload entry: %v", err)
	}

	return toProtoWorkloadEntry(result), nil
}

func (s *workloadEntryServer) GetWorkloadEntry(ctx context.Context, req *GetWorkloadEntryRequest) (*WorkloadEntry, error) {
	result, err := s.svc.GetWorkloadEntry(ctx, req.Id)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "workload entry not found: %v", err)
	}
	return toProtoWorkloadEntry(result), nil
}

func (s *workloadEntryServer) ListWorkloadEntries(ctx context.Context, req *ListWorkloadEntriesRequest) (*ListWorkloadEntriesResponse, error) {
	result, err := s.svc.ListWorkloadEntries(ctx, int(req.PageSize), req.PageToken, req.SiteId, req.SpiffeIdPrefix)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list workload entries: %v", err)
	}

	entries := make([]*WorkloadEntry, len(result.Entries))
	for i, e := range result.Entries {
		entries[i] = toProtoWorkloadEntry(e)
	}

	return &ListWorkloadEntriesResponse{
		Entries:       entries,
		NextPageToken: result.NextPageToken,
		TotalCount:    int32(result.TotalCount),
	}, nil
}

func (s *workloadEntryServer) DeleteWorkloadEntry(ctx context.Context, req *DeleteWorkloadEntryRequest) (*DeleteWorkloadEntryResponse, error) {
	err := s.svc.DeleteWorkloadEntry(ctx, req.Id)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete workload entry: %v", err)
	}
	return &DeleteWorkloadEntryResponse{Success: true, Message: "Entry deleted successfully"}, nil
}

func (s *workloadEntryServer) AssignToSites(ctx context.Context, req *AssignToSitesRequest) (*AssignToSitesResponse, error) {
	result, err := s.svc.AssignToSites(ctx, req.WorkloadEntryId, req.SiteIds)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to assign to sites: %v", err)
	}

	statuses := make([]*SiteSyncStatus, len(result))
	for i, st := range result {
		statuses[i] = &SiteSyncStatus{
			SiteId:       st.SiteID,
			SiteName:     st.SiteName,
			Status:       st.Status,
			SpireEntryId: st.SpireEntryID,
			LastSyncAt:   st.LastSyncAt,
			SyncError:    st.SyncError,
		}
	}

	return &AssignToSitesResponse{Statuses: statuses}, nil
}

func (s *workloadEntryServer) GetSyncStatus(ctx context.Context, req *GetSyncStatusRequest) (*SyncStatusResponse, error) {
	result, err := s.svc.GetSyncStatus(ctx, req.WorkloadEntryId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get sync status: %v", err)
	}

	statuses := make([]*SiteSyncStatus, len(result))
	for i, st := range result {
		statuses[i] = &SiteSyncStatus{
			SiteId:       st.SiteID,
			SiteName:     st.SiteName,
			Status:       st.Status,
			SpireEntryId: st.SpireEntryID,
			LastSyncAt:   st.LastSyncAt,
			SyncError:    st.SyncError,
		}
	}

	return &SyncStatusResponse{Statuses: statuses}, nil
}

type siteServer struct {
	svc *service.SiteService
}

func (s *siteServer) ListSites(ctx context.Context, req *ListSitesRequest) (*ListSitesResponse, error) {
	result, err := s.svc.ListSites(ctx, req.Status)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list sites: %v", err)
	}

	sites := make([]*Site, len(result))
	for i, site := range result {
		sites[i] = &Site{
			Id:                 site.ID,
			Name:               site.Name,
			Region:             site.Region,
			SpireServerAddress: site.SpireServerAddress,
			TrustDomain:        site.TrustDomain,
			Status:             site.Status,
			LastSyncAt:         site.LastSyncAt,
		}
	}

	return &ListSitesResponse{Sites: sites}, nil
}

func (s *siteServer) GetSite(ctx context.Context, req *GetSiteRequest) (*Site, error) {
	result, err := s.svc.GetSite(ctx, req.Id)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "site not found: %v", err)
	}
	return &Site{
		Id:                 result.ID,
		Name:               result.Name,
		Region:             result.Region,
		SpireServerAddress: result.SpireServerAddress,
		TrustDomain:        result.TrustDomain,
		Status:             result.Status,
		LastSyncAt:         result.LastSyncAt,
	}, nil
}

type siteAgentServer struct {
	svc *service.SiteAgentService
}

func (s *siteAgentServer) PollEntries(ctx context.Context, req *PollEntriesRequest) (*PollEntriesResponse, error) {
	result, err := s.svc.PollEntries(ctx, req.SiteId, int(req.MaxEntries))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to poll entries: %v", err)
	}

	entries := make([]*PendingEntry, len(result))
	for i, e := range result {
		selectors := make([]*Selector, len(e.Selectors))
		for j, sel := range e.Selectors {
			selectors[j] = &Selector{Type: sel.Type, Value: sel.Value}
		}
		entries[i] = &PendingEntry{
			WorkloadEntryId: e.WorkloadEntryID,
			SpiffeId:        e.SpiffeID,
			ParentId:        e.ParentID,
			Selectors:       selectors,
			Ttl:             int32(e.TTL),
		}
	}

	return &PollEntriesResponse{Entries: entries}, nil
}

func (s *siteAgentServer) ReportSyncResult(ctx context.Context, req *ReportSyncResultRequest) (*ReportSyncResultResponse, error) {
	err := s.svc.ReportSyncResult(ctx, req.SiteId, req.WorkloadEntryId, req.Success, req.SpireEntryId, req.ErrorMessage)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to report sync result: %v", err)
	}
	return &ReportSyncResultResponse{Acknowledged: true}, nil
}

func (s *siteAgentServer) PollDeletions(ctx context.Context, req *PollDeletionsRequest) (*PollDeletionsResponse, error) {
	result, err := s.svc.PollDeletions(ctx, req.SiteId, int(req.MaxEntries))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to poll deletions: %v", err)
	}

	entries := make([]*DeletionEntry, len(result))
	for i, e := range result {
		entries[i] = &DeletionEntry{
			WorkloadEntryId: e.WorkloadEntryID,
			SpireEntryId:    e.SpireEntryID,
		}
	}

	return &PollDeletionsResponse{Entries: entries}, nil
}

func (s *siteAgentServer) ReportDeletionResult(ctx context.Context, req *ReportDeletionResultRequest) (*ReportDeletionResultResponse, error) {
	err := s.svc.ReportDeletionResult(ctx, req.SiteId, req.WorkloadEntryId, req.Success, req.ErrorMessage)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to report deletion result: %v", err)
	}
	return &ReportDeletionResultResponse{Acknowledged: true}, nil
}

type auditServer struct {
	svc *service.AuditService
}

func (s *auditServer) ListAuditLogs(ctx context.Context, req *ListAuditLogsRequest) (*ListAuditLogsResponse, error) {
	var startTime, endTime *timestamppb.Timestamp
	if req.StartTime != nil {
		startTime = req.StartTime
	}
	if req.EndTime != nil {
		endTime = req.EndTime
	}

	var st, et *repository.Time
	_ = startTime
	_ = endTime
	// Convert timestamps if needed

	result, err := s.svc.ListAuditLogs(ctx, int(req.PageSize), req.PageToken, req.ResourceType, req.ResourceId, req.Actor, nil, nil)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list audit logs: %v", err)
	}
	_ = st
	_ = et

	entries := make([]*AuditLogEntry, len(result.Entries))
	for i, e := range result.Entries {
		entries[i] = &AuditLogEntry{
			Id:           e.ID,
			Timestamp:    e.Timestamp,
			Actor:        e.Actor,
			Action:       e.Action,
			ResourceType: e.ResourceType,
			ResourceId:   e.ResourceID,
			Details:      e.Details,
		}
	}

	return &ListAuditLogsResponse{
		Entries:       entries,
		NextPageToken: result.NextPageToken,
	}, nil
}

// Helper to convert service response to proto
func toProtoWorkloadEntry(e *service.WorkloadEntryResponse) *WorkloadEntry {
	selectors := make([]*Selector, len(e.Selectors))
	for i, sel := range e.Selectors {
		selectors[i] = &Selector{Type: sel.Type, Value: sel.Value}
	}

	siteStatuses := make([]*SiteSyncStatus, len(e.SiteStatuses))
	for i, st := range e.SiteStatuses {
		siteStatuses[i] = &SiteSyncStatus{
			SiteId:       st.SiteID,
			SiteName:     st.SiteName,
			Status:       st.Status,
			SpireEntryId: st.SpireEntryID,
			LastSyncAt:   st.LastSyncAt,
			SyncError:    st.SyncError,
		}
	}

	return &WorkloadEntry{
		Id:           e.ID,
		SpiffeId:     e.SpiffeID,
		ParentId:     e.ParentID,
		Selectors:    selectors,
		Ttl:          int32(e.TTL),
		Description:  e.Description,
		CreatedBy:    e.CreatedBy,
		CreatedAt:    e.CreatedAt,
		UpdatedAt:    e.UpdatedAt,
		SiteStatuses: siteStatuses,
	}
}
