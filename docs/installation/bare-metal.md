# Bare Metal Installation

## Prerequisites

- Linux (Ubuntu 22.04+, Debian 12+, CentOS 8+)
- Root or sudo access
- Go 1.21+ (for building from source) OR pre-built binary
- Port 2053 available (default web UI)

## Quick Install (One-liner)

```bash
bash <(curl -Ls https://raw.githubusercontent.com/exhxx-tg/3x-ui-multiport/main/install.sh)
```

After installation, access: `http://<server-ip>:2053`

Default credentials: `admin` / `admin`

## Manual Installation

### 1. Download the Binary

```bash
# Download latest release
wget https://github.com/exhxx-tg/3x-ui-multiport/releases/latest/download/x-ui-linux-amd64.tar.gz

# Extract
tar -xzf x-ui-linux-amd64.tar.gz
cd x-ui
```

### 2. Install as System Service

```bash
# Copy binary
cp x-ui /usr/local/bin/
chmod +x /usr/local/bin/x-ui

# Create directories
mkdir -p /etc/x-ui
mkdir -p /var/log/x-ui

# Install systemd service
cp x-ui.service /etc/systemd/system/
systemctl daemon-reload
systemctl enable x-ui
systemctl start x-ui
```

### 3. Verify Installation

```bash
systemctl status x-ui
curl http://localhost:2053/panel/api/server/status
```

## Build from Source

```bash
git clone https://github.com/exhxx-tg/3x-ui-multiport.git
cd 3x-ui
go mod download
cd web && npm install && npm run build && cd ..
CGO_ENABLED=1 go build -ldflags="-s -w" -o x-ui main.go
```

## Standalone Services Installation

For OpenVPN, WireGuard, and Dropbear support:

```bash
bash <(curl -Ls https://raw.githubusercontent.com/exhxx-tg/3x-ui-multiport/main/deploy/install-standalone-services.sh)
```

## Post-Installation

1. Change default password via Web UI (Settings > Panel)
2. Configure TLS certificate (Settings > Certificate)
3. Set up firewall rules
4. Configure automatic backups (Settings > Backup)
