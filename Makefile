.PHONY: help build test

help:
	@echo 'Available targets:'
	@echo '  build      - Build all binaries'
	@echo '  test       - Run tests'
	@echo '  proto      - Generate protobuf code'

build:
	@mkdir -p bin
	go build -o bin/api-server ./cmd/api-server
	go build -o bin/site-agent ./cmd/site-agent

test:
	go test -v ./...

proto:
	protoc --go_out=. --go-grpc_out=. api/proto/*.proto
