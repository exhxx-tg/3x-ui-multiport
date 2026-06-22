# Docker Installation

## Prerequisites

- Docker Engine 24+
- Docker Compose 2+ (optional)
- Port 2053 available

## Quick Start (one-liner)

```bash
docker run -d \
  --name x-ui-pro \
  --restart unless-stopped \
  -p 2053:2053 \
  -v x-ui-data:/etc/x-ui \
  hsanaeii/3x-ui:latest
```

Access: `http://localhost:2053`

Default credentials: `admin` / `admin`

## Docker Compose (Recommended)

```yaml
version: '3.8'
services:
  3xui:
    image: hsanaeii/3x-ui:latest
    container_name: 3x-ui
    hostname: 3x-ui
    volumes:
      - $PWD/db/:/etc/x-ui/
      - $PWD/cert/:/root/cert/
    environment:
      XRAY_VMESS_AEAD_FORCED: "false"
      PGSSLMODE: "disable"
    ports:
      - "2053:2053"
    cap_add:
      - NET_ADMIN
      - NET_RAW
    restart: unless-stopped
```

Save as `docker-compose.yml` and run:

```bash
docker compose up -d
```

## With PostgreSQL (Optional)

```yaml
version: '3.8'
services:
  3xui:
    image: hsanaeii/3x-ui:latest
    ports:
      - "2053:2053"
    environment:
      XRAY_VMESS_AEAD_FORCED: "false"
      DB_TYPE: postgres
      PG_HOST: postgres
      PG_PORT: 5432
      PG_USER: xui
      PG_PASS: xui_pass
      PG_NAME: xui_db
      PGSSLMODE: disable
    cap_add:
      - NET_ADMIN
      - NET_RAW
    restart: unless-stopped
    profiles:
      - postgres

  postgres:
    image: postgres:16-alpine
    environment:
      POSTGRES_USER: xui
      POSTGRES_PASSWORD: xui_pass
      POSTGRES_DB: xui_db
    volumes:
      - pgdata:/var/lib/postgresql/data
    profiles:
      - postgres

volumes:
  pgdata:
```

Run with: `docker compose --profile postgres up -d`

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `TZ` | `UTC` | Timezone |
| `LOG_LEVEL` | `info` | Log level (debug/info/warning/error) |
| `XRAY_VMESS_AEAD_FORCED` | `false` | Force VMess AEAD authentication |
| `DB_TYPE` | `sqlite` | Database type (sqlite/postgres) |
| `PG_HOST` | - | PostgreSQL host |
| `PG_PORT` | - | PostgreSQL port |
| `PG_USER` | - | PostgreSQL user |
| `PG_PASS` | - | PostgreSQL password |
| `PG_NAME` | - | PostgreSQL database name |
| `PGSSLMODE` | `disable` | PostgreSQL SSL mode |

## Docker Compose (Development)

```bash
docker compose -f docker-compose.dev.yml up -d
```

This exposes additional ports (8080 API, 10085 Xray API) and maps logs to `./logs/`.

## Building the Image

```bash
docker build -t x-ui-pro:latest .
docker build -t x-ui-pro:latest --build-arg TARGETARCH=arm64 .
```
