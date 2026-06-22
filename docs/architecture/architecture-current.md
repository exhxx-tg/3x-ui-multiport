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
   - Registers monitor notifiers (Telegram, Email) and health/readyz/metrics endpoints

2. **Controllers** (`internal/web/controller/`)
   - Handle HTTP requests
   - Validate input (entity/ DTOs)
   - Call service layer
   - Return JSON responses
   - MonitorController: health checks, metrics, alert rules, alert history
   - AuditController: audit log querying and management

3. **Services** (`internal/web/service/`)
   - Business logic layer
   - Database operations via GORM
   - Xray config management
   - Telegram bot notifications
   - AuditService: records and queries audit log entries

4. **Xray Package** (`internal/xray/`)
   - Xray process lifecycle
   - Config generation/validation
   - Traffic data collection via gRPC API
   - Hot reload (diff-based config updates)

5. **Database** (`internal/database/`)
   - GORM models and migrations
   - SQLite (default) or PostgreSQL support
   - Dump/restore utilities
   - Tables: users, inbounds, clients, settings, nodes, hosts, alert_rules, alert_history, protocol_metrics, audit_logs

6. **Monitoring Package** (`internal/monitor/`)
   - HealthChecker: protocol health checks with optional auto-recovery
   - MetricsCollector: in-memory protocol metrics with traffic/connection tracking
   - RuleEngine: alert rule evaluation with cooldown, condition matching, multi-channel notification
   - HistoryStore: alert event persistence to database
   - PrometheusExporter: exposes metrics in Prometheus text format on `/metrics`
   - Notifiers: LogNotifier, WebhookNotifier, TelegramNotifier, EmailNotifier
   - Cron jobs: protocol health check (30s), metrics collection (60s)
