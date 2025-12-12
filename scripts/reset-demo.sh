#!/bin/bash
set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${YELLOW}=== Resetting Demo Data ===${NC}"

# Get MySQL pod
MYSQL_POD=$(kubectl get pods -n spire-mgmt -l app.kubernetes.io/name=mysql -o jsonpath='{.items[0].metadata.name}')

if [ -z "$MYSQL_POD" ]; then
    echo -e "${RED}MySQL pod not found. Is the cluster running?${NC}"
    exit 1
fi

# Clear workload entries and audit log
echo -e "${YELLOW}Clearing database tables...${NC}"
kubectl exec -n spire-mgmt "$MYSQL_POD" -- mysql -u root -pdemo-password spire_mgmt -e "
SET FOREIGN_KEY_CHECKS = 0;
TRUNCATE TABLE site_workload_entries;
TRUNCATE TABLE workload_entries;
TRUNCATE TABLE audit_log;
SET FOREIGN_KEY_CHECKS = 1;
"

# Clear SPIRE entries in site-a
echo -e "${YELLOW}Clearing SPIRE entries in site-a...${NC}"
kubectl exec -n site-a deploy/spire-a-server -- /opt/spire/bin/spire-server entry show -output json 2>/dev/null | \
    jq -r '.entries[]?.id // empty' | \
    while read -r entry_id; do
        if [ -n "$entry_id" ]; then
            kubectl exec -n site-a deploy/spire-a-server -- /opt/spire/bin/spire-server entry delete -entryID "$entry_id" 2>/dev/null || true
        fi
    done

# Clear SPIRE entries in site-b
echo -e "${YELLOW}Clearing SPIRE entries in site-b...${NC}"
kubectl exec -n site-b deploy/spire-b-server -- /opt/spire/bin/spire-server entry show -output json 2>/dev/null | \
    jq -r '.entries[]?.id // empty' | \
    while read -r entry_id; do
        if [ -n "$entry_id" ]; then
            kubectl exec -n site-b deploy/spire-b-server -- /opt/spire/bin/spire-server entry delete -entryID "$entry_id" 2>/dev/null || true
        fi
    done

echo -e "${GREEN}=== Demo data reset complete! ===${NC}"
