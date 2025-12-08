# SPIFFE/SPIRE Workload Entry Management System

A centralized management system for SPIFFE workload entries across globally distributed Kubernetes clusters, integrated with Backstage.

## Documentation

- [docs/DESIGN.md](docs/DESIGN.md) - Complete technical design document
- [docs/HACKATHON.md](docs/HACKATHON.md) - 2-week hackathon implementation plan
- [api/proto/spire_mgmt.proto](api/proto/spire_mgmt.proto) - gRPC API definitions

## Quick Start

```bash
# Start development environment
docker-compose up -d

# Build all binaries
make build

# Run tests
make test
```

For complete instructions, see the documentation files.
