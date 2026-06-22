# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added (Phase 10 - Enterprise Hardening & Production Excellence)
- **Security Bug Fixes** (`internal/web/controller/security.go`, `internal/web/service/security.go`):
  - Fixed `RefreshIPAccessGlobal()` no-op bug: now properly calls `middleware.RefreshIPAccess()` to reload IP access rules into the global IPAccessManager without requiring a panel restart (import cycle avoided via function variable `service.RefreshIPAccessFunc` wired in `web.go`)
  - Fixed 2FA backup codes: `BackupCode` model added to `internal/database/model/ip_access.go` with bcrypt-hashed storage; `generateBackupCodes()` now persists hashed codes to the database before returning plaintext codes; `verifyBackupCode()` checks code against stored hashes and marks consumed codes; database migration includes `backup_codes` table
  - `BackupCode` model registered in `internal/database/db.go` AutoMigrate
- **Kubernetes Manifests** (`deploy/k8s/`): Complete K8s deployment package with `namespace.yaml`, `configmap.yaml`, `secret.yaml`, `pvc.yaml` (data + certs), `deployment.yaml` (health probes, resource limits, securityContext), `service.yaml` (ClusterIP), `ingress.yaml` (nginx + cert-manager annotations), `hpa.yaml` (CPU 70% / memory 80%, 1-5 replicas), and `kustomization.yaml` entry point
- **Installation Documentation** (`docs/installation/`): Complete bare-metal installation guide (`bare-metal.md`), Docker setup guide (`docker.md`), and Kubernetes deployment guide (`kubernetes.md`)
- **Configuration Documentation** (`docs/configuration/`): Environment variables reference (`environment.md`) and security configuration guide (`security.md`)
- **API Documentation** (`docs/api/`): Security API reference (`security.md`), Monitoring API reference (`monitoring.md`), RBAC API reference (`rbac.md`), and Backup API reference (`backup.md`)
- **API Routes Documentation Sync** (`frontend/src/pages/api-docs/endpoints.ts`): Added 29 missing routes covering Protocol Ecosystem (11 endpoints), Standalone Services (5), Transport Wrappers (4), Audit Log (2), and RBAC (6) sections

### Added (Phase 9 - Production Hardening & Release v1.0.0)
- **CLI Protocol Management** (`main.go`, `internal/protocol/cli.go`): New `x-ui protocol` subcommand with `list`, `start`, `stop`, `restart`, `status`, and `health` subcommands. Manage all 13 protocols from the command line without the web UI.
- **CLI Integration in main.go**: New `runProtocolCommand()` initializes the protocol ecosystem and dispatches CLI commands. The `protocol` case in the switch statement handles `x-ui protocol ...` from the command line.
- **CLI Test Suite** (`internal/protocol/cli_test.go`): 10 tests covering CLI initialization, list output, start/stop/restart lifecycle, status details, health checks, error handling for unknown commands, missing IDs, and unknown protocols.
- **Integration Tests** (`internal/protocol/integration_test.go`): 6 integration tests covering full protocol lifecycle (list → start → status → health → restart → stop → list), standalone service CLI operations, wrapper CLI operations, unhealthy protocol detection, and restart-all scenarios.
- **Wrapper Package Tests** (`internal/protocol/wrapper/wrapper_test.go`): 12 tests covering all 5 wrapper constructors, lifecycle (start/stop/restart), info metadata, config/apply config, wrap config, supported protocols matrix, and port defaults.
- **CLI Documentation** (`docs/cli.md`): Complete CLI reference with all commands, options, protocol IDs, environment variables, and sample output.
- **Protocol Ecosystem Documentation** (`docs/protocols/ecosystem.md`): Comprehensive guide to all 13 protocols with configuration details, use cases, and management instructions.
- **API Reference** (`docs/api/protocols.md`): REST API documentation for protocol management endpoints with request/response examples and error codes.
- **Phase 9 Architecture Document** (`docs/architecture/phase9-release.md`): Implementation plan and architecture overview for the release phase.

### Added (Phase 8 - Protocol Ecosystem Finalization & Integration)
- **XrayBaseProtocol Adapter** (`internal/protocol/xray/`): Bridges the unified protocol registry with existing Xray inbound management. Each Xray-native protocol (VMess, VLESS, Trojan, Shadowsocks, Hysteria) is now represented as a `BaseProtocol` in the registry, enabling unified lifecycle management (start/stop/restart/status) through a single API.
- **Manager.Initialize() Implementation**: The previously empty stub now registers all 13 protocols into the global registry via pluggable factory functions registered during package `init()`:
  - 5 Xray base protocols via `internal/protocol/xray/xray.go`
  - 3 standalone services (OpenVPN, WireGuard, Dropbear) via `internal/protocol/standalone/`
  - 5 transport wrappers (WebSocket, TLS, HTTP/2, gRPC, Naive) via `internal/protocol/wrapper/`
