#!/bin/bash
set -e

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

echo -e "${GREEN}=== Deploying Apps (Quick Redeploy) ===${NC}"

# Build images
echo -e "${YELLOW}Building Docker images...${NC}"
eval $(minikube docker-env)

docker build -t spire-mgmt-api:alpha -f "$PROJECT_DIR/deploy/docker/Dockerfile.api" "$PROJECT_DIR"
docker build -t site-agent:alpha -f "$PROJECT_DIR/deploy/docker/Dockerfile.agent" "$PROJECT_DIR"
docker build -t spire-mgmt-ui:alpha -f "$PROJECT_DIR/backstage-plugin/Dockerfile" "$PROJECT_DIR/backstage-plugin"

# Deploy
echo -e "${YELLOW}Deploying API server...${NC}"
helm upgrade --install api "$PROJECT_DIR/deploy/helm/spire-mgmt-api" -n spire-mgmt --wait

echo -e "${YELLOW}Seeding data...${NC}"
"$SCRIPT_DIR/seed-demo-data.sh" || true

echo -e "${YELLOW}Deploying site agents...${NC}"
helm upgrade --install agent-a "$PROJECT_DIR/deploy/helm/site-agent" -n site-a \
    --set siteId=site-a --set siteName="US East" \
    --set apiServer.address="api-spire-mgmt-api.spire-mgmt.svc.cluster.local:8081" --wait || true

helm upgrade --install agent-b "$PROJECT_DIR/deploy/helm/site-agent" -n site-b \
    --set siteId=site-b --set siteName="EU West" \
    --set apiServer.address="api-spire-mgmt-api.spire-mgmt.svc.cluster.local:8081" --wait || true

echo -e "${YELLOW}Deploying UI...${NC}"
helm upgrade --install ui "$PROJECT_DIR/deploy/helm/backstage" -n spire-mgmt --wait || true

echo -e "${GREEN}=== Deploy complete! ===${NC}"
echo ""
echo "Port forward and access:"
echo "  kubectl port-forward svc/api-spire-mgmt-api 8081:8081 -n spire-mgmt &"
echo "  kubectl port-forward svc/ui-spire-mgmt-ui 3000:3000 -n spire-mgmt &"
echo "  open http://localhost:3000"
