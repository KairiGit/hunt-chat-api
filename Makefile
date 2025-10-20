.PHONY: build run test clean fmt vet lint dev docker-build docker-run

# Build the application
build:
	go build -o bin/hunt-chat-api cmd/server/main.go

# Run the application
run:
	go run cmd/server/main.go

# Run with live reloading
dev:
	air

# Run tests
test:
	go test -v ./...

# Clean build artifacts
clean:
	rm -rf bin/ tmp/

# Format code
fmt:
	go fmt ./...

# Vet code
vet:
	go vet ./...

# Run linter
lint:
	golangci-lint run

# Run all checks
check: fmt vet lint test

# Install dependencies
deps:
	go mod tidy
	go mod download

# Build Docker image
docker-build:
	docker build -t hunt-chat-api .

# Run Docker container
docker-run:
	docker run -p 8080:8080 hunt-chat-api

# Deploy to Azure
deploy:
	azd deploy

# Initialize Azure resources
azure-init:
	azd init

# Initialize system documentation in vector DB
init-docs:
	go run scripts/init_system_docs.go

# Azure login
azure-login:
	az login
