# Current Architecture Diagram

```
┌─────────────────────────────────────────────────────────────┐
│                    Web Browser (SPA)                        │
│                React 18 + TypeScript + Vite                 │
└──────────────────────────┬──────────────────────────────────┘
                           │ HTTP/HTTPS (port 2053)
┌──────────────────────────▼──────────────────────────────────┐
│                 Gin Web Framework (Go)                      │
│   internal/web/web.go                                       │
│                                                             │
│   ┌──────────────────────────────────────────────────────┐  │
│   │                Middleware Stack                       │  │
│   │  Session → CSRF → Auth → RBAC → RateLimit → Logger   │  │
│   └──────────────────────────────────────────────────────┘  │
│                                                             │
│   ┌─────────────────┐  ┌──────────────────┐  ┌───────────┐ │
│   │  REST API        │  │  Subscription    │  │ WebSocket │ │
│   │  controllers/    │  │  Server (sub/)   │  │ stats     │ │
│   └────────┬────────┘  └────────┬─────────┘  └───────────┘ │
└────────────┼─────────────────────┼──────────────────────────┘
             │                     │
┌────────────▼─────────────────────▼──────────────────────────┐
│                    GORM Database Layer                      │
│                                                             │
│   ┌──────────────────────────────────────────────────────┐  │
│   │                  SQLite / PostgreSQL                  │  │
│   │  Tables: inbound, client, setting, user, traffic     │  │
│   └──────────────────────────────────────────────────────┘  │
└──────────────────────────────────────────────────────────────┘
             │
┌────────────▼──────────────────────────────────────────────┐
│                    Xray Core Process                       │
│                                                           │
│   Config Generation (config.go) → JSON file               │
│   Process Management (process.go) → start/stop/restart    │
│   gRPC API (api.go) → stats, online users                │
│   Hot Diff (hot_diff.go) → minimize restarts              │
│                                                           │
│   Protocols: VMess, VLESS, Trojan, Shadowsocks, MTProto   │
│   Transports: WS, TLS, HTTP/2, gRPC, TCP                 │
└───────────────────────────────────────────────────────────┘
```

## Data Flow

```
User Request → Nginx/Reverse Proxy → Gin Server → Middleware
    → Controller → Service → Database/Xray API → Response
```

## Component Interactions

1. **Web Server** (`internal/web/web.go`)
   - Creates Gin engine with routes
   - Embeds frontend dist via Go embed
   - Starts Xray process
   - Schedules cron jobs (traffic monitoring, expiry checks)

2. **Controllers** (`internal/web/controller/`)
   - Handle HTTP requests
   - Validate input (entity/ DTOs)
   - Call service layer
   - Return JSON responses

3. **Services** (`internal/web/service/`)
   - Business logic layer
   - Database operations via GORM
   - Xray config management
   - Telegram bot notifications

4. **Xray Package** (`internal/xray/`)
   - Xray process lifecycle
   - Config generation/validation
   - Traffic data collection via gRPC API
   - Hot reload (diff-based config updates)

5. **Database** (`internal/database/`)
   - GORM models and migrations
   - SQLite (default) or PostgreSQL support
   - Dump/restore utilities
