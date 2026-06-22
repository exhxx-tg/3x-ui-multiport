# Development Guide

## Getting Started

### Prerequisites
- Go 1.21+
- Node.js 18+
- Docker & Docker Compose
- Git
- VS Code (recommended)

### Setup
```bash
git clone https://github.com/YourUsername/3x-ui-pro.git
cd 3x-ui-pro
git remote add upstream https://github.com/exhxx-tg/3x-ui-multiport.git
```

### Build & Run
```bash
go mod download
cd app && npm install && npm run build && cd ..
go build -o x-ui-pro main.go
./x-ui-pro
```

### Docker
```bash
docker compose -f docker-compose.dev.yml up
```

## Project Structure
- `main.go` - Entry point
- `internal/` - Go packages
- `frontend/` - Web UI
- `docs/` - Documentation
- `deploy/` - Deployment scripts

## Git Workflow
See [CONTRIBUTING.md](../CONTRIBUTING.md)
