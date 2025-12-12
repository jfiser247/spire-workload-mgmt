#!/bin/bash
set -e

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo -e "${YELLOW}=== Seeding Demo Data ===${NC}"

# Get MySQL pod
MYSQL_POD=$(kubectl get pods -n spire-mgmt -l app.kubernetes.io/name=mysql -o jsonpath='{.items[0].metadata.name}')

if [ -z "$MYSQL_POD" ]; then
    echo -e "${RED}MySQL pod not found. Is the cluster running?${NC}"
    exit 1
fi

# Insert/update sites
echo -e "${YELLOW}Inserting site configurations...${NC}"
kubectl exec -n spire-mgmt "$MYSQL_POD" -- mysql -u root -pdemo-password spire_mgmt -e "
INSERT INTO sites (id, name, region, spire_server_address, trust_domain, status) VALUES
    ('site-a', 'US East', 'us-east-1', 'spire-a-server.site-a.svc.cluster.local:8081', 'site-a.demo', 'active'),
    ('site-b', 'EU West', 'eu-west-1', 'spire-b-server.site-b.svc.cluster.local:8081', 'site-b.demo', 'active')
ON DUPLICATE KEY UPDATE
    name = VALUES(name),
    region = VALUES(region),
    spire_server_address = VALUES(spire_server_address),
    trust_domain = VALUES(trust_domain),
    status = VALUES(status);
"

echo -e "${GREEN}=== Demo data seeded successfully! ===${NC}"
echo ""
echo "Available sites:"
kubectl exec -n spire-mgmt "$MYSQL_POD" -- mysql -u root -pdemo-password spire_mgmt -e "SELECT id, name, region, status FROM sites;"
