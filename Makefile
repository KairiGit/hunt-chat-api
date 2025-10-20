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

# Initialize system documentation in production Qdrant Cloud (manual execution only)
init-docs-prod:
	@if [ -z "$$CI" ]; then \
		echo "⚠️  本番環境のQdrant Cloudにドキュメントを投入します"; \
		echo "接続先: Qdrant Cloud (AWS US-East-1)"; \
		read -p "続行しますか？ [y/N]: " confirm && [ "$$confirm" = "y" ] || exit 1; \
	else \
		echo "ℹ️  CI環境では init-docs-prod は実行をスキップします"; \
		echo "本番へのドキュメント投入は手動で実行してください"; \
		exit 0; \
	fi
	QDRANT_URL="passthrough:///abf582ca-03a0-4b91-bb17-8671d82bbd53.us-east-1-1.aws.cloud.qdrant.io:6334" \
	QDRANT_API_KEY="eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJhY2Nlc3MiOiJtIn0.dIqS4imHUnoAi6HVwRXhEDxwiuBGl_jdeY_5xIhHuCA" \
	go run scripts/init_system_docs.go

# Initialize docs (auto mode - for CI/CD)
# Set ENABLE_INIT_DOCS=true in Vercel environment variables to enable
init-docs-auto:
	@if [ "$$ENABLE_INIT_DOCS" = "true" ]; then \
		echo "🚀 自動モード: システムドキュメントを投入します"; \
		go run scripts/init_system_docs.go; \
	else \
		echo "ℹ️  ENABLE_INIT_DOCS が true でないため、スキップします"; \
	fi

# Azure login
azure-login:
	az login
