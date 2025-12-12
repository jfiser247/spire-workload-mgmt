# SPIRE Workload Management - Demo Guide

This guide walks you through setting up and running the alpha demo of the SPIFFE/SPIRE Workload Entry Management System.

## Prerequisites

Before starting, ensure you have the following installed:

- **Docker Desktop** (or Docker Engine)
- **minikube** - [Install Guide](https://minikube.sigs.k8s.io/docs/start/)
- **kubectl** - [Install Guide](https://kubernetes.io/docs/tasks/tools/)
- **Helm 3** - [Install Guide](https://helm.sh/docs/intro/install/)

Verify installations:
```bash
docker --version
minikube version
kubectl version --client
helm version
```

## Quick Start

### 1. Start Minikube

```bash
./scripts/setup-minikube.sh
```

This starts minikube with 4 CPUs and 8GB RAM. If minikube is already running, it will be restarted.

### 2. Deploy Everything

```bash
./scripts/setup-all.sh
```

This script will:
- Add required Helm repositories (Bitnami, SPIFFE)
- Create namespaces: `spire-mgmt`, `site-a`, `site-b`
- Deploy MySQL with the database schema
- Build Docker images for API server, site agents, and UI
- Deploy all components via Helm
- Seed initial site configuration

**Expected time:** ~5 minutes on first run

### 3. Access the Demo

Open **two terminal windows** for port forwarding:

**Terminal 1 - API Server:**
```bash
kubectl port-forward svc/api-spire-mgmt-api 8081:8081 -n spire-mgmt
```

**Terminal 2 - UI:**
```bash
kubectl port-forward svc/ui-spire-mgmt-ui 3000:3000 -n spire-mgmt
```

**Open the UI:** http://localhost:3000

---

## Running the Demo

### Demo Script (20 minutes)

#### Step 1: View the Dashboard (2 min)

1. Open http://localhost:3000
2. Note the **demo-user** badge in the header (no auth required)
3. Click **Sites** tab to see configured sites:
   - US East (site-a)
   - EU West (site-b)
4. Click **Workload Entries** tab - should be empty initially

#### Step 2: Create a Workload Entry (5 min)

1. Click **+ Create Entry** button
2. Fill in the form:
   - **SPIFFE ID:** `spiffe://example.org/workload/api-service`
   - **Parent ID:** (leave default)
   - **Selectors:**
     - Type: `k8s:ns` Value: `production`
     - Click "+ Add Selector"
     - Type: `k8s:sa` Value: `api-service`
   - **Target Sites:** Check both "US East" and "EU West"
   - **Description:** `API service workload identity`
3. Click **Create Entry**

#### Step 3: Watch Sync Status (3 min)

1. Observe the new entry in the table
2. Watch the **Site Status** column:
   - Initially shows `PENDING` (yellow) for both sites
   - Within 10-15 seconds, changes to `SYNCED` (green)
3. Explain: Site agents poll every 10 seconds and sync entries to SPIRE

#### Step 4: Verify in SPIRE (3 min)

Open a new terminal and run:
```bash
# Check site-a SPIRE server
kubectl logs -n site-a -l app.kubernetes.io/name=site-agent --tail=20

# Check site-b SPIRE server
kubectl logs -n site-b -l app.kubernetes.io/name=site-agent --tail=20
```

You should see logs like:
```
[site-a] Syncing entry xxx (SPIFFE ID: spiffe://example.org/workload/api-service)
[site-a] Created SPIRE entry spire-xxxxxxxx for xxx
```

#### Step 5: Create Another Entry (2 min)

1. Click **+ Create Entry**
2. Create a second entry:
   - **SPIFFE ID:** `spiffe://example.org/workload/web-frontend`
   - **Selectors:** `k8s:ns` = `production`
   - **Target Sites:** Only "US East"
3. Note this entry only syncs to one site

#### Step 6: Delete an Entry (3 min)

1. Click **Delete** on the first entry (api-service)
2. Confirm the deletion
3. Watch the entry disappear from the table
4. Check site agent logs to see deletion propagated

#### Step 7: View Audit Log (2 min)

1. Click **Audit Log** tab
2. Show the audit trail:
   - `create` actions for entries created
   - `delete` action for the deleted entry
   - `sync` actions from site agents
3. Point out the **Actor** column shows `demo-user` for UI actions

---

## Resetting the Demo

### Option 1: Quick Data Reset (Instant)

Reset database tables without restarting services:
```bash
./scripts/reset-demo.sh
```

### Option 2: Redeploy Apps (~2 min)

Remove and redeploy application components:
```bash
./scripts/teardown.sh
./scripts/deploy-apps.sh
```

### Option 3: Full Reset (~5 min)

Complete teardown and redeploy:
```bash
./scripts/teardown.sh --full
./scripts/setup-all.sh
```

---

## Troubleshooting

### Check Pod Status

```bash
kubectl get pods -A | grep -E '(spire-mgmt|site-a|site-b)'
```

All pods should be `Running` with `1/1` ready.

### View Logs

```bash
# API Server logs
kubectl logs -n spire-mgmt -l app.kubernetes.io/name=spire-mgmt-api -f

# Site Agent logs
kubectl logs -n site-a -l app.kubernetes.io/name=site-agent -f
kubectl logs -n site-b -l app.kubernetes.io/name=site-agent -f

# UI logs
kubectl logs -n spire-mgmt -l app.kubernetes.io/name=spire-mgmt-ui -f
```

### MySQL Connection Issues

```bash
# Check MySQL is running
kubectl get pods -n spire-mgmt -l app.kubernetes.io/name=mysql

# View MySQL logs
kubectl logs -n spire-mgmt -l app.kubernetes.io/name=mysql

# Connect to MySQL directly
kubectl exec -it -n spire-mgmt $(kubectl get pods -n spire-mgmt -l app.kubernetes.io/name=mysql -o jsonpath='{.items[0].metadata.name}') -- mysql -u root -pdemo-password spire_mgmt
```

### Port Forward Issues

If port forward disconnects, restart it:
```bash
# Kill existing port forwards
pkill -f "port-forward"

# Restart
kubectl port-forward svc/api-spire-mgmt-api 8081:8081 -n spire-mgmt &
kubectl port-forward svc/ui-spire-mgmt-ui 3000:3000 -n spire-mgmt &
```

### Image Pull Errors

If pods show `ImagePullBackOff`:
```bash
# Ensure you're using minikube's Docker daemon
eval $(minikube docker-env)

# Rebuild images
docker build -t spire-mgmt-api:alpha -f deploy/docker/Dockerfile.api .
docker build -t site-agent:alpha -f deploy/docker/Dockerfile.agent .
docker build -t spire-mgmt-ui:alpha -f backstage-plugin/Dockerfile backstage-plugin/

# Restart deployments
kubectl rollout restart deployment -n spire-mgmt
kubectl rollout restart deployment -n site-a
kubectl rollout restart deployment -n site-b
```

---

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                      MINIKUBE CLUSTER                            │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │              NAMESPACE: spire-mgmt                       │   │
│  │                                                          │   │
│  │  ┌──────────┐  ┌──────────────┐  ┌──────────────┐       │   │
│  │  │    UI    │  │  API Server  │  │    MySQL     │       │   │
│  │  │  :3000   │──│  :8080/8081  │──│    :3306     │       │   │
│  │  │ (React)  │  │   (Go/gRPC)  │  │   (Bitnami)  │       │   │
│  │  └──────────┘  └──────┬───────┘  └──────────────┘       │   │
│  │                       │                                  │   │
│  └───────────────────────┼──────────────────────────────────┘   │
│                          │                                       │
│          ┌───────────────┴───────────────┐                      │
│          │         HTTP REST API         │                      │
│          ▼                               ▼                      │
│  ┌───────────────────────┐   ┌───────────────────────┐         │
│  │  NAMESPACE: site-a    │   │  NAMESPACE: site-b    │         │
│  │                       │   │                       │         │
│  │  ┌─────────────────┐  │   │  ┌─────────────────┐  │         │
│  │  │   Site Agent    │  │   │  │   Site Agent    │  │         │
│  │  │  (polls API)    │  │   │  │  (polls API)    │  │         │
│  │  └────────┬────────┘  │   │  └────────┬────────┘  │         │
│  │           │           │   │           │           │         │
│  │           ▼           │   │           ▼           │         │
│  │  ┌─────────────────┐  │   │  ┌─────────────────┐  │         │
│  │  │  SPIRE Server   │  │   │  │  SPIRE Server   │  │         │
│  │  │  (mock in alpha)│  │   │  │  (mock in alpha)│  │         │
│  │  └─────────────────┘  │   │  └─────────────────┘  │         │
│  └───────────────────────┘   └───────────────────────┘         │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

### Data Flow

1. **User creates entry** via UI (React) → REST API
2. **API Server** stores entry in MySQL with `PENDING` status for selected sites
3. **Site Agents** poll API every 10 seconds for pending entries
4. **Site Agent** creates entry in local SPIRE server
5. **Site Agent** reports success → API updates status to `SYNCED`
6. **UI** polls API every 5 seconds, shows updated status

---

## Demo Tips

### Before the Demo

1. Run through the demo once to ensure everything works
2. Pre-create a backup of the commands you'll run
3. Have the architecture diagram ready to show
4. Prepare talking points for each step

### During the Demo

1. Keep terminal windows arranged for easy viewing
2. Zoom in on the UI for visibility
3. Explain what's happening at each step
4. Show logs to prove sync is actually happening
5. If something fails, use it as a teaching moment about distributed systems

### Backup Plan

If the live demo fails:
1. Have screenshots ready
2. Walk through the code instead
3. Show the architecture and explain the flow

---

## Scripts Reference

| Script | Description |
|--------|-------------|
| `./scripts/setup-minikube.sh` | Start/restart minikube |
| `./scripts/setup-all.sh` | Full deployment |
| `./scripts/deploy-apps.sh` | Rebuild and deploy apps only |
| `./scripts/reset-demo.sh` | Clear database tables |
| `./scripts/teardown.sh` | Remove apps, keep MySQL |
| `./scripts/teardown.sh --full` | Remove everything |
| `./scripts/seed-demo-data.sh` | Insert site configurations |

---

## Next Steps (Post-Demo)

For production deployment, the following would be needed:

1. **Real SPIRE Integration** - Connect site agents to actual SPIRE servers
2. **Authentication** - Integrate with Okta/OIDC via Backstage
3. **RBAC** - Implement role-based access control
4. **High Availability** - Multiple API replicas, MySQL replication
5. **Monitoring** - Prometheus metrics, Grafana dashboards
6. **Multi-Cluster** - Deploy to separate Kubernetes clusters

See `docs/DESIGN.md` for the full production architecture.
