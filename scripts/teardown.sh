#!/bin/bash

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${YELLOW}=== SPIRE Workload Management - Teardown ===${NC}"

# Parse arguments
FULL_TEARDOWN=false
if [ "$1" == "--full" ] || [ "$1" == "-f" ]; then
    FULL_TEARDOWN=true
fi

# Uninstall our helm releases
echo -e "${YELLOW}Removing UI...${NC}"
helm uninstall ui -n spire-mgmt 2>/dev/null || true

echo -e "${YELLOW}Removing site agents...${NC}"
helm uninstall agent-a -n site-a 2>/dev/null || true
helm uninstall agent-b -n site-b 2>/dev/null || true

echo -e "${YELLOW}Removing API server...${NC}"
helm uninstall api -n spire-mgmt 2>/dev/null || true

if [ "$FULL_TEARDOWN" = true ]; then
    echo -e "${YELLOW}Removing MySQL (full teardown)...${NC}"
    helm uninstall mysql -n spire-mgmt 2>/dev/null || true
    kubectl delete -f "$SCRIPT_DIR/../deploy/k8s/mysql-deployment.yaml" 2>/dev/null || true

    # Delete PVCs to fully reset data
    echo -e "${YELLOW}Removing persistent volume claims...${NC}"
    kubectl delete pvc -n spire-mgmt --all 2>/dev/null || true

    echo -e "${YELLOW}Removing SPIRE installations...${NC}"
    helm uninstall spire-a -n site-a 2>/dev/null || true
    helm uninstall spire-b -n site-b 2>/dev/null || true
    helm uninstall spire-crds -n spire-mgmt 2>/dev/null || true

    echo -e "${YELLOW}Removing namespaces...${NC}"
    kubectl delete namespace site-a 2>/dev/null || true
    kubectl delete namespace site-b 2>/dev/null || true
    kubectl delete namespace spire-mgmt 2>/dev/null || true

    echo -e "${GREEN}=== Full teardown complete! ===${NC}"
    echo ""
    echo "To redeploy from scratch:"
    echo "  ./scripts/setup-all.sh"
else
    # Just reset the data, keep MySQL running
    echo -e "${YELLOW}Resetting demo data (keeping MySQL)...${NC}"

    MYSQL_POD=$(kubectl get pods -n spire-mgmt -l app=mysql -o jsonpath='{.items[0].metadata.name}' 2>/dev/null)

    if [ -n "$MYSQL_POD" ]; then
        kubectl exec -n spire-mgmt "$MYSQL_POD" -- mysql -u root -pdemo-password spire_mgmt -e "
SET FOREIGN_KEY_CHECKS = 0;
TRUNCATE TABLE site_workload_entries;
TRUNCATE TABLE workload_entries;
TRUNCATE TABLE audit_log;
SET FOREIGN_KEY_CHECKS = 1;
" 2>/dev/null || true
    fi

    echo -e "${GREEN}=== Quick teardown complete! ===${NC}"
    echo ""
    echo "MySQL is still running with empty tables."
    echo ""
    echo "To redeploy apps only (faster):"
    echo "  ./scripts/deploy-apps.sh"
    echo ""
    echo "For full teardown including MySQL:"
    echo "  ./scripts/teardown.sh --full"
fi
