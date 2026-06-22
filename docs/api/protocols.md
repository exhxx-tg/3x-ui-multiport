# Protocol API Reference

## Overview

The Protocol API provides programmatic access to manage the 13-protocol
ecosystem. All endpoints are under the `/panel/api/protocols/` base path.

## Authentication

All API requests require authentication via:
- Session cookie (browser)
- API token (`X-API-Key` header)

## Endpoints

### List All Protocols

```
GET /panel/api/protocols/detailed
```

Returns all 13 protocols with current status and health information.

**Response:**
```json
{
  "success": true,
  "protocols": [
    {
      "id": "vmess",
      "name": "VMess",
      "category": "base",
      "description": "Socks5-like proxy with encryption",
      "xrayNative": true,
      "status": "running",
      "healthy": true,
      "port": 10086
    }
  ]
}
```

### Start Protocol

```
POST /panel/api/protocols/{id}/start
```

Start a protocol by its ID.

**Parameters:**
- `id` - Protocol ID (vmess, vless, trojan, shadowsocks, hysteria, openvpn, wireguard, dropbear, websocket, tls, http2, grpc, naive)

**Response:**
```json
{
  "success": true,
  "message": "Protocol started successfully"
}
```

### Stop Protocol

```
POST /panel/api/protocols/{id}/stop
```

Stop a running protocol.

**Response:**
```json
{
  "success": true,
  "message": "Protocol stopped successfully"
}
```

### Restart Protocol

```
POST /panel/api/protocols/{id}/restart
```

Restart a protocol (stop then start).

**Response:**
```json
{
  "success": true,
  "message": "Protocol restarted successfully"
}
```

### Protocol Status

```
GET /panel/api/protocols/{id}/status
```

Get detailed status of a specific protocol.

**Response:**
```json
{
  "success": true,
  "protocol": {
    "id": "openvpn",
    "name": "OpenVPN",
    "category": "standalone",
    "status": "running",
    "healthy": true,
    "port": 1194,
    "installed": true,
    "serviceName": "openvpn",
    "config": {}
  }
}
```

### Health Check

```
GET /panel/api/protocols/{id}/health
```

Run a health check on a specific protocol.

**Response:**
```json
{
  "success": true,
  "healthy": true,
  "status": "running",
  "latency": "5ms"
}
```

## Status Values

| Status | Description |
|--------|-------------|
| `running` | Protocol is active and handling traffic |
| `stopped` | Protocol is inactive |
| `error` | Protocol encountered an error |
| `installing` | Protocol is being set up |
| `unknown` | Status cannot be determined |

## Error Codes

| HTTP Status | Code | Description |
|-------------|------|-------------|
| 404 | `protocol_not_found` | The specified protocol ID does not exist |
| 409 | `port_conflict` | Port is already in use by another protocol |
| 500 | `start_failed` | Protocol failed to start |
| 500 | `stop_failed` | Protocol failed to stop |
| 503 | `not_installed` | Standalone service binary not found |
