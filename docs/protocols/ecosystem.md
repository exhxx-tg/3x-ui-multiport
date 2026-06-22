# 13 Protocol Ecosystem

X-UI PRO supports a comprehensive ecosystem of **13 protocols** organized into
three categories. This document provides an overview of each protocol, its
configuration, and how to manage it.

## Category 1: Base Protocols (Xray-native)

Five protocols built into Xray-core. They handle actual proxy traffic with
various trade-offs between speed, security, and obfuscation.

### 1. VMess

- **ID:** `vmess`
- **Type:** Encrypted proxy protocol
- **Port:** Configurable (default: 10086)
- **Features:**
  - Built-in encryption (AEAD)
  - Multi-user support
  - Load balancing
  - Metadata obfuscation
- **Use case:** General-purpose circumvention with strong encryption
- **Source:** [github.com/XTLS/Xray-core](https://github.com/XTLS/Xray-core)

### 2. VLESS

- **ID:** `vless`
- **Type:** Lightweight proxy protocol (no encryption)
- **Port:** Configurable (default: 10087)
- **Features:**
  - Zero encryption overhead
  - XTLS Vision flow
  - Fallback routing
  - REALITY support
- **Use case:** High-speed scenarios where TLS is handled at transport level
- **Source:** [github.com/XTLS/Xray-core](https://github.com/XTLS/Xray-core)

### 3. Trojan

- **ID:** `trojan`
- **Type:** TLS-based proxy
- **Port:** Configurable (default: 10088)
- **Features:**
  - Mimics HTTPS traffic
  - Password authentication
  - Fallback to web server
  - Hard to detect
- **Use case:** Firewall evasion, ISP blocking bypass
- **Source:** [github.com/XTLS/Xray-core](https://github.com/XTLS/Xray-core)

### 4. Shadowsocks

- **ID:** `shadowsocks`
- **Type:** Simple SOCKS5 with stream cipher
- **Port:** Configurable (default: 10089)
- **Features:**
  - Very low resource usage
  - Multiple cipher support (AEAD, 2022)
  - Single/multi-user modes
  - UDP support
- **Use case:** Lightweight circumvention, mobile devices
- **Source:** [github.com/XTLS/Xray-core](https://github.com/XTLS/Xray-core)

### 5. Hysteria

- **ID:** `hysteria`
- **Type:** UDP-based protocol
- **Port:** Configurable (default: 10090)
- **Features:**
  - QUIC-based transport
  - Speed optimization for poor networks
  - Built-in bandwidth estimation
  - Brutal congestion control
- **Use case:** Mobile networks, high-latency environments
- **Source:** [github.com/HysteriaNet/Hysteria](https://github.com/HysteriaNet/Hysteria)

## Category 2: Standalone Services

Three independent services that run as separate systemd daemons, each with their
own configuration and lifecycle.

### 6. OpenVPN

- **ID:** `openvpn`
- **Type:** Industry-standard VPN
- **Port:** 1194 (UDP/TCP)
- **Features:**
  - Full PKI management (CA, server, client certs)
  - tls-crypt key protection
  - .ovpn client config export
  - Client CRUD (add/remove/list/get config)
  - ECDSA P-256 certificates
- **Requirements:** `openvpn` package installed
- **Use case:** Secure tunneling, corporate VPN
- **Source:** [github.com/OpenVPN/openvpn](https://github.com/OpenVPN/openvpn)

### 7. WireGuard

- **ID:** `wireguard`
- **Type:** Modern kernel-based VPN
- **Port:** 51820 (UDP)
- **Features:**
  - Curve25519 key generation
  - Preshared key support
  - Peer management
  - `wg syncconf` live reload
  - Subnet allocation
- **Requirements:** `wireguard` kernel module + `wg-quick` or `wireguard-go`
- **Use case:** High-speed VPN, mobile-first
- **Source:** [github.com/WireGuard/wireguard-go](https://github.com/WireGuard/wireguard-go)

### 8. Dropbear

- **ID:** `dropbear`
- **Type:** Lightweight SSH server
- **Port:** 2222 (TCP)
- **Features:**
  - RSA 2048-bit host key generation
  - SSH user management (add/remove/list)
  - Public key injection via authorized_keys
  - Lower resource usage than OpenSSH
- **Requirements:** `dropbear` package installed
- **Use case:** Lightweight SSH access for embedded systems
- **Source:** [github.com/mkj/dropbear](https://github.com/mkj/dropbear)

## Category 3: Transport Wrappers

Five transport layer wrappers that obfuscate or tunnel base protocol traffic
through different protocols.

### 9. WebSocket Wrapper

- **ID:** `websocket`
- **Port:** 80
- **Supported protocols:** VMess, VLESS, Trojan, Shadowsocks
- **Use case:** Firewall evasion through HTTP ports
- **Config:** `path` (default: `/ws`), `host`

### 10. TLS/HTTPS Wrapper

- **ID:** `tls`
- **Port:** 443
- **Supported protocols:** VMess, VLESS, Trojan, Shadowsocks
- **Use case:** Encrypted transport on HTTPS port
- **Config:** `certFile`, `keyFile`

### 11. HTTP/2 Wrapper

- **ID:** `http2`
- **Port:** 443
- **Supported protocols:** VLESS, Trojan
- **Use case:** Multiplexed connections via HTTP/2
- **Config:** `path` (default: `/h2`), `host`

### 12. gRPC Wrapper

- **ID:** `grpc`
- **Port:** 443
- **Supported protocols:** VLESS, Trojan
- **Use case:** Microservice-style connections
- **Config:** `serviceName` (default: `grpc`), `multiMode`

### 13. Naive Wrapper

- **ID:** `naive`
- **Port:** 8080
- **Supported protocols:** VMess, VLESS, Trojan, Shadowsocks, Hysteria
- **Use case:** Corporate proxy bypass via HTTP CONNECT
- **Config:** `proxyType`, `auth`
- **Source:** [github.com/klzgrad/naiveproxy](https://github.com/klzgrad/naiveproxy)

## Management

### CLI

All protocols are managed via the `x-ui protocol` CLI:

```bash
# List all protocols
x-ui protocol list

# Start a protocol
x-ui protocol start <id>

# Stop a protocol
x-ui protocol stop <id>

# Check status
x-ui protocol status <id>

# Health check
x-ui protocol health <id>
```

### API

Protocols are also accessible via REST API:

```
GET    /panel/api/protocols/detailed        - List all protocols with status
POST   /panel/api/protocols/{id}/start      - Start a protocol
POST   /panel/api/protocols/{id}/stop       - Stop a protocol
POST   /panel/api/protocols/{id}/restart    - Restart a protocol
GET    /panel/api/protocols/{id}/health     - Health check
```

### Web UI

Navigate to **Protocols** in the sidebar to see all 13 protocols with their
current status, health, and start/stop/restart controls.

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│                   Protocol Registry                       │
│               (internal/protocol/)                        │
│  ┌──────────┐  ┌───────────┐  ┌──────────────────────┐  │
│  │   Xray    │  │ Standalone │  │ Transport Wrappers   │  │
│  │ 5 Base    │  │ 3 Services │  │ 5 Wrappers           │  │
│  └──────────┘  └───────────┘  └──────────────────────┘  │
└────────────────────────┬────────────────────────────────┘
                         │
          ┌──────────────┼──────────────┐
          │              │              │
     ┌────▼───┐    ┌─────▼────┐   ┌────▼────┐
     │  Xray  │    │  System  │   │   CLI   │
     │  Core  │    │  Daemons │   │  Commands│
     └────────┘    └──────────┘   └─────────┘
```
