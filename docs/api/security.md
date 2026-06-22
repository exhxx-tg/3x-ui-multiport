# Security API

Base path: `/panel/api/security`

## Overview

Get security overview data.

```http
GET /panel/api/security/overview
```

Response:
```json
{
  "success": true,
  "obj": {
    "totalSessions": 3,
    "loginAttempts24h": 42,
    "failedLogins24h": 5,
    "activeRules": 2,
    "blockedIPs": 1,
    "allowedIPs": 1,
    "expiringCerts": 0,
    "recentEvents": [],
    "recentLoginAttempts": []
  }
}
```

## Login Attempts

List login attempt history:

```http
GET /panel/api/security/login-attempts?offset=0&limit=50
```

## Security Events

List security events:

```http
GET /panel/api/security/events?offset=0&limit=50
```

## IP Access Control

### List Rules
```http
GET /panel/api/security/ip-access
```

### Create Rule
```http
POST /panel/api/security/ip-access
Content-Type: application/json

{
  "type": "allow",
  "cidr": "10.0.0.0/8",
  "remark": "Internal network",
  "priority": 10
}
```

### Update Rule
```http
POST /panel/api/security/ip-access/:id/update
```

### Delete Rule
```http
POST /panel/api/security/ip-access/:id/delete
```

## Sessions

### List Active Sessions
```http
GET /panel/api/security/sessions
```

### Revoke Session
```http
POST /panel/api/security/sessions/revoke/:id
```

### Revoke All Sessions
```http
POST /panel/api/security/sessions/revoke-all
```

## Two-Factor Authentication

### Generate Backup Codes
```http
POST /panel/api/security/2fa/generate-backup-codes
```

Response:
```json
{
  "success": true,
  "obj": {
    "codes": ["ABCD1234...", ...],
    "count": 10
  }
}
```

### Verify Backup Code
```http
POST /panel/api/security/2fa/verify-backup-code
Content-Type: application/json

{
  "code": "ABCD1234..."
}
```
