# CLI Reference

X-UI PRO provides a command-line interface for managing the panel and all 13 protocols.

## Usage

```
x-ui <command> [options]
```

## Commands

### `run`

Start the web panel.

```
x-ui run
```

### `setting`

View or modify panel settings.

```
x-ui setting [options]

Options:
  --show                  Display current settings
  --reset                 Reset all settings to defaults
  --port <number>         Set panel port
  --username <string>     Set login username
  --password <string>     Set login password
  --webBasePath <string>  Set web base path
  --listenIP <string>     Set listen IP address
  --resetTwoFactor        Reset 2FA
  --webCert <path>        TLS certificate public key
  --webCertKey <path>     TLS certificate private key
  --tgbottoken <string>   Telegram bot token
  --tgbotchatid <string>  Telegram chat ID
  --tgbotRuntime <string> Telegram notification cron schedule
  --enabletgbot           Enable Telegram bot
```

### `protocol`

Manage the 13-protocol ecosystem.

```
x-ui protocol <subcommand> [id]

Subcommands:
  list               List all protocols with status
  start    <id>      Start a protocol
  stop     <id>      Stop a protocol
  restart  <id>      Restart a protocol
  status   <id>      Show detailed status of a protocol
  health   <id>      Run a health check on a protocol
```

#### Protocol IDs

| ID | Name | Category |
|----|------|----------|
| `vmess` | VMess | Base |
| `vless` | VLESS | Base |
| `trojan` | Trojan | Base |
| `shadowsocks` | Shadowsocks | Base |
| `hysteria` | Hysteria | Base |
| `openvpn` | OpenVPN | Standalone |
| `wireguard` | WireGuard | Standalone |
| `dropbear` | Dropbear | Standalone |
| `websocket` | WebSocket Wrapper | Transport |
| `tls` | TLS/HTTPS Wrapper | Transport |
| `http2` | HTTP/2 Wrapper | Transport |
| `grpc` | gRPC Wrapper | Transport |
| `naive` | Naive Wrapper | Transport |

#### Examples

```bash
# List all protocols
x-ui protocol list

# Start Xray VMess proxy
x-ui protocol start vmess

# Stop OpenVPN service
x-ui protocol stop openvpn

# Restart WireGuard
x-ui protocol restart wireguard

# Check detailed status of Trojan
x-ui protocol status trojan

# Run health check on Shadowsocks
x-ui protocol health shadowsocks
```

Sample output of `x-ui protocol list`:

```
ID            NAME                  CATEGORY      STATUS    HEALTHY   PORT
--            ----                  --------      ------    -------   ----
vmess         VMess                 base          running   true      10086
vless         VLESS                 base          running   true      10087
trojan        Trojan                base          stopped   false     10088
shadowsocks   Shadowsocks           base          running   true      10089
hysteria      Hysteria              base          stopped   false     -
openvpn       OpenVPN               standalone    running   true      1194
wireguard     WireGuard             standalone    running   true      51820
dropbear      Dropbear              standalone    stopped   false     2222
websocket     WebSocket Wrapper     wrapper       running   true      80
tls           TLS/HTTPS Wrapper     wrapper       running   true      443
http2         HTTP/2 Wrapper        wrapper       running   true      443
grpc          gRPC Wrapper          wrapper       stopped   false     443
naive         Naive Wrapper         wrapper       running   true      8080
```

### `migrate`

Migrate from an older X-UI version.

```
x-ui migrate
```

### `migrate-db`

Database migration and backup utilities.

```
x-ui migrate-db [options]

Options:
  --dump <file>         Export SQLite database to SQL dump
  --restore <file>      Restore SQLite database from SQL dump
  --out <path>          Destination path for restore
  --dsn <postgres-dsn>  Migrate to PostgreSQL
  --src <path>          Source SQLite path (default: x-ui.db)
```

### `cert`

Manage TLS certificates.

```
x-ui cert [options]

Options:
  --webCert <path>      Set certificate public key
  --webCertKey <path>   Set certificate private key
  --reset               Clear certificate settings
```

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `XUI_DEBUG` | `false` | Enable debug mode |
| `XUI_PORT` | - | Override web panel port |
| `XUI_LOG_LEVEL` | `info` | Log level (debug, info, warning, error) |
| `XUI_DB_FOLDER` | `/etc/x-ui` | Database directory |
| `XUI_DB_TYPE` | `sqlite` | Database type (sqlite, postgres) |
| `XUI_DB_DSN` | - | PostgreSQL connection string |
| `XUI_BIN_FOLDER` | `bin` | Binary storage directory |
| `XUI_LOG_FOLDER` | `/var/log/x-ui` | Log directory |
| `XUI_SKIP_HSTS` | `false` | Skip HSTS headers |
