#!/bin/bash

# HUNT Chat-API Development Environment Setup Script
echo "🚀 Setting up HUNT Chat-API development environment..."

# Update package list
sudo apt-get update

# Install additional tools
sudo apt-get install -y \
    curl \
    wget \
    git \
    make \
    build-essential \
    ca-certificates \
    software-properties-common \
    jq \
    tree \
    vim \
    htop

# Install Go tools
echo "📦 Installing Go tools..."
go install golang.org/x/tools/cmd/goimports@latest
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
go install github.com/go-delve/delve/cmd/dlv@latest

# Install air for hot reloading
go install github.com/cosmtrek/air@latest
curl -fsSL https://aka.ms/install-azd.sh | bash

# Install Go tools
echo "🔧 Installing Go development tools..."
go install -v github.com/ramya-rao-a/go-outline@latest
go install -v github.com/cweill/gotests/gotests@latest
go install -v github.com/fatih/gomodifytags@latest
go install -v github.com/josharian/impl@latest
go install -v github.com/haya14busa/goplay/cmd/goplay@latest
go install -v github.com/go-delve/delve/cmd/dlv@latest
go install -v honnef.co/go/tools/cmd/staticcheck@latest
go install -v golang.org/x/tools/cmd/gopls@latest
go install -v golang.org/x/tools/cmd/goimports@latest
go install -v github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Install project dependencies
echo "📚 Installing project dependencies..."
go mod tidy
go mod download

# Install air for live reloading (optional)
echo "🔄 Installing air for live reloading..."
go install github.com/cosmtrek/air@latest

# Create .env file from example if it doesn't exist
if [ ! -f ".env" ]; then
    echo "📝 Creating .env file from example..."
    cp .env.example .env
fi

# Set up git hooks (optional)
echo "🪝 Setting up git hooks..."
if [ -d ".git" ]; then
    # Create pre-commit hook
    mkdir -p .git/hooks
    cat > .git/hooks/pre-commit << 'EOF'
#!/bin/bash
# Run go fmt
go fmt ./...
# Run go vet
go vet ./...
# Run tests
go test ./...
EOF
    chmod +x .git/hooks/pre-commit
fi

# Create air configuration for live reloading
echo "🌬️ Creating air configuration..."
cat > .air.toml << 'EOF'
root = "."
testdata_dir = "testdata"
tmp_dir = "tmp"

[build]
  args_bin = []
  bin = "./tmp/main"
  cmd = "go build -o ./tmp/main ./cmd/server"
  delay = 1000
  exclude_dir = ["assets", "tmp", "vendor", "testdata"]
  exclude_file = []
  exclude_regex = ["_test.go"]
  exclude_unchanged = false
  follow_symlink = false
  full_bin = ""
  include_dir = []
  include_ext = ["go", "tpl", "tmpl", "html"]
  include_file = []
  kill_delay = "0s"
  log = "build-errors.log"
  poll = false
  poll_interval = 0
  rerun = false
  rerun_delay = 500
  send_interrupt = false
  stop_on_root = false

[color]
  app = ""
  build = "yellow"
  main = "magenta"
  runner = "green"
  watcher = "cyan"

[log]
  main_only = false
  time = false

[misc]
  clean_on_exit = false

[screen]
  clear_on_rebuild = false
  keep_scroll = true
EOF

# Create Makefile for common tasks
echo "📋 Creating Makefile..."
cat > Makefile << 'EOF'
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

# Azure login
azure-login:
	az login
EOF

echo "✅ Development environment setup complete!"
echo ""
echo "🎯 Quick start commands:"
echo "  make run       - Run the application"
echo "  make dev       - Run with live reloading"
echo "  make test      - Run tests"
echo "  make check     - Run all checks (fmt, vet, lint, test)"
echo "  make build     - Build the application"
echo ""
echo "🔧 Azure commands:"
echo "  make azure-login - Login to Azure"
echo "  make azure-init  - Initialize Azure resources"
echo "  make deploy      - Deploy to Azure"
echo ""
echo "🌟 Ready to start developing HUNT Chat-API!"
