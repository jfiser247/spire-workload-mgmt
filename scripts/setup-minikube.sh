#!/bin/bash
set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}=== SPIRE Workload Management - Minikube Setup ===${NC}"

# Check prerequisites
echo -e "${YELLOW}Checking prerequisites...${NC}"

if ! command -v minikube &> /dev/null; then
    echo -e "${RED}minikube is not installed. Please install it first.${NC}"
    exit 1
fi

if ! command -v helm &> /dev/null; then
    echo -e "${RED}helm is not installed. Please install it first.${NC}"
    exit 1
fi

if ! command -v kubectl &> /dev/null; then
    echo -e "${RED}kubectl is not installed. Please install it first.${NC}"
    exit 1
fi

echo -e "${GREEN}All prerequisites found!${NC}"

# Check if minikube is already running
if minikube status &> /dev/null; then
    echo -e "${YELLOW}Minikube is already running. Deleting existing cluster...${NC}"
    minikube delete
fi

# Start minikube with adequate resources
echo -e "${YELLOW}Starting minikube with 4 CPUs and 8GB RAM...${NC}"
minikube start \
    --cpus=4 \
    --memory=8192 \
    --driver=docker \
    --kubernetes-version=v1.28.0

# Enable required addons
echo -e "${YELLOW}Enabling minikube addons...${NC}"
minikube addons enable metrics-server
minikube addons enable ingress

# Verify cluster is running
echo -e "${YELLOW}Verifying cluster status...${NC}"
kubectl cluster-info

echo -e "${GREEN}=== Minikube setup complete! ===${NC}"
echo ""
echo "Next steps:"
echo "  1. Run ./scripts/setup-all.sh to deploy all components"
echo "  2. Or run individual setup scripts for specific components"