- **Factory Registration Pattern**: `RegisterBaseFactory()`, `RegisterStandaloneFactory()`, `RegisterWrapperFactory()` allow implementation packages to self-register without creating import cycles. Init functions in standalone, wrapper, and xray packages auto-register at startup.
- **Global Registry Singleton**: `protocol.Global()` now returns a properly initialized singleton. `protocol.InitGlobal()` is called at server startup in `web.go` before monitor initialization. `protocol.GlobalManager()` provides access to the initialized Manager.
- **ProtocolService Refactored**: `NewProtocolService()` now uses `protocol.Global()` singleton instead of creating a private local registry, ensuring all system components share the same registry instance.
- **Naive Proxy Implementation** (`internal/protocol/wrapper/naive.go`): Full process lifecycle management for the Naive proxy (HTTP CONNECT tunnel). Supports binary detection, config generation, start/stop/restart, health checks, and auth/TLS configuration.
- **Port Conflict Validation**: `Manager.Initialize()` validates that no two registered protocols share the same port, providing early detection of configuration conflicts.
- **Comprehensive Test Suite**: 20 unit tests for `internal/protocol/` covering all protocol definitions, registry lifecycle, manager operations, category listing, wrapper discovery, singleton behavior, and initialization idempotency.

### Added (Phase 6 - Standalone Services)
- **OpenVPN Enhancement**: Full PKI management (CA, server cert, client cert generation using ECDSA P-256), tls-crypt key generation, .ovpn client config export, client CRUD (Add/Remove/List/GetConfig)
- **WireGuard Enhancement**: Proper Curve25519 key generation via `golang.org/x/crypto/curve25519` (replaced fake byte-based keys), preshared key generation, peer management with JSON-based peer store, client config export, `wg syncconf` live reload
- **Dropbear Enhancement**: RSA 2048-bit host key generation, SSH user management (add/remove/list users), public key injection via authorized_keys
- **Port Conflict Detection**: `checkPortAvailable()` runs before starting any standalone service to prevent port binding conflicts
- **Resource Limiting**: `ApplyResourceLimits()` creates systemd drop-in files for MemoryMax, CPUQuota, TasksMax, LimitNOFILE; `ApplyUlimit()` for nice values
- **Auto-Install Script**: `deploy/install-standalone-services.sh` detects OS and installs OpenVPN/WireGuard/Dropbear via apt/yum/dnf/apk
- **PKI Store**: `PKIStore` struct with file-based certificate management (GenerateCA, GenerateServerCert, GenerateClientCert, ReadCA, ListClients, RemoveClient)
- **32 Unit Tests**: Full test coverage for PKI, OpenVPN, WireGuard, Dropbear, and ResourceLimits packages

### Added (Phase 4 - Monitoring & Alerting)
- Prometheus `/metrics` endpoint exposing protocol health, traffic, connections, users, errors, and uptime
- Kubernetes `healthz` and `readyz` endpoints for container orchestration
- Alert rules persistence: rules are now loaded from the database into the rule engine at startup
- Telegram and Email notifiers for the alert rule engine (wired via NotifierFactory)
- Auto-recovery: protocols automatically restart when unhealthy and an auto-recovery rule is enabled
- Audit logging system: `audit_logs` table, `AuditService`, `AuditMiddleware` for tracking all state-changing API calls
- Audit log API endpoints: `GET /panel/api/audit/logs`, `POST /panel/api/audit/logs/clear`
- Frontend Monitoring page with 4 tabs: Health Status, Metrics, Alert Rules, Alert History
- Monitoring page added to sidebar navigation
- `MonitorController.listRules` now returns `dbId` for proper DB record linkage

### Fixed
- Fixed `UptimeSeconds` calculation in `MetricsCollector.Collect` (was `time.Since(time.Now())` which always returned 0; now tracks real protocol start times with `protocolStartTimes sync.Map`)
- Fixed alert rule DB operations: create/update/delete now use correct integer DB IDs instead of mismatched string IDs

### Changed
- `monitor.InitGlobal` now calls `loadAlertRulesFromDB()` to hydrate the rule engine from persistent storage
- `AlertRule` struct now includes `DBID` field for proper database record linkage
- `protocol_health_job.go` now evaluates alert rules, records alert history, and triggers auto-recovery
- `internal/database/db.go`: added `&model.AuditLog{}` to the migration list
- `internal/web/controller/api.go`: registered `AuditMiddleware` for all API routes and `NewAuditController`
- `internal/web/web.go`: wired Telegram/Email notifiers and NotifierFactory in the startup sequence
