# SPIFFE/SPIRE Workload Entry Management System

## 2-Week Hackathon Plan

**Proof of Concept Implementation**

| | |
|---|---|
| **Team Size** | 3 Engineers |
| **Duration** | 10 Working Days |
| **Environment** | Internal Dev Kubernetes Cluster |
| **Authentication** | Okta OIDC |
| **Portal** | Backstage (existing instance) |
| **Deliverable** | Working demo with live SPIRE sync |

---

## Table of Contents

- [1. Executive Summary](#1-executive-summary)
- [2. Team Structure & Roles](#2-team-structure--roles)
- [3. POC Scope Definition](#3-poc-scope-definition)
- [4. Day-by-Day Schedule](#4-day-by-day-schedule)
- [5. Technical Specifications](#5-technical-specifications)
- [6. Demo Script](#6-demo-script)
- [7. Risk Mitigation](#7-risk-mitigation)
- [8. Prerequisites Checklist](#8-prerequisites-checklist)
- [9. Post-Hackathon Recommendations](#9-post-hackathon-recommendations)
- [10. Appendix](#10-appendix)

---

## 1. Executive Summary

This hackathon plan outlines a focused 2-week sprint to build a proof-of-concept SPIFFE/SPIRE Workload Entry Management System. The goal is to demonstrate the core value proposition: a unified API and Backstage UI for managing workload identities across multiple SPIRE deployments.

### 1.1 Demo Objectives

At the end of Week 2, the team will demonstrate:

1. Creating a workload entry via Backstage UI
2. Entry automatically syncing to 2 SPIRE servers (simulating multi-site)
3. Real-time sync status updates in the UI
4. Basic audit trail showing who created what and when
5. Authentication via Okta OIDC

### 1.2 Success Criteria

| Criteria | Description |
|----------|-------------|
| **Functional** | End-to-end flow works - entry created in UI appears in SPIRE servers |
| **Reliable** | Demo runs without manual intervention or restarts |
| **Understandable** | Non-technical stakeholders can follow the demo flow |
| **Extensible** | Architecture clearly supports adding more sites and features |

---

## 2. Team Structure & Roles

| Role | Focus Area | Responsibilities |
|------|------------|------------------|
| **Engineer A** | Backend / API | gRPC service, MySQL schema, sync engine, site agent |
| **Engineer B** | Frontend / Backstage | Backstage plugin (FE + BE), UI components, Okta integration |
| **Engineer C** | Infrastructure / DevOps | K8s deployment, SPIRE setup, CI/CD, demo environment |

### Collaboration Model

- Daily 15-min standup at 9:30 AM
- Shared Slack channel for async communication
- End-of-day integration sync (30 min) to merge work
- Shared Git repo with feature branches, PR reviews required

---

## 3. POC Scope Definition

### 3.1 In Scope (Must Have for Demo)

| Component | POC Implementation |
|-----------|-------------------|
| **gRPC API** | CreateEntry, GetEntry, ListEntries, DeleteEntry operations |
| **Data Store** | MySQL with entries, selectors, sites, entry_sites tables |
| **Sites** | 2 SPIRE servers (site-a, site-b) in same cluster, different namespaces |
| **Sync** | Basic polling agent that syncs entries to local SPIRE server |
| **Backstage UI** | Entry list view, create form, sync status indicator |
| **Auth** | Okta OIDC via Backstage, JWT validation in API |
| **Audit** | Basic created_by/created_at fields, no separate audit log |

### 3.2 Out of Scope (Post-Hackathon)

- High availability / leader election
- Full RBAC implementation (POC uses basic auth check)
- Retry logic with exponential backoff
- Batch operations
- Streaming sync updates
- Metrics, tracing, alerting
- Production hardening and security review
- Multi-cluster deployment (POC uses single cluster with namespaces)

### 3.3 Simplified Architecture for POC

```
┌─────────────────────────────────────────────────────────────────┐
│                      DEV KUBERNETES CLUSTER                     │
├─────────────────────────────────────────────────────────────────┤
│  ┌─────────────────────────────────────────────────────────┐   │
│  │              NAMESPACE: spire-mgmt                       │   │
│  │  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐   │   │
│  │  │  Backstage   │  │   gRPC API   │  │    MySQL     │   │   │
│  │  │   + Plugin   │─▶│   Service    │─▶│  (single)    │   │   │
│  │  └──────────────┘  └──────┬───────┘  └──────────────┘   │   │
│  └───────────────────────────┼─────────────────────────────┘   │
│                              │                                  │
│          ┌───────────────────┴───────────────────┐              │
│          ▼                                       ▼              │
│  ┌───────────────────────┐       ┌───────────────────────┐     │
│  │  NAMESPACE: site-a    │       │  NAMESPACE: site-b    │     │
│  │  ┌─────────────────┐  │       │  ┌─────────────────┐  │     │
│  │  │   Site Agent    │  │       │  │   Site Agent    │  │     │
│  │  └────────┬────────┘  │       │  └────────┬────────┘  │     │
│  │           ▼           │       │           ▼           │     │
│  │  ┌─────────────────┐  │       │  ┌─────────────────┐  │     │
│  │  │  SPIRE Server   │  │       │  │  SPIRE Server   │  │     │
│  │  └─────────────────┘  │       │  └─────────────────┘  │     │
│  └───────────────────────┘       └───────────────────────┘     │
└─────────────────────────────────────────────────────────────────┘
```

---

## 4. Day-by-Day Schedule

### Week 1: Foundation & Core Services

#### Day 1 (Monday) - Setup & Planning

| Engineer | Tasks |
|----------|-------|
| A (Backend) | Set up Go project structure, define protobuf schemas, generate gRPC stubs |
| B (Frontend) | Create Backstage plugin scaffold, configure Okta OIDC in Backstage |
| C (DevOps) | Provision dev cluster namespaces, deploy MySQL, set up SPIRE servers |
| **All** | Kickoff meeting (1hr): align on scope, APIs, data model, demo script |

> **Day 1 Deliverable:** Working dev environment with SPIRE servers responding to health checks

---

#### Day 2 (Tuesday) - Data Layer

| Engineer | Tasks |
|----------|-------|
| A (Backend) | Create MySQL schema migrations, implement repository layer (CRUD for entries) |
| B (Frontend) | Build entry list component (mock data), design create entry form |
| C (DevOps) | Configure SPIRE workload registrar, test manual entry creation in SPIRE |

> **Day 2 Deliverable:** MySQL schema deployed, can manually insert/query entries

---

#### Day 3 (Wednesday) - API Implementation

| Engineer | Tasks |
|----------|-------|
| A (Backend) | Implement CreateEntry, GetEntry, ListEntries gRPC handlers |
| B (Frontend) | Connect frontend to gRPC-Web proxy, display real entries from API |
| C (DevOps) | Deploy API service to cluster, configure Envoy gRPC-Web proxy |

> **Day 3 Deliverable:** API deployed, can create entry via grpcurl and see it in DB

---

#### Day 4 (Thursday) - Site Agent Foundation

| Engineer | Tasks |
|----------|-------|
| A (Backend) | Build site agent: poll API for pending entries, update sync status |
| B (Frontend) | Implement create entry form with site selection, form validation |
| C (DevOps) | Configure site agent SPIFFE identity, deploy to site-a namespace |

> **Day 4 Deliverable:** Site agent polling API, logs show entries being picked up

---

#### Day 5 (Friday) - SPIRE Integration

| Engineer | Tasks |
|----------|-------|
| A (Backend) | Integrate site agent with SPIRE Registration API, create entries in SPIRE |
| B (Frontend) | Add sync status display to entry list (pending/synced/failed badges) |
| C (DevOps) | Deploy site agent to site-b, verify both sites receiving entries |
| **All** | Week 1 review (1hr): demo current state, identify blockers for Week 2 |

> **🎯 Week 1 Milestone:** Entry created via API appears in both SPIRE servers

---

### Week 2: Integration & Polish

#### Day 6 (Monday) - Backstage Backend Plugin

| Engineer | Tasks |
|----------|-------|
| A (Backend) | Add JWT validation middleware, extract user identity from Okta token |
| B (Frontend) | Build Backstage backend plugin to proxy gRPC calls, pass auth headers |
| C (DevOps) | Configure Backstage deployment with plugin, set up Okta app registration |

> **Day 6 Deliverable:** Backstage authenticates user, passes identity to API

---

#### Day 7 (Tuesday) - End-to-End Flow

| Engineer | Tasks |
|----------|-------|
| A (Backend) | Implement DeleteEntry with cascade to SPIRE, add created_by tracking |
| B (Frontend) | Complete create entry flow: form → API → success message → list refresh |
| C (DevOps) | Test E2E flow, document any environment issues, create troubleshooting guide |

> **Day 7 Deliverable:** Can create entry in Backstage UI and see it sync to SPIRE

---

#### Day 8 (Wednesday) - UI Polish & Error Handling

| Engineer | Tasks |
|----------|-------|
| A (Backend) | Add error handling, meaningful error messages, input validation |
| B (Frontend) | Polish UI: loading states, error messages, confirmation dialogs, styling |
| C (DevOps) | Set up demo environment (separate from dev), prepare demo data/scripts |

> **Day 8 Deliverable:** UI handles errors gracefully, no ugly crashes during demo

---

#### Day 9 (Thursday) - Demo Prep & Dry Run

| Engineer | Tasks |
|----------|-------|
| A (Backend) | Fix any bugs found in dry run, add simple entry detail view |
| B (Frontend) | Create demo walkthrough slides, practice demo script |
| C (DevOps) | Prepare demo reset script, verify all services healthy |
| **All** | Full dry run of demo (2hr): identify issues, practice narrative |

> **Day 9 Deliverable:** Successful dry run completed, demo script finalized

---

#### Day 10 (Friday) - Demo Day 🎉

| Time | Activity |
|------|----------|
| 9:00 AM | Final environment check, reset demo data |
| 10:00 AM | Team rehearsal (last chance to catch issues) |
| **2:00 PM** | **🎯 DEMO PRESENTATION (45 min + 15 min Q&A)** |
| 3:30 PM | Retrospective: what worked, what to improve, next steps |

---

## 5. Technical Specifications

### 5.1 Technology Stack

| Component | Technology |
|-----------|------------|
| API Service | Go 1.21+, gRPC, protobuf |
| Database | MySQL 8.0 (single instance for POC) |
| Site Agent | Go 1.21+, SPIRE SDK |
| Backstage Plugin | TypeScript, React, Material-UI |
| gRPC-Web Proxy | Envoy or grpc-web standalone proxy |
| Auth | Okta OIDC, JWT validation via go-oidc |
| Container Runtime | Docker, Kubernetes 1.28+ |
| SPIRE | SPIRE 1.8+ (server + agent) |

### 5.2 Simplified Database Schema

```sql
-- POC Schema (simplified from full design)

CREATE TABLE sites (
  id VARCHAR(36) PRIMARY KEY,
  name VARCHAR(255) NOT NULL UNIQUE,
  spire_server_addr VARCHAR(255) NOT NULL,
  trust_domain VARCHAR(255) NOT NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE workload_entries (
  id VARCHAR(36) PRIMARY KEY,
  spiffe_id VARCHAR(2048) NOT NULL,
  parent_id VARCHAR(2048) NOT NULL,
  ttl INT DEFAULT 3600,
  created_by VARCHAR(255),
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

CREATE TABLE workload_entry_selectors (
  id VARCHAR(36) PRIMARY KEY,
  workload_entry_id VARCHAR(36) NOT NULL,
  type VARCHAR(255) NOT NULL,
  value VARCHAR(4096) NOT NULL,
  FOREIGN KEY (workload_entry_id) REFERENCES workload_entries(id) ON DELETE CASCADE
);

CREATE TABLE workload_entry_sites (
  workload_entry_id VARCHAR(36) NOT NULL,
  site_id VARCHAR(36) NOT NULL,
  sync_status ENUM('PENDING', 'SYNCED', 'FAILED') DEFAULT 'PENDING',
  spire_entry_id VARCHAR(255),
  last_error TEXT,
  last_sync_at TIMESTAMP,
  PRIMARY KEY (workload_entry_id, site_id),
  FOREIGN KEY (workload_entry_id) REFERENCES workload_entries(id) ON DELETE CASCADE,
  FOREIGN KEY (site_id) REFERENCES sites(id)
);
```

### 5.3 POC API Definition

```protobuf
syntax = "proto3";
package spire.mgmt.v1;

service WorkloadEntryService {
  rpc CreateEntry(CreateEntryRequest) returns (Entry);
  rpc GetEntry(GetEntryRequest) returns (Entry);
  rpc ListEntries(ListEntriesRequest) returns (ListEntriesResponse);
  rpc DeleteEntry(DeleteEntryRequest) returns (DeleteEntryResponse);
}

message Entry {
  string id = 1;
  string spiffe_id = 2;
  string parent_id = 3;
  repeated Selector selectors = 4;
  int32 ttl = 5;
  repeated string site_ids = 6;
  repeated SiteStatus site_statuses = 7;
  string created_by = 8;
  string created_at = 9;
}

message Selector {
  string type = 1;
  string value = 2;
}

message SiteStatus {
  string site_id = 1;
  string site_name = 2;
  string status = 3;  // PENDING, SYNCED, FAILED
  string last_error = 4;
}
```

---

## 6. Demo Script

**Target Duration:** 45 minutes including Q&A

### 6.1 Demo Flow

#### Part 1: Problem Statement (5 min)

- Show current pain: SSH into two different SPIRE servers
- Demonstrate manual entry creation with spire-server CLI
- Highlight: no visibility, no audit, error-prone, doesn't scale

#### Part 2: Solution Overview (5 min)

- Show architecture diagram
- Explain: central API, site agents, Backstage integration
- Mention: built on existing infra (K8s, Okta, Backstage)

#### Part 3: Live Demo (20 min)

| Step | Action | Duration |
|------|--------|----------|
| 1 | **Login to Backstage** - Show Okta authentication flow | 2 min |
| 2 | **View existing entries** - Show list view with sync status badges | 2 min |
| 3 | **Create new entry** - Fill form: SPIFFE ID, selectors, select both sites | 5 min |
| 4 | **Watch sync happen** - Status changes from PENDING to SYNCED | 3 min |
| 5 | **Verify in SPIRE** - Show entry exists in both SPIRE servers via CLI | 3 min |
| 6 | **Delete entry** - Show cascade delete from both sites | 3 min |
| 7 | **Show audit info** - Point out created_by field shows Okta user | 2 min |

#### Part 4: Roadmap & Discussion (15 min)

- Show full design document
- Discuss: what's needed for production (HA, RBAC, monitoring)
- Proposed timeline: 12 weeks to production MVP
- Q&A

### 6.2 Demo Backup Plan

If live demo fails, have these ready:

- ✅ Pre-recorded video of the full flow
- ✅ Screenshots of each step
- ✅ Terminal output showing API calls working
- ✅ Database queries showing data flow

---

## 7. Risk Mitigation

| Risk | Impact | Mitigation |
|------|--------|------------|
| SPIRE API complexity | Agent can't create entries | Day 1: spike on SPIRE SDK, have fallback to CLI wrapper |
| Okta integration issues | Can't demo auth flow | Fallback: hardcode test user, show Okta config separately |
| gRPC-Web proxy issues | Frontend can't reach API | Have REST fallback endpoint for critical paths |
| Scope creep | Don't finish core features | Daily scope check, strict "out of scope" list |
| Environment instability | Demo fails | Separate demo env, reset scripts, recorded backup |
| Team member unavailable | Work blocked | Daily knowledge sharing, documented handoffs |

---

## 8. Prerequisites Checklist

Complete before Day 1:

### 8.1 Access & Permissions

- [ ] Dev Kubernetes cluster access for all 3 engineers
- [ ] Permission to create namespaces and deploy workloads
- [ ] Okta admin access to create/configure application
- [ ] Backstage instance access with plugin deployment rights
- [ ] Git repository created with CI/CD pipeline

### 8.2 Infrastructure

- [ ] Container registry access for pushing images
- [ ] MySQL instance provisioned (or permission to deploy)
- [ ] Network policies allow inter-namespace communication
- [ ] Ingress controller available for Backstage/API exposure

### 8.3 Documentation & Reference

- [ ] SPIRE documentation reviewed by Engineer A and C
- [ ] Backstage plugin development guide reviewed by Engineer B
- [ ] Okta OIDC setup guide available
- [ ] Full design document shared with team

---

## 9. Post-Hackathon Recommendations

If the POC is successful and approved for production development:

### 9.1 Immediate Next Steps (Weeks 3-4)

1. Code review and refactoring of POC code
2. Add comprehensive unit and integration tests
3. Implement proper error handling and retry logic
4. Security review of authentication/authorization

### 9.2 Production Readiness (Weeks 5-12)

| Area | Tasks |
|------|-------|
| **High Availability** | Leader election, multiple replicas |
| **Security** | Full RBAC implementation per design document |
| **Observability** | Metrics, tracing, alerting |
| **Scale** | Multi-cluster deployment (actual separate clusters) |
| **Features** | Batch operations and streaming updates |
| **Compliance** | Full audit logging with retention |
| **Operations** | Documentation and runbooks |

---

## 10. Appendix

### A. Key Contacts

| Role | Name | Responsibility |
|------|------|----------------|
| Project Sponsor | [TBD] | Final approval, resource allocation |
| Tech Lead | [TBD] | Technical decisions, architecture |
| Platform Team | [TBD] | K8s cluster, infrastructure support |
| Security Team | [TBD] | Okta config, security review |

### B. Reference Links

- [SPIRE Documentation](https://spiffe.io/docs/latest/spire-about/)
- [Backstage Plugin Development](https://backstage.io/docs/plugins/)
- [gRPC-Web](https://github.com/grpc/grpc-web)
- [Okta OIDC](https://developer.okta.com/docs/guides/)

### C. Revision History

| Version | Date | Author | Changes |
|---------|------|--------|---------|
| 1.0 | Dec 2025 | - | Initial hackathon plan |
