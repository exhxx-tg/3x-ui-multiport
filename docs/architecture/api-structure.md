# API Structure (internal/web/)

## Directory Layout
```
internal/web/
├── web.go              # Main server, routing setup, Gin engine
├── controller/         # Request handlers by domain
├── middleware/         # Auth, session, CSRF, headers
├── service/            # Business logic layer
│   ├── panel/          # Panel-specific services (users, inbounds)
│   ├── tgbot/          # Telegram bot notifications
│   └── email/          # Email service
├── session/            # Session management
├── websocket/          # Real-time WebSocket (traffic stats)
├── job/                # Cron jobs (traffic monitoring, expiry)
├── locale/             # i18n translation
├── network/            # Network utils
├── runtime/            # Runtime state
├── global/             # Global state / singletons
└── entity/             # Request/response DTOs
```

## Routing Pattern
Routes are registered in `web.go` via Gin:
- `GET /panel/...` - Admin panel pages
- `GET /api/...` - REST API endpoints
- `POST /api/...` - REST API mutations
- `GET /sub/...` - Subscription links
- `GET /assets/...` - Static frontend assets
- WebSocket endpoints for real-time stats

## Middleware Chain
1. Session middleware (cookie store)
2. CSRF protection
3. Authentication check
4. Authorization (RBAC)
5. Rate limiting
6. Request logging
7. Gzip compression

## API Controllers

### Current endpoints:
- `GET /api/inbounds` - List all inbounds
- `GET /api/inbounds/:id` - Get inbound details
- `POST /api/inbounds` - Create inbound
- `PUT /api/inbounds/:id` - Update inbound
- `DELETE /api/inbounds/:id` - Delete inbound
- `POST /api/inbounds/:id/reset` - Reset traffic stats
- `GET /api/inbounds/clientIps/:email` - Get client IPs
- `GET /api/inbounds/online` - Online users

### User/Admin:
- `POST /api/login` - Login
- `GET /api/logout` - Logout
- `GET /api/settings` - Get settings
- `PUT /api/settings` - Update settings

### System:
- `GET /api/status` - System status
- `POST /api/restart` - Restart panel
- `POST /api/restartXray` - Restart Xray

## Key Patterns
- Gin framework with middleware
- DTO validation in entity/ package
- Service layer for business logic
- Xray API calls for config management
