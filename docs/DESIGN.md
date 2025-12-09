# SPIFFE/SPIRE Workload Entry Management System

## Technical Design Document

**Version:** 1.0  
**Date:** December 2025  
**Status:** Draft  
**Classification:** Internal  
**Target Platform:** Kubernetes (Multi-cluster)

---

## Table of Contents

- [1. Executive Summary](#1-executive-summary)
- [2. Problem Statement](#2-problem-statement)
- [3. System Architecture](#3-system-architecture)
- [4. Data Model](#4-data-model)
- [5. gRPC API Design](#5-grpc-api-design)
- [6. Synchronization Design](#6-synchronization-design)
- [7. Kubernetes Deployment](#7-kubernetes-deployment)
- [8. Security Design](#8-security-design)
- [9. Observability](#9-observability)
- [10. Backstage Integration](#10-backstage-integration)
- [11. Implementation Phases](#11-implementation-phases)
- [12. Appendix](#12-appendix)

---

## 1. Executive Summary

This document describes the technical design for a centralized SPIFFE/SPIRE Workload Entry Management System. The system provides a unified gRPC API for managing SPIFFE workload entries across a globally distributed multi-site Kubernetes infrastructure. It is designed as a Backstage application to integrate with existing developer portal infrastructure.

The platform addresses the operational complexity of managing workload identities at scale by providing centralized control, multi-site synchronization, and comprehensive audit capabilities while maintaining the security guarantees provided by SPIFFE/SPIRE.

---

## 2. Problem Statement

### 2.1 Current Challenges

Organizations operating SPIRE across multiple geographic regions face several operational challenges:

1. **Fragmented Management:** Each SPIRE server requires individual management, leading to configuration drift and inconsistencies across sites.

2. **Lack of Visibility:** No centralized view of workload entries across the entire infrastructure.

3. **Manual Synchronization:** Propagating workload entries to multiple sites requires manual intervention or custom scripting.

4. **Audit Gaps:** Difficulty tracking who created or modified workload entries and when.

5. **Developer Experience:** Developers lack self-service capabilities for requesting workload identities.

### 2.2 Goals

- Provide a single API for managing workload entries across all SPIRE deployments
- Enable automated synchronization of entries to designated sites
- Maintain complete audit trail of all changes
- Integrate with Backstage for developer self-service
- Support high availability and disaster recovery

---

## 3. System Architecture

### 3.1 High-Level Architecture

The system follows a hub-and-spoke architecture where the central management service acts as the source of truth, and site agents synchronize entries to local SPIRE servers.

```
┌─────────────────────────────────────────────────────────────────┐
│                    BACKSTAGE PORTAL                             │
│              (Developer Self-Service UI)                        │
└────────────────────────────┬────────────────────────────────────┘
                             │ gRPC
┌────────────────────────────▼────────────────────────────────────┐
│              WORKLOAD ENTRY MANAGEMENT SERVICE                  │
│    ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐      │
│    │ gRPC API │  │   Sync   │  │  Audit   │  │  Health  │      │
│    │  Server  │  │  Engine  │  │  Logger  │  │  Monitor │      │
│    └──────────┘  └──────────┘  └──────────┘  └──────────┘      │
└────────────────────────────┬────────────────────────────────────┘
                             │
                    ┌────────┴────────┐
                    │   MySQL Cluster  │
                    │  (Primary Store) │
                    └─────────────────┘
                             │
          ┌──────────────────┼──────────────────┐
          ▼                  ▼                  ▼
   ┌────────────┐     ┌────────────┐     ┌────────────┐
   │ Site Agent │     │ Site Agent │     │ Site Agent │
   │  (US-EAST) │     │  (EU-WEST) │     │ (APAC-SGP) │
   └─────┬──────┘     └─────┬──────┘     └─────┬──────┘
         ▼                  ▼                  ▼
   ┌────────────┐     ┌────────────┐     ┌────────────┐
   │   SPIRE    │     │   SPIRE    │     │   SPIRE    │
   │   Server   │     │   Server   │     │   Server   │
   └────────────┘     └────────────┘     └────────────┘
```

### 3.2 Core Components

#### 3.2.1 Workload Entry Management Service

The central service responsible for all workload entry operations. Key responsibilities:

- Exposing gRPC API for CRUD operations on workload entries
- Managing site configurations and entry-to-site mappings
- Coordinating synchronization across sites
- Recording audit events for all mutations

#### 3.2.2 Site Agent

Deployed at each SPIRE site to handle local synchronization:

- Polling central service for entry updates
- Translating entries to SPIRE registration API calls
- Reporting synchronization status back to central service
- Handling local SPIRE server connectivity

#### 3.2.3 MySQL Database

Persistent storage for all system data:

- Workload entry definitions and selectors
- Site configurations and entry-to-site assignments
- Synchronization state, history, and audit logs

---

## 4. Data Model

### 4.1 Core Tables

#### 4.1.1 sites

| Column | Type | Description |
|--------|------|-------------|
| `id` | UUID | Primary key |
| `name` | VARCHAR(255) | Human-readable site name (e.g., us-east-1) |
| `region` | VARCHAR(100) | Geographic region identifier |
| `trust_domain` | VARCHAR(255) | SPIFFE trust domain for this site |
| `spire_server_addr` | VARCHAR(255) | SPIRE server endpoint address |
| `status` | ENUM | ACTIVE, INACTIVE, MAINTENANCE |
| `created_at` | TIMESTAMP | Record creation timestamp |
| `updated_at` | TIMESTAMP | Last modification timestamp |

#### 4.1.2 workload_entries

| Column | Type | Description |
|--------|------|-------------|
| `id` | UUID | Primary key |
| `spiffe_id` | VARCHAR(2048) | Full SPIFFE ID (spiffe://trust-domain/path) |
| `parent_id` | VARCHAR(2048) | Parent SPIFFE ID for attestation hierarchy |
| `ttl` | INT | SVID TTL in seconds |
| `dns_names` | JSON | Array of DNS names for X.509 SVID |
| `admin` | BOOLEAN | Admin flag for SPIRE server access |
| `downstream` | BOOLEAN | Downstream entry flag |
| `revision` | BIGINT | Optimistic locking revision number |
| `created_by` | VARCHAR(255) | Creator identity (user/service) |
| `created_at` | TIMESTAMP | Record creation timestamp |
| `updated_at` | TIMESTAMP | Last modification timestamp |

#### 4.1.3 workload_entry_selectors

| Column | Type | Description |
|--------|------|-------------|
| `id` | UUID | Primary key |
| `workload_entry_id` | UUID | Foreign key to workload_entries |
| `type` | VARCHAR(255) | Selector type (e.g., k8s:ns, k8s:sa) |
| `value` | VARCHAR(4096) | Selector value |

#### 4.1.4 workload_entry_sites

| Column | Type | Description |
|--------|------|-------------|
| `workload_entry_id` | UUID | Foreign key to workload_entries |
| `site_id` | UUID | Foreign key to sites |
| `sync_status` | ENUM | PENDING, SYNCED, FAILED, DELETING |
| `spire_entry_id` | VARCHAR(255) | Entry ID assigned by SPIRE server |
| `last_sync_at` | TIMESTAMP | Last successful synchronization time |
| `last_error` | TEXT | Last synchronization error message |

---

## 5. gRPC API Design

### 5.1 Service Definitions

#### 5.1.1 WorkloadEntryService

```protobuf
service WorkloadEntryService {
  rpc CreateEntry(CreateEntryRequest) returns (Entry);
  rpc GetEntry(GetEntryRequest) returns (Entry);
  rpc UpdateEntry(UpdateEntryRequest) returns (Entry);
  rpc DeleteEntry(DeleteEntryRequest) returns (DeleteEntryResponse);
  rpc ListEntries(ListEntriesRequest) returns (ListEntriesResponse);
  rpc BatchCreateEntries(BatchCreateEntriesRequest) returns (BatchCreateEntriesResponse);
  rpc BatchDeleteEntries(BatchDeleteEntriesRequest) returns (BatchDeleteEntriesResponse);
}
```

#### 5.1.2 SiteService

```protobuf
service SiteService {
  rpc CreateSite(CreateSiteRequest) returns (Site);
  rpc GetSite(GetSiteRequest) returns (Site);
  rpc UpdateSite(UpdateSiteRequest) returns (Site);
  rpc DeleteSite(DeleteSiteRequest) returns (DeleteSiteResponse);
  rpc ListSites(ListSitesRequest) returns (ListSitesResponse);
  rpc GetSiteHealth(GetSiteHealthRequest) returns (SiteHealth);
}
```

#### 5.1.3 SyncService

```protobuf
service SyncService {
  rpc TriggerSync(TriggerSyncRequest) returns (TriggerSyncResponse);
  rpc GetSyncStatus(GetSyncStatusRequest) returns (SyncStatus);
  rpc StreamSyncUpdates(StreamSyncUpdatesRequest) returns (stream SyncUpdate);
  // Site Agent endpoints
  rpc PollEntries(PollEntriesRequest) returns (PollEntriesResponse);
  rpc ReportSyncResult(ReportSyncResultRequest) returns (ReportSyncResultResponse);
}
```

### 5.2 Entry Message

```protobuf
message Entry {
  string id = 1;
  string spiffe_id = 2;
  string parent_id = 3;
  repeated Selector selectors = 4;
  int32 ttl = 5;
  repeated string dns_names = 6;
  bool admin = 7;
  bool downstream = 8;
  google.protobuf.Timestamp expiry_time = 9;
  repeated string site_ids = 10;
  int64 revision = 11;
  google.protobuf.Timestamp created_at = 12;
  google.protobuf.Timestamp updated_at = 13;
}

message Selector {
  string type = 1;   // e.g., "k8s:ns", "k8s:sa", "k8s:pod-label"
  string value = 2;  // e.g., "production", "api-server"
}
```

### 5.3 API Operations Summary

| Operation | Service | Description |
|-----------|---------|-------------|
| CreateEntry | Entry | Create new workload entry with site assignments |
| UpdateEntry | Entry | Update entry with optimistic locking via revision |
| ListEntries | Entry | Paginated listing with filters by site, selector, SPIFFE ID |
| TriggerSync | Sync | Force immediate synchronization to specified sites |
| StreamSyncUpdates | Sync | Server streaming for real-time sync status updates |
| PollEntries | Sync | Site agent polls for entries needing sync (long-poll) |

---

## 6. Synchronization Design

### 6.1 Synchronization Flow

The synchronization system ensures entries are propagated to designated SPIRE servers reliably with eventual consistency.

#### 6.1.1 Entry Creation/Update Flow

1. Client submits entry creation/update via gRPC API
2. Service validates entry and persists to MySQL
3. For each assigned site, creates workload_entry_sites record with PENDING status
4. Audit event recorded
5. Site agents poll for pending entries on their configured interval
6. Agent creates/updates entry in local SPIRE server
7. Agent reports result back to central service
8. Status updated to SYNCED or FAILED with error details

#### 6.1.2 Synchronization States

| State | Description |
|-------|-------------|
| **PENDING** | Entry queued for synchronization to site |
| **SYNCED** | Entry successfully created/updated in SPIRE server |
| **FAILED** | Synchronization failed; error captured; will retry |
| **DELETING** | Entry deletion in progress at site |

### 6.2 Conflict Resolution

- **Central Wins:** Central service is source of truth; unmanaged SPIRE entries are flagged but not modified
- **Optimistic Locking:** Entry updates require matching revision to prevent lost updates
- **Idempotent Operations:** All sync operations are idempotent; safe to retry on failure

### 6.3 Retry Strategy

Failed synchronizations use exponential backoff with jitter:

| Parameter | Value |
|-----------|-------|
| Initial retry | 5 seconds |
| Maximum interval | 5 minutes |
| Backoff multiplier | 2x |
| Jitter | ±20% |
| Max retries before alerting | 10 |

---

## 7. Kubernetes Deployment

### 7.1 Central Service Components

| Component | Replicas | Type | Notes |
|-----------|----------|------|-------|
| spire-mgmt-api | 3+ | Deployment | gRPC API server, horizontally scalable |
| spire-mgmt-sync | 2 | Deployment | Sync coordinator, leader-elected |
| mysql | 3 | StatefulSet | Primary + 2 replicas, or managed service |

### 7.2 Resource Requirements

| Component | CPU | Memory | Storage |
|-----------|-----|--------|---------|
| spire-mgmt-api | 500m-2000m | 512Mi-2Gi | N/A (stateless) |
| spire-mgmt-sync | 250m-1000m | 256Mi-1Gi | N/A (stateless) |
| mysql | 1000m-4000m | 2Gi-8Gi | 100Gi+ SSD |
| spire-site-agent | 100m-500m | 128Mi-512Mi | N/A (stateless) |

### 7.3 High Availability

- **API Server:** Multiple replicas behind load balancer
- **Sync Coordinator:** Leader election with standby failover
- **Database:** MySQL group replication or managed service
- **Site Agent:** Leader election per site with standby failover

---

## 8. Security Design

### 8.1 Authentication

| Method | Use Case |
|--------|----------|
| mTLS with SPIFFE SVIDs | Primary method for service-to-service communication |
| OIDC/JWT Tokens | User authentication via Backstage (Okta) integration |
| API Keys | Programmatic access from CI/CD pipelines |

### 8.2 Authorization (RBAC)

| Role | Permissions |
|------|-------------|
| Admin | Full access to all operations including site management |
| Operator | Manage entries and view sites; cannot modify site configuration |
| Developer | Create/modify entries in allowed namespaces only |
| Viewer | Read-only access to entries and sites |
| Site Agent | Poll entries and report sync status for assigned site only |

### 8.3 Data Security

- **Encryption at Rest:** Database encryption using AES-256
- **Encryption in Transit:** All gRPC connections use TLS 1.3
- **Secrets Management:** Integration with Kubernetes secrets or external vault
- **Audit Logging:** All mutations recorded with actor identity and timestamp

---

## 9. Observability

### 9.1 Metrics (Prometheus)

| Metric | Type | Description |
|--------|------|-------------|
| `grpc_requests_total` | Counter | Total API requests by method |
| `grpc_request_duration` | Histogram | Request latency distribution |
| `entries_total` | Gauge | Total managed entries |
| `sync_status_by_site` | Gauge | Entry count by site and sync status |
| `sync_lag_seconds` | Gauge | Time since last successful sync per site |
| `sync_failures_total` | Counter | Total sync failures by site and reason |

### 9.2 Logging & Tracing

Structured JSON logging with fields:

| Field | Description |
|-------|-------------|
| `timestamp` | RFC3339 format |
| `level` | DEBUG, INFO, WARN, ERROR |
| `trace_id` | Distributed tracing correlation |
| `component` | Service component name |
| `actor` | Authenticated identity |

OpenTelemetry integration for distributed tracing across all components.

### 9.3 Critical Alerts

- Site agent disconnected > 5 minutes
- Sync backlog > 100 pending entries
- Repeated sync failures for specific entries
- Database replication lag exceeding threshold
- API error rate exceeding threshold

---

## 10. Backstage Integration

### 10.1 Plugin Architecture

#### Frontend Plugin

- Workload entry listing, search, and creation wizard
- Site selection and sync status visualization
- Audit history viewer and catalog entity integration

#### Backend Plugin

- gRPC-Web proxy for browser communication
- Authentication bridge to Backstage identity
- Catalog integration for component-to-entry mapping

### 10.2 Developer Workflow

```
┌─────────────────────────────────────────────────────────────────┐
│  1. Developer navigates to SPIFFE Management in Backstage       │
│                              ↓                                  │
│  2. Creates new workload entry request via form                 │
│                              ↓                                  │
│  3. Selects target sites (may require approval)                 │
│                              ↓                                  │
│  4. Monitors sync status in real-time                           │
│                              ↓                                  │
│  5. Entry becomes active across selected sites                  │
└─────────────────────────────────────────────────────────────────┘
```

---

## 11. Implementation Phases

### Phase 1: Core Platform (Weeks 1-4)

- Database schema and migrations
- Core gRPC API implementation
- Basic CRUD operations
- Authentication framework
- Unit and integration tests

### Phase 2: Synchronization (Weeks 5-8)

- Site agent implementation
- Sync coordinator service
- SPIRE server integration
- Retry logic and error handling
- Multi-site E2E testing

### Phase 3: Observability & Security (Weeks 9-10)

- Metrics and tracing implementation
- Audit logging
- RBAC implementation
- Security hardening
- Alerting configuration

### Phase 4: Backstage Integration (Weeks 11-12)

- Frontend plugin development
- Backend plugin development
- Catalog integration
- User acceptance testing
- Documentation

---

## 12. Appendix

### A. Glossary

| Term | Definition |
|------|------------|
| SPIFFE | Secure Production Identity Framework for Everyone |
| SPIRE | SPIFFE Runtime Environment |
| SVID | SPIFFE Verifiable Identity Document |
| Trust Domain | Administrative domain for SPIFFE identities |
| Workload Entry | Registration record mapping workload attestation to SPIFFE ID |
| Selector | Attribute used to match workloads (e.g., k8s:ns:production) |
| Site | Deployment location with its own SPIRE server |

### B. References

1. [SPIFFE Specification](https://spiffe.io/docs/latest/spiffe-about/)
2. [SPIRE Documentation](https://spiffe.io/docs/latest/spire-about/)
3. [Backstage Documentation](https://backstage.io/docs/)
4. [gRPC Documentation](https://grpc.io/docs/)

### C. Revision History

| Version | Date | Author | Changes |
|---------|------|--------|---------|
| 1.0 | Dec 2025 | - | Initial draft |
