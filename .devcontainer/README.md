# HUNT Chat-API Development Environment

This directory contains configuration files for the HUNT Chat-API development environment using VS Code Dev Containers.

## 🚀 Quick Start

1. **Prerequisites**:

   - Visual Studio Code
   - Docker Desktop
   - Dev Containers extension for VS Code

2. **Open in Dev Container**:

   - Open VS Code in the project root
   - Press `F1` and select "Dev Containers: Reopen in Container"
   - Wait for the container to build and setup to complete

3. **Start Development**:
   ```bash
   make dev    # Run with live reloading
   make run    # Run normally
   make test   # Run tests
   make check  # Run all checks (fmt, vet, lint, test)
   ```

## 📦 What's Included

### Development Tools

- **Go 1.21** - Latest Go runtime
- **Azure CLI** - Azure command-line interface
- **Azure Developer CLI** - Modern Azure development experience
- **Docker** - Container runtime
- **Git** - Version control
- **GitHub CLI** - GitHub command-line interface

### VS Code Extensions

- **Go** - Go language support
- **Azure Tools** - Azure development tools
- **Docker** - Docker support
- **GitHub Copilot** - AI-powered code completion
- **REST Client** - API testing

### Go Tools

- **gopls** - Go language server
- **goimports** - Import management
- **golangci-lint** - Comprehensive linter
- **dlv** - Delve debugger
- **air** - Live reloading

## 🔧 Configuration

### Environment Variables

Set these in your `.env` file:

```env
AZURE_OPENAI_ENDPOINT=https://your-openai-resource.openai.azure.com/
AZURE_OPENAI_MODEL=gpt-4
PORT=8080
ENVIRONMENT=development
```

### Azure Authentication

The container mounts your local `~/.azure` directory for seamless Azure authentication.

## 🎯 Available Commands

### Development

```bash
make dev      # Run with live reloading
make run      # Run normally
make build    # Build application
make test     # Run tests
make check    # Run all checks (fmt, vet, lint, test)
```

### Azure

```bash
make azure-login  # Login to Azure
make azure-init   # Initialize Azure resources
make deploy       # Deploy to Azure
```

### Docker

```bash
make docker-build  # Build Docker image
make docker-run    # Run Docker container
```

## 🌟 Features

- **Live Reloading**: Automatic restart on code changes
- **Integrated Testing**: Run tests with `make test`
- **Code Quality**: Automatic formatting and linting
- **Azure Integration**: Seamless Azure development
- **Port Forwarding**: Access your app at `http://localhost:8080`

## 🔄 Customization

Edit `.devcontainer/devcontainer.json` to:

- Add new VS Code extensions
- Modify container features
- Change environment variables
- Update port forwarding

## 📋 Troubleshooting

### Container Won't Start

1. Ensure Docker Desktop is running
2. Check if ports 8080, 8081, 3000 are available
3. Rebuild container: `Dev Containers: Rebuild Container`

### Azure Authentication Issues

1. Run `az login` in the container terminal
2. Ensure your Azure credentials are properly configured
3. Check if the Azure CLI is properly installed

### Go Module Issues

1. Run `go mod tidy` to clean dependencies
2. Delete `go.sum` and run `go mod download`
3. Ensure you're using Go 1.21 or later
