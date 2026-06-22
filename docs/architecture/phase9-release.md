# Phase 9: Production Hardening & Release v1.0.0

## Overview

Phase 9 is the final production hardening phase that transforms the 13-protocol
ecosystem into a polished, production-ready v1.0.0 release. It adds CLI management
commands, comprehensive tests, complete documentation, frontend polish, and the
release pipeline.

## Key Deliverables

| # | Deliverable | Module | Priority |
|---|-------------|--------|----------|
| 1 | CLI protocol management commands | `main.go` | HIGH |
| 2 | Integration & E2E test suite | `internal/protocol/`, `test/` | HIGH |
| 3 | Complete documentation | `docs/` | MEDIUM |
| 4 | Frontend protocol detail pages | `frontend/src/pages/protocols/` | MEDIUM |
| 5 | Release v1.0.0 pipeline | `.github/workflows/` | MEDIUM |

## Architecture

```
CLI (main.go)
  │
  ├── protocol list        → ProtocolManager.ListProtocols()
  ├── protocol start <id>  → ProtocolManager.StartProtocol(id)
  ├── protocol stop <id>   → ProtocolManager.StopProtocol(id)
  ├── protocol restart <id> → ProtocolManager.RestartProtocol(id)
  ├── protocol status <id>  → ProtocolManager.GetProtocolStatus(id)
  └── protocol health <id>  → Standalone.HealthCheck() / Base.Status()
```

## Implementation Plan

### Phase 9.1: CLI Protocol Management (Days 1-3)

Add `protocol` subcommand to `main.go`:
- `x-ui protocol list` — Table of all 13 protocols with ID, name, category, status
- `x-ui protocol start <id>` — Start a protocol by ID
- `x-ui protocol stop <id>` — Stop a protocol by ID
- `x-ui protocol restart <id>` — Restart a protocol by ID
- `x-ui protocol status <id>` — Show detailed status of one protocol
- `x-ui protocol health <id>` — Run health check on a protocol

### Phase 9.2: Comprehensive Tests (Days 4-7)

- Integration tests for CLI commands
- E2E tests for protocol lifecycle via API
- Edge case tests (port conflicts, missing binaries, invalid configs)

### Phase 9.3: Documentation Completion (Days 8-11)

- Complete API reference docs
- Protocol configuration guides
- Installation & deployment docs
- CLI usage guide

### Phase 9.4: Frontend Polish (Days 12-14)

- Protocol detail pages with per-protocol settings
- Better monitoring UX with real-time protocol health
- Protocol configuration forms

### Phase 9.5: Release v1.0.0 (Days 15-19)

- Version bump to v1.0.0
- CHANGELOG finalization
- Release workflow in GitHub Actions
- Docker image tagging
