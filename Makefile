.PHONY: help build test proto clean docker-build setup

help:
	@echo 'Available targets:'
	@echo '  build        - Build all binaries'
	@echo '  test         - Run tests'
	@echo '  proto        - Generate protobuf code'
	@echo '  clean        - Clean build artifacts'
	@echo '  docker-build - Build Docker images'
	@echo '  setup        - Setup development environment'

# Build binaries
build: proto
	@mkdir -p bin
	go build -o bin/api-server ./cmd/api-server
	go build -o bin/site-agent ./cmd/site-agent

# Run tests
test:
	go test -v ./...

# Generate protobuf code
proto:
	@mkdir -p api/proto/gen
	protoc \
		--go_out=. \
		--go_opt=paths=source_relative \
		--go-grpc_out=. \
		--go-grpc_opt=paths=source_relative \
		-I. \
		-I$(shell go env GOPATH)/src \
		api/proto/spire_mgmt.proto

# Install protoc plugins
proto-deps:
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# Clean build artifacts
clean:
	rm -rf bin/
	rm -rf api/proto/gen/

# Build Docker images
docker-build:
	docker build -t spire-mgmt-api:alpha -f deploy/docker/Dockerfile.api .
	docker build -t site-agent:alpha -f deploy/docker/Dockerfile.agent .

# Build Docker images for minikube
docker-build-minikube:
	eval $$(minikube docker-env) && \
	docker build -t spire-mgmt-api:alpha -f deploy/docker/Dockerfile.api . && \
	docker build -t site-agent:alpha -f deploy/docker/Dockerfile.agent .

# Run locally with docker-compose
run-local:
	docker-compose up -d

# Stop local services
stop-local:
	docker-compose down

# Tidy dependencies
tidy:
	go mod tidy

# Setup development environment
setup: proto-deps tidy proto build
	@echo "Development environment ready!"

# Run API server locally (requires MySQL running)
run-api:
	DB_HOST=localhost DB_PORT=3306 DB_USER=root DB_PASSWORD=demo-password DB_NAME=spire_mgmt \
	GRPC_PORT=8080 \
	go run ./cmd/api-server

# Deploy to minikube
deploy: docker-build-minikube
	./scripts/setup-all.sh

# Reset demo data
reset-demo:
	./scripts/reset-demo.sh
