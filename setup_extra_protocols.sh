#!/usr/bin/env bash
set -Eeuo pipefail

# 3x-ui Multiport Extra Protocols installation engine.
# Installs and wires systemd services for: SSH, Dropbear, Stunnel, SSH-WS,
# UDP Custom (BadVPN), SlowDNS (DNSTT), and Psiphon.
#
# SlowDNS source (required): https://github.com/powermx/dnstt
# Psiphon sources (required): https://github.com/tipsytux/psiphon or https://github.com/thispc/psiphon

export DEBIAN_FRONTEND=noninteractive

EXTRA_DIR="/etc/3x-ui/extra"
BIN_DIR="/usr/local/bin"
LOG_DIR="/var/log/3x-ui-extra"
SSH_PORT="${SSH_PORT:-2222}"
DROPBEAR_PORT="${DROPBEAR_PORT:-2223}"
STUNNEL_PORT="${STUNNEL_PORT:-444}"
SSHWS_PORT="${SSHWS_PORT:-8443}"
UDP_CUSTOM_PORT="${UDP_CUSTOM_PORT:-7300}"
SLOWDNS_PORT="${SLOWDNS_PORT:-5353}"
PSIPHON_PORT="${PSIPHON_PORT:-443}"
SSH_BACKEND_PORT="${SSH_BACKEND_PORT:-22}"

DNSTT_REPO="https://github.com/powermx/dnstt"
PSIPHON_REPOS=("https://github.com/tipsytux/psiphon" "https://github.com/thispc/psiphon")

log() { printf '\033[1;32m[extra-protocols]\033[0m %s\n' "$*"; }
warn() { printf '\033[1;33m[extra-protocols][warn]\033[0m %s\n' "$*" >&2; }
die() { printf '\033[1;31m[extra-protocols][error]\033[0m %s\n' "$*" >&2; exit 1; }

need_root() {
  [[ "${EUID}" -eq 0 ]] || die "Please run as root."
}

detect_pkg_manager() {
  if command -v apt-get >/dev/null 2>&1; then echo apt; return; fi
  if command -v dnf >/dev/null 2>&1; then echo dnf; return; fi
  if command -v yum >/dev/null 2>&1; then echo yum; return; fi
  if command -v pacman >/dev/null 2>&1; then echo pacman; return; fi
  die "Unsupported distribution: apt, dnf, yum, or pacman is required."
}

install_packages() {
  local pm="$1"; shift
  case "$pm" in
    apt) apt-get update -y && apt-get install -y "$@" ;;
    dnf) dnf install -y "$@" ;;
    yum) yum install -y "$@" ;;
    pacman) pacman -Sy --noconfirm "$@" ;;
  esac
}

pkg_bootstrap() {
  local pm="$1"
  case "$pm" in
    apt) install_packages "$pm" ca-certificates curl wget git build-essential golang openssh-server dropbear stunnel4 python3 python3-pip openssl tar gzip ;;
    dnf|yum) install_packages "$pm" ca-certificates curl wget git gcc gcc-c++ make golang openssh-server dropbear stunnel python3 python3-pip openssl tar gzip ;;
    pacman) install_packages "$pm" ca-certificates curl wget git base-devel go openssh dropbear stunnel python python-pip openssl tar gzip ;;
  esac
}

download_with_fallback() {
  local dest="$1"; shift
  local url
  for url in "$@"; do
    log "Downloading $url"
    if curl -fsSL --retry 3 --connect-timeout 12 "$url" -o "$dest"; then
      return 0
    fi
    warn "Download failed: $url"
  done
  return 1
}

clone_with_fallback() {
  local dest="$1"; shift
  rm -rf "$dest"
  local repo
  for repo in "$@"; do
    log "Cloning $repo"
    if git clone --depth=1 "$repo" "$dest"; then
      return 0
    fi
    warn "Clone failed: $repo"
    rm -rf "$dest"
  done
  return 1
}

ensure_dirs() {
  mkdir -p "$EXTRA_DIR" "$BIN_DIR" "$LOG_DIR" /etc/stunnel /etc/ssh-ws /etc/psiphon /etc/dnstt
}

write_service() {
  local name="$1"
  local content="$2"
  printf '%s\n' "$content" > "/etc/systemd/system/${name}.service"
  systemctl daemon-reload
  systemctl enable "${name}.service" >/dev/null 2>&1 || true
}

