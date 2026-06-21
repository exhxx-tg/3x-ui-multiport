# X-UI PRO Documentation

Welcome to X-UI PRO documentation.

## Quick Links
- [Installation Guide](installation/)
- [Configuration Guide](configuration/)
- [API Reference](api/)
- [Architecture](architecture/)
- [Development Guide](development/)
- [Protocols](protocols/)

## About X-UI PRO

X-UI PRO is an enhanced fork of [3X-UI](https://github.com/MHSanaei/3x-ui) with support for **13 protocols** and enterprise-grade features.

## Features
- 5 Base Protocols (Xray-native): VMess, VLESS, Trojan, Shadowsocks, Hysteria
- 3 Standalone Services: OpenVPN, WireGuard, Dropbear
- 5 Transport Wrappers: WebSocket, TLS/HTTPS, HTTP/2, gRPC, Naive
- Complete Monitoring System with real-time stats
- Enterprise Security (RBAC, 2FA, Audit, Rate Limiting)
- Easy Installation (Docker one-liner or bare metal)

## Quick Start

### Docker (Recommended)
```bash
docker run -d \
  --name x-ui-pro \
  -p 2053:2053 \
  -p 8080:8080 \
  yourregistry/x-ui-pro:latest
```

Then access: http://localhost:2053

### Bare Metal
See [Installation Guide](installation/)
