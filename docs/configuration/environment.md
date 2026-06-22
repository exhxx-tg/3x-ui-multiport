# Environment Configuration

## Overview

X-UI PRO is configured through environment variables, CLI flags, and the web UI. This document covers environment variables and configuration files.

## Environment Variables

### Core Settings

| Variable | Default | Description |
|----------|---------|-------------|
| `WEB_PORT` | `2053` | Web UI and API port |
| `DB_PATH` | `./x-ui.db` | SQLite database path |
| `JWT_SECRET` | auto-generated | JWT signing secret |
| `LOG_LEVEL` | `info` | Log level: debug, info, warning, error |
| `LOG_FILE` | - | Log file path (empty = stdout) |
| `TZ` | `UTC` | Server timezone |

### Database

| Variable | Default | Description |
|----------|---------|-------------|
| `DB_TYPE` | `sqlite` | Database type: sqlite, postgres |
| `PG_HOST` | - | PostgreSQL host |
| `PG_PORT` | `5432` | PostgreSQL port |
| `PG_USER` | - | PostgreSQL user |
| `PG_PASS` | - | PostgreSQL password |
| `PG_NAME` | - | PostgreSQL database name |
| `PGSSLMODE` | `disable` | PostgreSQL SSL mode |

### Xray

| Variable | Default | Description |
|----------|---------|-------------|
| `XRAY_VMESS_AEAD_FORCED` | `false` | Force VMess AEAD auth |
| `XRAY_PORT` | `10085` | Xray gRPC API port |

### Performance

| Variable | Default | Description |
|----------|---------|-------------|
| `XUI_CACHE_DISABLE` | `false` | Disable caching |
| `XUI_CACHE_TTL_SEC` | `30` | Default cache TTL in seconds |
| `XUI_WORKER_POOL` | `16` | Async worker pool size |
| `XUI_DB_MAX_OPEN` | `8` (sqlite) / `25` (pg) | Max open DB connections |
| `XUI_DB_MAX_IDLE` | `4` (sqlite) / `25` (pg) | Max idle DB connections |
| `XUI_PPROF_ENABLE` | `false` | Enable pprof debug endpoints |
| `XUI_GZIP_LEVEL` | `1` | Gzip compression level (1-9) |

### Subscriptions

| Variable | Default | Description |
|----------|---------|-------------|
| `SUB_PORT` | `10882` | Subscription server port |
| `SUB_PATH` | `/sub/` | Subscription path prefix |

## Configuration File

Settings can be loaded from a `.env` file in the working directory:

```bash
# .env
WEB_PORT=2053
LOG_LEVEL=debug
JWT_SECRET=your-secure-secret-here
DB_PATH=/etc/x-ui/x-ui.db
```

## CLI Configuration

The panel can also be configured via CLI:

```bash
# Set port
x-ui setting -port 2053

# Set username/password
x-ui setting -username admin -password newpass

# Show current settings
x-ui setting -show
```

## Security

- **JWT_SECRET**: Must be at least 32 characters. Generate with: `openssl rand -base64 48`
- **ADMIN_PASSWORD**: Change immediately after first login
- **TLS**: Always use HTTPS in production. Configure via Settings > Certificate in the web UI