configure_ssh() {
  log "Configuring OpenSSH on port ${SSH_PORT}"
  mkdir -p /etc/ssh/sshd_config.d
  cat > /etc/ssh/sshd_config.d/99-3x-ui-extra.conf <<EOF_SSH
Port 22
Port ${SSH_PORT}
PasswordAuthentication yes
PubkeyAuthentication yes
PermitTunnel yes
Banner /etc/issue.net
EOF_SSH
  touch /etc/issue.net
  systemctl enable ssh >/dev/null 2>&1 || systemctl enable sshd >/dev/null 2>&1 || true
  systemctl restart ssh >/dev/null 2>&1 || systemctl restart sshd >/dev/null 2>&1 || warn "Could not restart ssh/sshd now."
}

configure_dropbear() {
  log "Configuring Dropbear service on port ${DROPBEAR_PORT}"
  write_service "dropbear-extra" "[Unit]
Description=3x-ui Extra Dropbear SSH
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=/usr/sbin/dropbear -F -E -p ${DROPBEAR_PORT} -b /etc/issue.net
Restart=always
RestartSec=3

[Install]
WantedBy=multi-user.target"
  systemctl restart dropbear-extra.service || warn "Dropbear extra service did not start; check package path."
}

configure_stunnel() {
  log "Configuring Stunnel TLS wrapper on port ${STUNNEL_PORT} -> 127.0.0.1:${SSH_BACKEND_PORT}"
  local cert="/etc/stunnel/3x-ui-extra.pem"
  [[ -s "$cert" ]] || openssl req -new -x509 -days 3650 -nodes \
    -out "$cert" -keyout "$cert" -subj "/CN=3x-ui-extra" >/dev/null 2>&1
  chmod 600 "$cert"
  cat > /etc/stunnel/3x-ui-extra.conf <<EOF_STUNNEL
foreground = yes
pid = /run/stunnel-3x-ui-extra.pid
cert = ${cert}

[ssh-tls]
accept = 0.0.0.0:${STUNNEL_PORT}
connect = 127.0.0.1:${SSH_BACKEND_PORT}
TIMEOUTclose = 0
EOF_STUNNEL
  write_service "stunnel-extra" "[Unit]
Description=3x-ui Extra Stunnel SSH TLS
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=/usr/bin/stunnel /etc/stunnel/3x-ui-extra.conf
Restart=always
RestartSec=3

[Install]
WantedBy=multi-user.target"
  systemctl restart stunnel-extra.service || warn "Stunnel extra service did not start."
}

