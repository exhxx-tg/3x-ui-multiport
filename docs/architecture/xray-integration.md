# Xray Integration (internal/xray/)

## Current Architecture
- Xray runs as subprocess managed by the panel
- Configuration generated from database inbound models
- Communication via Xray gRPC API (port 10085)
- Traffic stats collected via Xray API polling

## Key Files
| File | Purpose |
|------|---------|
| `process.go` | Xray process lifecycle (start/stop/restart) |
| `config.go` | Xray JSON config generation |
| `inbound.go` | Inbound configuration builders |
| `api.go` | gRPC API client for Xray |
| `traffic.go` | Traffic data collection |
| `client_traffic.go` | Per-client traffic tracking |
| `hot_diff.go` | Config diff to minimize restarts |
| `log_writer.go` | Capture Xray stdout logs |
| `online.go` | Online user detection |

## Config Flow
1. Load inbound configs from database (GORM models)
2. Build Xray JSON config with inbounds, routing, policies
3. Generate config file on disk
4. Validate config syntax
5. Apply hot-diff (if only specific inbounds changed, avoid full restart)
6. Restart Xray with new config (or hot update)

## Supported Protocols (Current)
| Protocol | Status | Notes |
|----------|--------|-------|
| VMess | ✅ Full | Stream settings, TLS, WS, gRPC, etc. |
| VLESS | ✅ Full | Full XTLS support, flow control |
| Trojan | ✅ Full | TLS required, fallback support |
| Shadowsocks | ✅ Full | Multiple ciphers, AEAD |
| MTProto | ✅ Full | Telegram proxy via mtg subprocess |
| Hysteria | ❌ Missing | Not implemented yet |
| OpenVPN | ❌ Missing | Standalone service needed |
| WireGuard | ❌ Missing | Standalone service needed |
| Dropbear | ❌ Missing | Standalone service needed |

## Transport Wrappers (Current)
| Wrapper | Status | Notes |
|---------|--------|-------|
| WebSocket | ✅ Built-in | Xray native transport |
| TLS/HTTPS | ✅ Built-in | Xray native transport |
| HTTP/2 | ✅ Built-in | Xray native transport |
| gRPC | ✅ Built-in | Xray native transport |
| Naive | ❌ Missing | External proxy needed |

## Limitations
- Only Xray-native protocols (VMess, VLESS, Trojan, SS)
- No standalone service orchestration
- No unified protocol management interface
- Protocol-specific logic scattered across handlers
- No resource isolation between protocols
