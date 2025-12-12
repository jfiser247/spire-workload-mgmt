#!/bin/bash
set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

echo -e "${GREEN}=== SPIRE Workload Management - Full Setup ===${NC}"

# Check if minikube is running
if ! minikube status &> /dev/null; then
    echo -e "${YELLOW}Minikube is not running. Starting minikube...${NC}"
    minikube start --cpus=4 --memory=8192 --driver=docker
fi

# Add helm repos
echo -e "${YELLOW}Adding helm repositories...${NC}"
helm repo add spiffe https://spiffe.github.io/helm-charts-hardened/ || true
helm repo add bitnami https://charts.bitnami.com/bitnami || true
helm repo update

# Create namespaces
echo -e "${YELLOW}Creating namespaces...${NC}"
kubectl create namespace spire-mgmt --dry-run=client -o yaml | kubectl apply -f -
kubectl create namespace site-a --dry-run=client -o yaml | kubectl apply -f -
kubectl create namespace site-b --dry-run=client -o yaml | kubectl apply -f -

# Install MySQL (using official image)
echo -e "${YELLOW}Installing MySQL...${NC}"
kubectl apply -f "$PROJECT_DIR/deploy/k8s/mysql-deployment.yaml"

# Wait for MySQL to be ready
echo -e "${YELLOW}Waiting for MySQL to be ready...${NC}"
kubectl wait --for=condition=available deployment/mysql -n spire-mgmt --timeout=180s
# Give MySQL a bit more time to initialize the database
sleep 10

# Build and load Docker images into minikube
echo -e "${YELLOW}Building Docker images...${NC}"
eval $(minikube docker-env)

echo -e "${YELLOW}Building API server image...${NC}"
docker build -t spire-mgmt-api:alpha -f "$PROJECT_DIR/deploy/docker/Dockerfile.api" "$PROJECT_DIR"

echo -e "${YELLOW}Building site agent image...${NC}"
docker build -t site-agent:alpha -f "$PROJECT_DIR/deploy/docker/Dockerfile.agent" "$PROJECT_DIR"

echo -e "${YELLOW}Building UI image...${NC}"
docker build -t spire-mgmt-ui:alpha -f "$PROJECT_DIR/backstage-plugin/Dockerfile" "$PROJECT_DIR/backstage-plugin"

# Install API server
echo -e "${YELLOW}Installing API server...${NC}"
helm upgrade --install api "$PROJECT_DIR/deploy/helm/spire-mgmt-api" \
    -n spire-mgmt \
    --wait

# Wait for API to be ready
echo -e "${YELLOW}Waiting for API server to be ready...${NC}"
kubectl wait --for=condition=ready pod -l app.kubernetes.io/name=spire-mgmt-api -n spire-mgmt --timeout=120s || true

# Seed initial data
echo -e "${YELLOW}Seeding initial site data...${NC}"
"$SCRIPT_DIR/seed-demo-data.sh" || echo -e "${YELLOW}Warning: seed script may have issues, continuing...${NC}"

# Install site agents
echo -e "${YELLOW}Installing site agents...${NC}"
helm upgrade --install agent-a "$PROJECT_DIR/deploy/helm/site-agent" \
    -n site-a \
    --set siteId=site-a \
    --set siteName="US East" \
    --set apiServer.address="api-spire-mgmt-api.spire-mgmt.svc.cluster.local:8081" \
    --wait || true

helm upgrade --install agent-b "$PROJECT_DIR/deploy/helm/site-agent" \
    -n site-b \
    --set siteId=site-b \
    --set siteName="EU West" \
    --set apiServer.address="api-spire-mgmt-api.spire-mgmt.svc.cluster.local:8081" \
    --wait || true

# Install UI
echo -e "${YELLOW}Installing UI...${NC}"
helm upgrade --install ui "$PROJECT_DIR/deploy/helm/backstage" \
    -n spire-mgmt \
    --wait || true

echo -e "${GREEN}=== Setup complete! ===${NC}"
echo ""
echo "To access the services, run these port-forwards in separate terminals:"
echo ""
echo -e "${YELLOW}  # API Server (HTTP)${NC}"
echo "  kubectl port-forward svc/api-spire-mgmt-api 8081:8081 -n spire-mgmt"
echo ""
echo -e "${YELLOW}  # UI${NC}"
echo "  kubectl port-forward svc/ui-spire-mgmt-ui 3000:3000 -n spire-mgmt"
echo ""
echo -e "${GREEN}Then open: http://localhost:3000${NC}"
echo ""
echo "To check pod status:"
echo "  kubectl get pods -A | grep -E '(spire-mgmt|site-a|site-b)'"