install_ssh_ws() {
  log "Installing lightweight SSH WebSocket bridge"
  cat > "${BIN_DIR}/ssh-ws" <<'PY'
#!/usr/bin/env python3
import asyncio, base64, hashlib, os, signal, struct

HOST = os.environ.get("SSH_WS_HOST", "0.0.0.0")
PORT = int(os.environ.get("SSH_WS_PORT", "8443"))
BACKEND_HOST = os.environ.get("SSH_WS_BACKEND_HOST", "127.0.0.1")
BACKEND_PORT = int(os.environ.get("SSH_WS_BACKEND_PORT", "22"))
GUID = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"

async def read_http(reader):
    data = b""
    while b"\r\n\r\n" not in data and len(data) < 16384:
        chunk = await reader.read(1024)
        if not chunk:
            break
        data += chunk
    return data.decode("latin1", "ignore")

def parse_headers(req):
    headers = {}
    for line in req.split("\r\n")[1:]:
        if ":" in line:
            k, v = line.split(":", 1)
            headers[k.strip().lower()] = v.strip()
    return headers

async def ws_recv(reader):
    head = await reader.readexactly(2)
    length = head[1] & 0x7F
    if length == 126:
        length = struct.unpack("!H", await reader.readexactly(2))[0]
    elif length == 127:
        length = struct.unpack("!Q", await reader.readexactly(8))[0]
    masked = head[1] & 0x80
    mask = await reader.readexactly(4) if masked else b""
    payload = await reader.readexactly(length) if length else b""
    if masked:
        payload = bytes(b ^ mask[i % 4] for i, b in enumerate(payload))
    return head[0] & 0x0F, payload

def ws_frame(payload):
    n = len(payload)
    if n < 126:
        return bytes([0x82, n]) + payload
    if n < 65536:
        return bytes([0x82, 126]) + struct.pack("!H", n) + payload
    return bytes([0x82, 127]) + struct.pack("!Q", n) + payload

async def proxy_ws_to_tcp(ws_reader, tcp_writer):
    try:
        while True:
            opcode, payload = await ws_recv(ws_reader)
            if opcode in (0x8, 0x9, 0xA):
                if opcode == 0x8: break
                continue
            tcp_writer.write(payload)
            await tcp_writer.drain()
    except Exception:
        pass
    finally:
        tcp_writer.close()

async def proxy_tcp_to_ws(tcp_reader, ws_writer):
    try:
        while True:
            data = await tcp_reader.read(4096)
            if not data: break
            ws_writer.write(ws_frame(data))
            await ws_writer.drain()
    except Exception:
        pass
    finally:
        ws_writer.close()

async def handle(reader, writer):
    req = await read_http(reader)
    headers = parse_headers(req)
    key = headers.get("sec-websocket-key", "")
    if not key:
        writer.write(b"HTTP/1.1 400 Bad Request\r\n\r\n")
        await writer.drain(); writer.close(); return
    accept = base64.b64encode(hashlib.sha1((key + GUID).encode()).digest()).decode()
    writer.write(("HTTP/1.1 101 Switching Protocols\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Accept: " + accept + "\r\n\r\n").encode())
    await writer.drain()
    tcp_reader, tcp_writer = await asyncio.open_connection(BACKEND_HOST, BACKEND_PORT)
    await asyncio.gather(proxy_ws_to_tcp(reader, tcp_writer), proxy_tcp_to_ws(tcp_reader, writer))

async def main():
    server = await asyncio.start_server(handle, HOST, PORT)
    async with server:
        await server.serve_forever()

if __name__ == "__main__":
    asyncio.run(main())
PY
  chmod +x "${BIN_DIR}/ssh-ws"
  write_service "ssh-ws-extra" "[Unit]
Description=3x-ui Extra SSH over WebSocket
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
Environment=SSH_WS_PORT=${SSHWS_PORT}
Environment=SSH_WS_BACKEND_HOST=127.0.0.1
Environment=SSH_WS_BACKEND_PORT=${SSH_BACKEND_PORT}
ExecStart=${BIN_DIR}/ssh-ws
Restart=always
RestartSec=3

[Install]
WantedBy=multi-user.target"
  systemctl restart ssh-ws-extra.service || warn "SSH-WS service did not start."
}

install_udp_custom() {
  log "Installing UDP Custom / BadVPN udpgw"
  local tmp="$(mktemp -d)"
  if clone_with_fallback "$tmp/badvpn" \
      "https://github.com/ambrop72/badvpn" \
      "https://github.com/daynix/badvpn"; then
    (cd "$tmp/badvpn" && cmake -DBUILD_NOTHING_BY_DEFAULT=1 -DBUILD_UDPGW=1 . && make -j"$(nproc || echo 1)") || warn "BadVPN source build failed."
    if [[ -x "$tmp/badvpn/udpgw/badvpn-udpgw" ]]; then
      install -m 0755 "$tmp/badvpn/udpgw/badvpn-udpgw" "${BIN_DIR}/badvpn-udpgw"
    fi
  fi
  if [[ ! -x "${BIN_DIR}/badvpn-udpgw" ]]; then
    warn "Falling back to distro badvpn package if available."
    command -v badvpn-udpgw >/dev/null 2>&1 && cp "$(command -v badvpn-udpgw)" "${BIN_DIR}/badvpn-udpgw" || true
  fi
  [[ -x "${BIN_DIR}/badvpn-udpgw" ]] || warn "badvpn-udpgw binary unavailable; install cmake and re-run if needed."
  write_service "udp-custom-extra" "[Unit]
Description=3x-ui Extra UDP Custom BadVPN Gateway
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=${BIN_DIR}/badvpn-udpgw --listen-addr 0.0.0.0:${UDP_CUSTOM_PORT} --max-clients 1024 --max-connections-for-client 16
Restart=always
RestartSec=3

[Install]
WantedBy=multi-user.target"
  systemctl restart udp-custom-extra.service || warn "UDP Custom service did not start."
  rm -rf "$tmp"
}

