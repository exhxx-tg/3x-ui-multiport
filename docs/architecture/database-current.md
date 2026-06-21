# Current Database Schema

## Database Engine
- **Default:** SQLite (via GORM)
- **Optional:** PostgreSQL (via `migrate-db --dsn`)
- **ORM:** GORM v2 with auto-migration

## Model Files (internal/database/model/)

| File | Models |
|------|--------|
| `model.go` | Inbound, Client, ClientTraffic, Setting, User |
| `client_global_traffic.go` | ClientGlobalTraffic |
| `node_client_ip.go` | NodeClientIP |
| `node_client_traffic.go` | NodeClientTraffic |
| `host_test.go` | Host model tests |
| `model_mtproto_test.go` | MTProto model tests |

## Core Tables

### inbound
| Column | Type | Description |
|--------|------|-------------|
| id | INTEGER PK | Auto-increment |
| user_id | INTEGER FK | Owner |
| up | INTEGER | Upload traffic (bytes) |
| down | INTEGER | Download traffic (bytes) |
| total | INTEGER | Traffic total limit (bytes) |
| remark | TEXT | Human-readable name |
| enable | BOOL | Active/inactive |
| expiry_time | INTEGER | Unix timestamp |
| client_stats | BOOL | Track per-client stats |
| port | INTEGER | Listen port |
| protocol | TEXT | vmess/vless/trojan/shadow-tls |
| settings | TEXT | JSON client configs |
| stream_settings | TEXT | JSON transport config |
| sniffing | TEXT | JSON sniffing config |
| tag | TEXT | Xray inbound tag |
| allocated_ip | TEXT | Allocated IP (if any) |
| listen | TEXT | Bind address |

### client
| Column | Type | Description |
|--------|------|-------------|
| id | INTEGER PK | Auto-increment |
| inbound_id | INTEGER FK | Parent inbound |
| email | TEXT | Unique client identifier |
| up | INTEGER | Upload traffic (bytes) |
| down | INTEGER | Download traffic (bytes) |
| total | INTEGER | Traffic limit (bytes) |
| expiry_time | INTEGER | Unix timestamp |
| enable | BOOL | Active/inactive |
| tg_id | TEXT | Telegram ID |
| sub_id | TEXT | Subscription ID |
| flow | TEXT | XTLS flow control |

### setting
| Column | Type | Description |
|--------|------|-------------|
| id | INTEGER PK | Auto-increment |
| key | TEXT UNIQUE | Setting key |
| value | TEXT | Setting value |

### user
| Column | Type | Description |
|--------|------|-------------|
| id | INTEGER PK | Auto-increment |
| username | TEXT | Login username |
| password | TEXT | Bcrypt hash |

### client_global_traffic
| Column | Type | Description |
|--------|------|-------------|
| id | INTEGER PK | Auto-increment |
| inbound_id | INTEGER FK | Parent inbound |
| email | TEXT | Client email |
| up | INTEGER | Global upload (bytes) |
| down | INTEGER | Global download (bytes) |

### node_client_ip / node_client_traffic
Per-node tracking for multi-server setups.

## Missing Tables (Need to Add)
| Table | Purpose |
|-------|---------|
| protocols | Protocol metadata registry |
| protocol_configs | Per-protocol detailed configuration |
| protocol_stats | Per-protocol traffic and connection stats |
| services | Standalone service management (OpenVPN, WG, etc.) |
| api_tokens | API key authentication |
| audit_logs | Security audit trail |
| backups | Backup records |
| certificates | SSL/TLS certificate management |
| permissions | RBAC roles and permissions |
