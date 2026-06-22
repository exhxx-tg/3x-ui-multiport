[English](/README.md) | [فارسی](/README.fa_IR.md) | [العربية](/README.ar_EG.md) | [中文](/README.zh_CN.md) | [Español](/README.es_ES.md) | [Русский](/README.ru_RU.md) | [Türkçe](/README.tr_TR.md)

<p align="center">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="./media/3x-ui-dark.png">
    <img alt="3x-ui-multiport" src="./media/3x-ui-light.png">
  </picture>
</p>

<p align="center">
  <a href="https://github.com/exhxx-tg/3x-ui-multiport/releases"><img src="https://img.shields.io/github/v/release/exhxx-tg/3x-ui-multiport" alt="Release"></a>
  <a href="https://github.com/exhxx-tg/3x-ui-multiport/actions"><img src="https://img.shields.io/github/actions/workflow/status/exhxx-tg/3x-ui-multiport/release.yml.svg" alt="Build"></a>
  <a href="#"><img src="https://img.shields.io/github/go-mod/go-version/exhxx-tg/3x-ui-multiport.svg" alt="GO Version"></a>
  <a href="https://github.com/exhxx-tg/3x-ui-multiport/releases/latest"><img src="https://img.shields.io/github/downloads/exhxx-tg/3x-ui-multiport/total.svg" alt="Downloads"></a>
  <a href="https://www.gnu.org/licenses/gpl-3.0.en.html"><img src="https://img.shields.io/badge/license-GPL%20V3-blue.svg?longCache=true" alt="License"></a>
  <a href="https://pkg.go.dev/github.com/exhxx-tg/3x-ui-multiport"><img src="https://pkg.go.dev/badge/github.com/exhxx-tg/3x-ui-multiport.svg" alt="Go Reference"></a>
  <a href="https://goreportcard.com/report/github.com/exhxx-tg/3x-ui-multiport"><img src="https://goreportcard.com/badge/github.com/exhxx-tg/3x-ui-multiport" alt="Go Report Card"></a>
</p>

