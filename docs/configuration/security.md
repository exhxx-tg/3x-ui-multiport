# Security Configuration

## Overview

X-UI PRO provides enterprise-grade security features accessible through the Security page in the web UI (`/panel/security`) or via API.

## Authentication

### Password Policy
- Minimum password length: 8 characters
- Bcrypt hashing (12+ rounds)
- Rate limiting: 5 failed attempts = 15-minute lockout

### Two-Factor Authentication (2FA)
- TOTP-based (Google Authenticator, Authy, etc.)
- Backup codes: 10 codes generated on setup
- Backup codes are bcrypt-hashed and stored in the database
- Each backup code can be used only once

### API Tokens
- Bearer token authentication for programmatic access
- Tokens are SHA-256 hashed (plaintext shown once on creation)
- Tokens can be enabled/disabled and deleted
- Rate limited per token

## Access Control

### IP Access Rules

Control which IPs can access the panel:

```json
{
  "type": "allow",
  "cidr": "10.0.0.0/8",
  "remark": "Internal network"
}
```

- **Allow rules**: Whitelist specific IPs/CIDRs
- **Block rules**: Blacklist specific IPs/CIDRs
- Rules are evaluated by priority (lower number = higher priority)
- When any allow rule exists, access defaults to deny

### Rate Limiting
- 100 requests/minute per IP (configurable)
- 1000 requests/hour per user
- Bypass for health check endpoints

## Session Management

- JWT-based sessions (15-minute access tokens, 7-day refresh tokens)
- Active sessions visible in Security dashboard
- Remote session revocation
- Force logout all sessions

## Audit Logging

All state-changing operations are logged:
- Login/logout attempts
- Configuration changes
- Protocol start/stop
- Backup/restore operations
- User management
- API token operations

## Security Headers

| Header | Value |
|--------|-------|
| `X-Content-Type-Options` | `nosniff` |
| `X-Frame-Options` | `DENY` |
| `X-XSS-Protection` | `1; mode=block` |
| `Strict-Transport-Security` | `max-age=31536000` |
| `Content-Security-Policy` | `default-src 'self'` |
| `Referrer-Policy` | `no-referrer` |

## TLS Configuration

### Certificate Management

Three options:

1. **Let's Encrypt** (auto-renewal)
   - Configure domain in Settings
   - Port 80 must be reachable for ACME challenge
   - Certificates auto-renew every 60 days

2. **Custom Certificate**
   - Upload PEM files via Security page
   - Supports ECDSA and RSA keys

3. **Self-Signed** (development only)
   - Generate via Security page
   - Not suitable for production

## Best Practices

1. **Change default credentials** immediately
2. **Enable 2FA** for admin accounts
3. **Use API tokens** for automation (not passwords)
4. **Configure IP access rules** for known networks
5. **Regularly review audit logs**
6. **Enable TLS** in production
7. **Use strong JWT secrets** (64+ chars)
8. **Keep backups** of certificate files
