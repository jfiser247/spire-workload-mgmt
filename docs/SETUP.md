# Project Setup Guide

## Prerequisites

- Go 1.21+
- Docker & Docker Compose
- kubectl (for Kubernetes deployment)
- MySQL 8.0+ (or use Docker Compose)

## Local Development Setup

### 1. Extract the Project

```bash
tar -xzf spire-workload-mgmt.tar.gz
cd spire-workload-mgmt
```

### 2. Install Dependencies

```bash
go mod download
```

### 3. Start Development Environment

```bash
# Start MySQL and other services
docker-compose up -d

# Wait for MySQL to be ready
sleep 10

# Apply database schema
mysql -h 127.0.0.1 -u root -proot spire_mgmt < deploy/kubernetes/schema.sql
```

### 4. Build and Run

```bash
# Build binaries
make build

# Run API server
./bin/api-server

# In another terminal, run site agent
SITE_ID=site-1 SPIRE_SERVER=localhost:8081 ./bin/site-agent
```

## Next Steps

1. Review docs/DESIGN.md for architecture details
2. Review docs/HACKATHON.md for implementation roadmap
3. Set up your GitHub repository and push the code
4. Start implementing according to the hackathon plan

## Troubleshooting

### Database Connection Issues

If you can't connect to MySQL:
- Check if Docker container is running: `docker ps`
- Verify port 3306 is not in use: `lsof -i :3306`
- Check MySQL logs: `docker logs <container-id>`

### Build Issues

If `make build` fails:
- Ensure Go 1.21+ is installed: `go version`
- Run `go mod tidy` to fix dependencies
- Check for missing protoc: `protoc --version`

## Development Workflow

1. Make changes to code
2. Run tests: `make test`
3. Build: `make build`  
4. Test locally with Docker Compose
5. Commit and push to your repository