**3X-UI-Multiport (X-UI PRO)** is an advanced, open-source web control panel for managing a unified **13-Protocol Ecosystem** — combining [Xray-core](https://github.com/XTLS/Xray-core), standalone VPN services, and transport wrappers into a single management interface. It provides a clean, multi-language dashboard for deploying, configuring, monitoring, and securing proxy and VPN protocols — from a single VPS to multi-node deployments.

Built as a superset of the 3X-UI project, X-UI PRO adds 8 additional protocols, enterprise-grade security, comprehensive monitoring, and production-hardened DevOps tooling.

> [!IMPORTANT]
> This project is intended for personal use only. Please do not use it for illegal purposes or in a production environment.

## 13 Protocol Ecosystem

### 🔹 Base Protocols (Xray-native)
| Protocol | Description | Source |
|---|---|---|
| **VMess** | Socks5-like proxy with encryption | Xray-core |
| **VLESS** | Lightweight VMess — no encryption overhead | Xray-core |
| **Trojan** | TLS-based protocol mimicking HTTPS | Xray-core |
| **Shadowsocks** | Simple socks5 + stream cipher encryption | Xray-core |
| **Hysteria** | UDP-based protocol — speed optimized | Xray-core |

### 🔹 Standalone Services
| Service | Description | Source |
|---|---|---|
| **OpenVPN** | Industry-standard VPN (TCP/UDP) | OpenVPN |
| **WireGuard** | Modern kernel-based VPN | WireGuard |
| **Dropbear** | Lightweight SSH server | Dropbear |

### 🔹 Transport Wrappers
| Wrapper | Description | Compatible With |
|---|---|---|
| **WebSocket** | HTTP WebSocket tunnel | VMess, VLESS, SS, Trojan |
| **TLS/HTTPS** | TLS encrypted transport | VMess, VLESS, SS, Trojan |
| **HTTP/2** | HTTP/2 multiplexed transport | VLESS, Trojan |
| **gRPC** | gRPC protocol wrapping | VLESS, Trojan |
| **Naive** | HTTP CONNECT tunnel | All base protocols |

## Features

- **13 Protocol Ecosystem** — 5 Base (VMess, VLESS, Trojan, Shadowsocks, Hysteria) + 3 Standalone (OpenVPN, WireGuard, Dropbear) + 5 Transport Wrappers (WebSocket, TLS, HTTP/2, gRPC, Naive)
- **Modern transports & security** — TCP (Raw), mKCP, WebSocket, gRPC, HTTPUpgrade, and XHTTP, secured with TLS, XTLS, and REALITY.
- **Fallbacks** — serve multiple protocols on a single port (e.g. VLESS and Trojan on 443) using Xray's fallback support.
- **Per-client management** — traffic quotas, expiry dates, IP limits, live online status, and one-click share links, QR codes, and subscriptions.
- **Traffic statistics** — per inbound, per client, and per outbound, with reset controls.
- **Multi-node support** — manage and scale across multiple servers from a single panel.
- **Outbound & routing** — WARP, NordVPN, custom routing rules, load balancers, and outbound proxy chaining.
- **Built-in subscription server** with multiple output formats and [custom page templates](docs/custom-subscription-templates.md).
- **Telegram bot** for remote monitoring and management.
- **Enterprise Security** — RBAC, 2FA, Audit Log, IP Access Control, Rate Limiting, API Tokens.
- **Comprehensive Monitoring** — Health checks per protocol, metrics collection, alert rules, Prometheus exporter.
- **Performance Optimizations** — Worker pools, in-memory caching, DB connection pooling, gzip compression.
- **CLI Tool** — Full protocol management from terminal (start, stop, restart, status, health).
- **Kubernetes Support** — Complete K8s manifests (deployment, service, ingress, HPA, PVC, configmap).
- **RESTful API** with in-panel Swagger documentation.
- **Flexible storage** — SQLite (default) or PostgreSQL.
- **13 UI languages** with dark and light themes.
- **Fail2ban integration** for enforcing per-client IP limits.

## Quick Start — كود تثبيت واحد يجهز كل شي

### 🐧 Linux (Bare Metal) — كلاسيكي
```bash
bash <(curl -Ls https://raw.githubusercontent.com/exhxx-tg/3x-ui-multiport/main/install.sh)
```
> بعد التثبيت: افتح `http://your-server-ip:2053`، أو شغّل `x-ui` للإدارة.

### 🐳 Docker — كود واحد يشتغل في أي مكان
```bash
docker run -d --name x-ui --restart unless-stopped --cap-add=NET_ADMIN --cap-add=NET_RAW -p 2053:2053 -v x-ui-db:/etc/x-ui ghcr.io/exhxx-tg/3x-ui-multiport:latest
```
> افتح المتصفح على `http://your-server-ip:2053`

### ☸️ Kubernetes
```bash
kubectl apply -k https://github.com/exhxx-tg/3x-ui-multiport/deploy/k8s
```

### ⚡ تثبيت بدون تدخل (للسحابات)
```bash
XUI_NONINTERACTIVE=1 bash <(curl -Ls https://raw.githubusercontent.com/exhxx-tg/3x-ui-multiport/main/install.sh)
```

لمزيد من التفاصيل: [docs/](docs/)

### Unattended install & cloud images

The installer also runs **non-interactively** for cloud-init and golden images.
Set `XUI_NONINTERACTIVE=1` (or pipe with no TTY) and it installs end-to-end with
zero prompts, generating random credentials and writing them to
`/etc/x-ui/install-result.env`. See [`deploy/`](deploy/) for:

- [Cloud-init user-data](deploy/cloud-init/) — unattended install on any cloud (Hetzner/AWS/DO/Vultr/GCP/Azure/Oracle)
- [Packer golden image](deploy/packer/) — build an AWS EC2 AMI + qcow2 (amd64/arm64) with per-instance credentials generated on first boot
- [Amazon Lightsail](deploy/lightsail/) — launch script + reusable snapshot builder
- [AWS Marketplace checklist](deploy/marketplace/aws/)

## Supported Platforms

**Operating systems:** Ubuntu, Debian, Armbian, Fedora, CentOS, RHEL, AlmaLinux, Rocky Linux, Oracle Linux, Amazon Linux, Virtuozzo, Arch, Manjaro, Parch, openSUSE (Tumbleweed / Leap), Alpine, and Windows.

**Architectures:** `amd64` · `386` · `arm64` (aarch64) · `armv7` · `armv6` · `armv5` · `s390x`.

## Database Options

X-UI PRO supports two backends, chosen during the install:

- **SQLite** (default) — a single file at `/etc/x-ui/x-ui.db`. Zero setup, ideal for small and medium deployments.
- **PostgreSQL** — recommended for high client counts or multi-node setups. The installer can install PostgreSQL locally for you, or accept a DSN to an existing server.

At runtime the backend is selected via environment variables (the installer writes these to `/etc/default/x-ui` for you):

```
XUI_DB_TYPE=postgres
XUI_DB_DSN=postgres://xui:password@127.0.0.1:5432/xui?sslmode=disable
```

### Migrating an existing SQLite install to PostgreSQL

```bash
x-ui migrate-db --dsn "postgres://xui:password@127.0.0.1:5432/xui?sslmode=disable"
# then set XUI_DB_TYPE and XUI_DB_DSN in /etc/default/x-ui and restart:
systemctl restart x-ui
```

The source SQLite file is left untouched; remove it manually once you have verified the new backend.

### Docker

The default `docker compose up -d` keeps using SQLite. To run with the bundled PostgreSQL service, uncomment the two `XUI_DB_*` env lines in `docker-compose.yml` and start with the profile:

```bash
docker compose --profile postgres up -d
```

The image bundles Fail2ban (enabled by default) to enforce per-client **IP limits**. Fail2ban bans offenders with `iptables`, which requires the `NET_ADMIN` capability. `docker-compose.yml` already grants it via `cap_add`; if you start the container with `docker run` instead, add the capabilities yourself, otherwise bans are logged but never applied:

```bash
docker run -d --cap-add=NET_ADMIN --cap-add=NET_RAW ... ghcr.io/exhxx-tg/3x-ui-multiport
```

## Environment Variables

| Variable | Description | Default |
| --- | --- | --- |
| `XUI_DB_TYPE` | Database backend: `sqlite` or `postgres` | `sqlite` |
| `XUI_DB_DSN` | PostgreSQL connection string (when `XUI_DB_TYPE=postgres`) | — |
| `XUI_DB_FOLDER` | Directory for the SQLite database file | `/etc/x-ui` |
| `XUI_DB_MAX_OPEN_CONNS` | Maximum open connections (PostgreSQL pool) | — |
| `XUI_DB_MAX_IDLE_CONNS` | Maximum idle connections (PostgreSQL pool) | — |
| `XUI_INIT_WEB_BASE_PATH` | The initial URI path for the web panel | `/` |
| `XUI_ENABLE_FAIL2BAN` | Enable Fail2ban-based IP-limit enforcement | `true` |
| `XUI_LOG_LEVEL` | Log verbosity (`debug`, `info`, `warning`, `error`) | `info` |
| `XUI_DEBUG` | Enable debug mode | `false` |
| `XUI_WORKER_POOL` | Worker pool size for async jobs | `16` |
| `XUI_CACHE_TTL_SEC` | Cache TTL in seconds | `30` |
| `XUI_CACHE_DISABLE` | Disable in-memory caching | `false` |

## Supported Languages

The panel UI is available in 13 languages:

English · فارسی · العربية · 中文（简体） · 中文（繁體） · Español · Русский · Українська · Türkçe · Tiếng Việt · 日本語 · Bahasa Indonesia · Português (Brasil)

## Documentation

Full documentation is available in the [docs/](docs/) directory:

- **Installation**: [Bare Metal](docs/installation/bare-metal.md), [Docker](docs/installation/docker.md), [Kubernetes](docs/installation/kubernetes.md)
- **Configuration**: [Environment](docs/configuration/environment.md), [Security](docs/configuration/security.md)
- **API**: [Security](docs/api/security.md), [Monitoring](docs/api/monitoring.md), [RBAC](docs/api/rbac.md), [Backup](docs/api/backup.md)
- **Architecture**: [Phase 7 - Performance](docs/architecture/phase7-performance.md), [Phase 8 - Integration](docs/architecture/phase8-integration.md), [Phase 9 - Release](docs/architecture/phase9-release.md), [Phase 10 - Enterprise](docs/architecture/phase10-enterprise.md)
- **Protocols**: [Ecosystem Overview](docs/protocols/ecosystem.md)

## CLI Usage

```
x-ui run              Run web panel
x-ui setting          Configure panel settings (port, user, TLS, Telegram)
x-ui protocol         Manage 13-protocol ecosystem (list, start, stop, restart, status, health)
x-ui migrate          Migrate from old x-ui
x-ui migrate-db       SQLite <-> .dump / PostgreSQL migration
x-ui cert             Manage SSL certificates
```

## Contributing

Contributions are welcome. Please read the [Contributing Guide](/CONTRIBUTING.md) before opening an issue or pull request.

## A Special Thanks to

- [alireza0](https://github.com/alireza0/)
- [MHSanaei](https://github.com/MHSanaei/) — original 3X-UI project

## Acknowledgment

- [Iran v2ray rules](https://github.com/chocolate4u/Iran-v2ray-rules) (License: **GPL-3.0**): _Enhanced v2ray/xray and v2ray/xray-clients routing rules with built-in Iranian domains and a focus on security and adblocking._
- [Russia v2ray rules](https://github.com/runetfreedom/russia-v2ray-rules-dat) (License: **GPL-3.0**): _This repository contains automatically updated V2Ray routing rules based on data on blocked domains and addresses in Russia._

## Community Tools

Tools and integrations built by the community around 3x-ui.

- [terraform-provider-3x-ui](https://github.com/batonogov/terraform-provider-threexui) (License: **MIT**): _Manage inbounds, clients, panel settings, and Xray configuration as code with Terraform / OpenTofu._

## Support project

**If this project is helpful to you, you may wish to give it a**:star2:

## Stargazers over Time

[![Stargazers over time](https://starchart.cc/exhxx-tg/3x-ui-multiport.svg?variant=adaptive)](https://starchart.cc/exhxx-tg/3x-ui-multiport)