install_slowdns() {
  log "Installing SlowDNS (DNSTT) from ${DNSTT_REPO}"
  local tmp="$(mktemp -d)"
  clone_with_fallback "$tmp/dnstt" "$DNSTT_REPO" "https://github.com/powermx/dnstt.git" || { warn "DNSTT clone failed."; rm -rf "$tmp"; return; }
  (cd "$tmp/dnstt" && go build -o "${BIN_DIR}/dnstt-server" ./dnstt-server && go build -o "${BIN_DIR}/dnstt-client" ./dnstt-client) || warn "DNSTT build failed."
  chmod +x "${BIN_DIR}/dnstt-server" "${BIN_DIR}/dnstt-client" 2>/dev/null || true
  if [[ ! -s /etc/dnstt/server.key || ! -s /etc/dnstt/server.pub ]]; then
    if [[ -x "${BIN_DIR}/dnstt-server" ]]; then
      "${BIN_DIR}/dnstt-server" -gen-key -privkey-file /etc/dnstt/server.key -pubkey-file /etc/dnstt/server.pub || true
    fi
  fi
  cat > /etc/dnstt/README.extra <<EOF_DNSTT
Set your NS/domain in the panel user's Protocol Config Payload as JSON, for example:
{"domain":"dns.example.com","publicKey":"$(cat /etc/dnstt/server.pub 2>/dev/null || true)"}
EOF_DNSTT
  write_service "slowdns-extra" "[Unit]
Description=3x-ui Extra SlowDNS DNSTT Server
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=${BIN_DIR}/dnstt-server -udp :${SLOWDNS_PORT} -privkey-file /etc/dnstt/server.key 127.0.0.1:${SSH_BACKEND_PORT}
Restart=always
RestartSec=3

[Install]
WantedBy=multi-user.target"
  systemctl restart slowdns-extra.service || warn "SlowDNS service did not start; configure NS/domain and check logs."
  rm -rf "$tmp"
}

install_psiphon() {
  log "Installing Psiphon from approved repositories"
  local tmp="$(mktemp -d)"
  clone_with_fallback "$tmp/psiphon" "${PSIPHON_REPOS[@]}" || { warn "Psiphon clone failed."; rm -rf "$tmp"; return; }
  if find "$tmp/psiphon" -name go.mod -print -quit | grep -q .; then
    local moddir
    moddir="$(dirname "$(find "$tmp/psiphon" -name go.mod -print -quit)")"
    (cd "$moddir" && go build -o "${BIN_DIR}/psiphon-server" ./... ) || warn "Psiphon Go build failed; repository layout may require manual build."
  fi
  if [[ ! -x "${BIN_DIR}/psiphon-server" ]]; then
    cat > "${BIN_DIR}/psiphon-server" <<'SH'
#!/usr/bin/env bash
echo "Psiphon server binary was not built automatically. Please build from /opt/psiphon or the approved repo and replace this stub." >&2
sleep infinity
SH
    chmod +x "${BIN_DIR}/psiphon-server"
  fi
  mkdir -p /opt
  rm -rf /opt/psiphon
  cp -a "$tmp/psiphon" /opt/psiphon
  cat > /etc/psiphon/server.json <<EOF_PSIPHON
{
  "listen_port": ${PSIPHON_PORT},
  "ssh_port": ${SSH_BACKEND_PORT},
  "web_server_port": ${PSIPHON_PORT}
}
EOF_PSIPHON
  write_service "psiphon-extra" "[Unit]
Description=3x-ui Extra Psiphon Server
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=${BIN_DIR}/psiphon-server -config /etc/psiphon/server.json
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target"
  systemctl restart psiphon-extra.service || warn "Psiphon service did not start; inspect /opt/psiphon for repo-specific build steps."
  rm -rf "$tmp"
}

main() {
  need_root
  local pm
  pm="$(detect_pkg_manager)"
  ensure_dirs
  pkg_bootstrap "$pm"
  # cmake is only needed for BadVPN source builds; install best-effort.
  install_packages "$pm" cmake >/dev/null 2>&1 || true

  configure_ssh
  configure_dropbear
  configure_stunnel
  install_ssh_ws
  install_udp_custom
  install_slowdns
  install_psiphon

  systemctl daemon-reload
  log "Installation complete. Tune ports with env vars before running if needed: SSH_PORT, DROPBEAR_PORT, STUNNEL_PORT, SSHWS_PORT, UDP_CUSTOM_PORT, SLOWDNS_PORT, PSIPHON_PORT."
}

main "$@"