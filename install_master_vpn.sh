#!/usr/bin/env bash
set -Eeuo pipefail

# 3x-ui Multiport Hybrid Master VPN Installer
#
# Sources integrated by design:
#   - SSH/Dropbear/Stunnel/SSH-WS compatibility assets: https://github.com/KhaiVpn767/multiport
#   - UDP Custom BadVPN gateway: https://github.com/ambrop72/badvpn
#   - SlowDNS DNSTT installer assets/logic: https://github.com/powermx/dnstt
#   - Psiphon installer baseline: https://github.com/tipsytux/psiphon
#
# This script intentionally creates real daemon-backed services and stores the
# real generated client material where the Go backend reads it:
#   /etc/dnstt/server.pub
#   /etc/psiphon/server-entry.dat

export DEBIAN_FRONTEND=noninteractive

BIN_DIR="${BIN_DIR:-/usr/local/bin}"
SRC_DIR="${SRC_DIR:-/usr/local/src/3x-ui-hybrid}"
LOG_DIR="${LOG_DIR:-/var/log/3x-ui-hybrid}"

SSH_PORT="${SSH_PORT:-22}"
DROPBEAR_PORT="${DROPBEAR_PORT:-143}"
STUNNEL_PORTS="${STUNNEL_PORTS:-443 444}"
SSHWS_PORTS="${SSHWS_PORTS:-80 8880}"
SSH_BACKEND_HOST="${SSH_BACKEND_HOST:-127.0.0.1}"
SSH_BACKEND_PORT="${SSH_BACKEND_PORT:-22}"
UDP_CUSTOM_PORT="${UDP_CUSTOM_PORT:-7300}"
SLOWDNS_NS="${SLOWDNS_NS:-ns.$(hostname -f 2>/dev/null || hostname).}"
SLOWDNS_UDP_LISTEN="${SLOWDNS_UDP_LISTEN:-:53}"
SLOWDNS_BACKEND="${SLOWDNS_BACKEND:-127.0.0.1:${DROPBEAR_PORT}}"
PSIPHON_PORT="${PSIPHON_PORT:-3001}"

KHAI_REPO="${KHAI_REPO:-https://github.com/KhaiVpn767/multiport.git}"
BADVPN_REPO="${BADVPN_REPO:-https://github.com/ambrop72/badvpn.git}"
DNSTT_REPO="${DNSTT_REPO:-https://github.com/powermx/dnstt.git}"
PSIPHON_REPO="${PSIPHON_REPO:-https://github.com/tipsytux/psiphon.git}"
PSIPHON_CORE_REPO="${PSIPHON_CORE_REPO:-https://github.com/Psiphon-Labs/psiphon-tunnel-core.git}"

log() { printf '\033[1;32m[master-vpn]\033[0m %s\n' "$*"; }
warn() { printf '\033[1;33m[master-vpn][warn]\033[0m %s\n' "$*" >&2; }
die() { printf '\033[1;31m[master-vpn][error]\033[0m %s\n' "$*" >&2; exit 1; }

need_root() { [[ "${EUID}" -eq 0 ]] || die "Run as root."; }

detect_pkg_manager() {
  if command -v apt-get >/dev/null 2>&1; then echo apt; return; fi
  if command -v dnf >/dev/null 2>&1; then echo dnf; return; fi
  if command -v yum >/dev/null 2>&1; then echo yum; return; fi
  if command -v pacman >/dev/null 2>&1; then echo pacman; return; fi
  die "Unsupported distro. Need apt, dnf, yum, or pacman."
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

bootstrap_packages() {
  local pm="$1"
  log "Installing required runtime/build dependencies (${pm})"
  case "$pm" in
    apt)
      install_packages "$pm" ca-certificates curl wget git build-essential cmake golang-go openssh-server dropbear stunnel4 python3 openssl tar gzip jq
      ;;
    dnf|yum)
      install_packages "$pm" ca-certificates curl wget git gcc gcc-c++ make cmake golang openssh-server dropbear stunnel python3 openssl tar gzip jq || true
      ;;
    pacman)
      install_packages "$pm" ca-certificates curl wget git base-devel cmake go openssh dropbear stunnel python openssl tar gzip jq
      ;;
  esac
}

ensure_dirs() {
  mkdir -p "$BIN_DIR" "$SRC_DIR" "$LOG_DIR" /etc/systemd/system \
    /etc/dropbear /etc/stunnel /etc/ssh/sshd_config.d /etc/ssh-ws \
    /etc/ssws /etc/dnstt /etc/psiphon /etc/slowdns
}

fetch_repo() {
  local url="$1" dest="$2"
  if [[ -d "$dest/.git" ]]; then
    git -C "$dest" fetch --depth=1 origin >/dev/null 2>&1 || true
    git -C "$dest" reset --hard FETCH_HEAD >/dev/null 2>&1 || true
  else
    rm -rf "$dest"
    git clone --depth=1 "$url" "$dest"
  fi
}

public_ip() {
  curl -4fsS --max-time 4 https://api.ipify.org 2>/dev/null || hostname -I 2>/dev/null | awk '{print $1}' || echo "YOUR_SERVER_IP"
}

write_service() {
  local name="$1" content="$2"
  printf '%s\n' "$content" > "/etc/systemd/system/${name}.service"
  systemctl daemon-reload
  systemctl enable "${name}.service" >/dev/null 2>&1 || true
}

restart_service() {
  local name="$1"
  systemctl restart "${name}.service" || warn "${name}.service did not start. Check: systemctl status ${name}.service"
}

install_khai_assets() {
  log "Fetching KhaiVpn767 SSH/Dropbear/Stunnel/WS assets"
  fetch_repo "$KHAI_REPO" "$SRC_DIR/khai-multiport"
  install -m 0644 "$SRC_DIR/khai-multiport/ssh-vpn/ssh-vpn.sh" /etc/ssh/khai-ssh-vpn.sh 2>/dev/null || true
  install -m 0755 "$SRC_DIR/khai-multiport/websocket-python/cdn-openssh.py" /etc/ssh-ws/khai-cdn-openssh.py 2>/dev/null || true
  install -m 0755 "$SRC_DIR/khai-multiport/websocket-python/cdn-dropbear.py" /etc/ssh-ws/khai-cdn-dropbear.py 2>/dev/null || true
}

configure_openssh() {
  log "Configuring OpenSSH legacy-compatible ciphers on ${SSH_PORT}"
  cat > /etc/ssh/sshd_config.d/99-3x-ui-hybrid.conf <<EOF_SSH
Port ${SSH_PORT}
PasswordAuthentication yes
KbdInteractiveAuthentication yes
UsePAM yes
PermitTunnel yes
AllowTcpForwarding yes
X11Forwarding no
Banner /etc/issue.net

# HTTP Custom compatibility profile inspired by KhaiVpn767/multiport SSH setup.
KexAlgorithms curve25519-sha256,curve25519-sha256@libssh.org,diffie-hellman-group-exchange-sha256,diffie-hellman-group14-sha256,diffie-hellman-group14-sha1,diffie-hellman-group1-sha1
Ciphers chacha20-poly1305@openssh.com,aes128-ctr,aes192-ctr,aes256-ctr,aes128-cbc,aes192-cbc,aes256-cbc,3des-cbc
MACs hmac-sha2-256,hmac-sha2-512,hmac-sha1,hmac-sha1-96
HostKeyAlgorithms +ssh-rsa
PubkeyAcceptedAlgorithms +ssh-rsa
EOF_SSH
  touch /etc/issue.net
  systemctl enable ssh >/dev/null 2>&1 || systemctl enable sshd >/dev/null 2>&1 || true
  systemctl restart ssh >/dev/null 2>&1 || systemctl restart sshd >/dev/null 2>&1 || warn "Could not restart ssh/sshd."
}

configure_dropbear() {
  log "Configuring Dropbear on ${DROPBEAR_PORT}"
  local dropbear_bin
  dropbear_bin="$(command -v dropbear || true)"
  [[ -n "$dropbear_bin" ]] || die "dropbear binary not found."
  cat > /etc/default/dropbear-3x-ui-hybrid <<EOF_DROPBEAR
DROPBEAR_ARGS="-F -E -p 0.0.0.0:${DROPBEAR_PORT} -b /etc/issue.net -K 60 -I 300"
EOF_DROPBEAR
  write_service "dropbear-hybrid" "[Unit]
Description=3x-ui Hybrid Dropbear SSH
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
EnvironmentFile=-/etc/default/dropbear-3x-ui-hybrid
ExecStart=${dropbear_bin} \$DROPBEAR_ARGS
Restart=always
RestartSec=3

[Install]
WantedBy=multi-user.target"
  restart_service "dropbear-hybrid"
}

generate_stunnel_cert() {
  local cert="/etc/stunnel/3x-ui-hybrid.pem"
  if [[ ! -s "$cert" ]]; then
    openssl req -new -x509 -days 3650 -nodes -out "$cert" -keyout "$cert" -subj "/CN=3x-ui-hybrid-stunnel" >/dev/null 2>&1
  fi
  chmod 600 "$cert"
  printf '%s' "$cert"
}

configure_stunnel() {
  log "Configuring Stunnel on ${STUNNEL_PORTS} -> ${SSH_BACKEND_HOST}:${SSH_BACKEND_PORT}"
  local stunnel_bin cert p
  stunnel_bin="$(command -v stunnel4 || command -v stunnel || true)"
  [[ -n "$stunnel_bin" ]] || die "stunnel/stunnel4 not found."
  cert="$(generate_stunnel_cert)"
  cat > /etc/stunnel/3x-ui-hybrid.conf <<EOF_STUNNEL
foreground = yes
pid = /run/stunnel-3x-ui-hybrid.pid
cert = ${cert}
sslVersion = all
ciphers = ALL:@SECLEVEL=0
ciphersuites = TLS_AES_256_GCM_SHA384:TLS_CHACHA20_POLY1305_SHA256:TLS_AES_128_GCM_SHA256
options = NO_SSLv2
options = NO_SSLv3
TIMEOUTclose = 0

EOF_STUNNEL
  for p in ${STUNNEL_PORTS}; do
    cat >> /etc/stunnel/3x-ui-hybrid.conf <<EOF_STUNNEL_PORT
[ssh-tls-${p}]
accept = 0.0.0.0:${p}
connect = ${SSH_BACKEND_HOST}:${SSH_BACKEND_PORT}

EOF_STUNNEL_PORT
  done
  sed -i 's/^ENABLED=.*/ENABLED=1/' /etc/default/stunnel4 2>/dev/null || true
  write_service "stunnel-hybrid" "[Unit]
Description=3x-ui Hybrid Stunnel SSL/TLS to SSH
After=network-online.target ssh.service sshd.service dropbear-hybrid.service
Wants=network-online.target

[Service]
Type=simple
ExecStart=${stunnel_bin} /etc/stunnel/3x-ui-hybrid.conf
Restart=always
RestartSec=3

[Install]
WantedBy=multi-user.target"
  restart_service "stunnel-hybrid"
}

install_ssh_ws() {
  log "Installing SSH WebSocket bridge from KhaiVpn767-compatible asset/fallback"
  local ws_script="/etc/ssh-ws/khai-cdn-openssh.py"
  if [[ -s "$ws_script" ]] && command -v python2 >/dev/null 2>&1; then
    cat > /etc/default/ssh-ws-hybrid <<EOF_WS_ENV
SSHWS_PORTS="${SSHWS_PORTS}"
SSHWS_SCRIPT="${ws_script}"
EOF_WS_ENV
    cat > "$BIN_DIR/ssh-ws-hybrid-runner" <<'EOF_RUNNER'
#!/usr/bin/env bash
set -Eeuo pipefail
source /etc/default/ssh-ws-hybrid
pids=()
for port in ${SSHWS_PORTS}; do
  python2 "${SSHWS_SCRIPT}" "${port}" &
  pids+=("$!")
done
trap 'kill "${pids[@]}" 2>/dev/null || true' EXIT
wait -n "${pids[@]}"
EOF_RUNNER
    chmod +x "$BIN_DIR/ssh-ws-hybrid-runner"
  else
    warn "Khai Python2 WS asset unavailable or python2 missing; installing Python3 compatible bridge."
    cat > "$BIN_DIR/ssh-ws-hybrid-runner" <<'PY'
#!/usr/bin/env python3
import asyncio, base64, hashlib, os, struct
HOST = "0.0.0.0"
PORTS = [int(p) for p in os.environ.get("SSHWS_PORTS", "80 8880").split()]
BACKEND_HOST = os.environ.get("SSH_BACKEND_HOST", "127.0.0.1")
BACKEND_PORT = int(os.environ.get("SSH_BACKEND_PORT", "22"))
GUID = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"
async def read_http(reader):
    data = b""
    while b"\r\n\r\n" not in data and len(data) < 65536:
        chunk = await reader.read(2048)
        if not chunk: break
        data += chunk
    return data.decode("latin1", "ignore")
def headers(req):
    h = {}
    for line in req.split("\r\n")[1:]:
        if ":" in line:
            k, v = line.split(":", 1); h[k.strip().lower()] = v.strip()
    return h
async def pipe(src, dst):
    try:
        while True:
            data = await src.read(8192)
            if not data: break
            dst.write(data); await dst.drain()
    except Exception: pass
    finally: dst.close()
async def ws_recv(reader):
    head = await reader.readexactly(2); length = head[1] & 0x7F
    if length == 126: length = struct.unpack("!H", await reader.readexactly(2))[0]
    elif length == 127: length = struct.unpack("!Q", await reader.readexactly(8))[0]
    mask = await reader.readexactly(4) if head[1] & 0x80 else b""
    payload = await reader.readexactly(length) if length else b""
    if mask: payload = bytes(b ^ mask[i % 4] for i, b in enumerate(payload))
    return head[0] & 0x0F, payload
def ws_frame(payload):
    n = len(payload)
    if n < 126: return bytes([0x82, n]) + payload
    if n < 65536: return bytes([0x82, 126]) + struct.pack("!H", n) + payload
    return bytes([0x82, 127]) + struct.pack("!Q", n) + payload
async def ws_to_tcp(ws_reader, tcp_writer):
    try:
        while True:
            opcode, payload = await ws_recv(ws_reader)
            if opcode == 0x8: break
            if opcode in (0x1, 0x2, 0x0): tcp_writer.write(payload); await tcp_writer.drain()
    except Exception: pass
    finally: tcp_writer.close()
async def tcp_to_ws(tcp_reader, ws_writer):
    try:
        while True:
            data = await tcp_reader.read(8192)
            if not data: break
            ws_writer.write(ws_frame(data)); await ws_writer.drain()
    except Exception: pass
    finally: ws_writer.close()
async def handle(reader, writer):
    req = await read_http(reader); h = headers(req)
    tcp_reader, tcp_writer = await asyncio.open_connection(BACKEND_HOST, BACKEND_PORT)
    key = h.get("sec-websocket-key", "")
    if key:
        accept = base64.b64encode(hashlib.sha1((key + GUID).encode()).digest()).decode()
        writer.write(("HTTP/1.1 101 Switching Protocols\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Accept: " + accept + "\r\n\r\n").encode()); await writer.drain()
        await asyncio.gather(ws_to_tcp(reader, tcp_writer), tcp_to_ws(tcp_reader, writer))
    else:
        writer.write(b"HTTP/1.1 101 <b><u><font color=\"blue\">Script By comingsoon</font></b>\r\n\r\n\r\n\r\nContent-Length: 104857600000\r\n\r\n"); await writer.drain()
        await asyncio.gather(pipe(reader, tcp_writer), pipe(tcp_reader, writer))
async def main():
    servers = [await asyncio.start_server(handle, HOST, p) for p in PORTS]
    await asyncio.gather(*(s.serve_forever() for s in servers))
asyncio.run(main())
PY
    chmod +x "$BIN_DIR/ssh-ws-hybrid-runner"
  fi
  write_service "ssh-ws-hybrid" "[Unit]
Description=3x-ui Hybrid SSH over WebSocket
After=network-online.target ssh.service sshd.service dropbear-hybrid.service
Wants=network-online.target

[Service]
Type=simple
Environment=SSHWS_PORTS=${SSHWS_PORTS}
Environment=SSH_BACKEND_HOST=${SSH_BACKEND_HOST}
Environment=SSH_BACKEND_PORT=${SSH_BACKEND_PORT}
ExecStart=${BIN_DIR}/ssh-ws-hybrid-runner
Restart=always
RestartSec=3

[Install]
WantedBy=multi-user.target"
  restart_service "ssh-ws-hybrid"
}

install_udp_custom_badvpn() {
  log "Building BadVPN udpgw from ${BADVPN_REPO} on port ${UDP_CUSTOM_PORT}"
  fetch_repo "$BADVPN_REPO" "$SRC_DIR/badvpn"
  cmake -S "$SRC_DIR/badvpn" -B "$SRC_DIR/badvpn/build" -DBUILD_NOTHING_BY_DEFAULT=1 -DBUILD_UDPGW=1
  cmake --build "$SRC_DIR/badvpn/build" --parallel "$(nproc 2>/dev/null || echo 1)"
  local built
  built="$(find "$SRC_DIR/badvpn/build" "$SRC_DIR/badvpn" -type f -name badvpn-udpgw -perm -111 2>/dev/null | head -n1 || true)"
  [[ -n "$built" ]] || die "badvpn-udpgw build failed."
  install -m 0755 "$built" "$BIN_DIR/badvpn-udpgw"
  write_service "udp-custom-badvpn" "[Unit]
Description=3x-ui Hybrid UDP Custom BadVPN Gateway
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=${BIN_DIR}/badvpn-udpgw --listen-addr 0.0.0.0:${UDP_CUSTOM_PORT} --max-clients 2048 --max-connections-for-client 32
Restart=always
RestartSec=3

[Install]
WantedBy=multi-user.target"
  restart_service "udp-custom-badvpn"
}

install_slowdns_dnstt() {
  log "Installing SlowDNS DNSTT using powermx/dnstt assets and DNSTT server"
  fetch_repo "$DNSTT_REPO" "$SRC_DIR/powermx-dnstt"
  install -m 0755 "$SRC_DIR/powermx-dnstt/dns-server" "$BIN_DIR/dnstt-server" 2>/dev/null || true
  if [[ ! -x "$BIN_DIR/dnstt-server" ]]; then
    GOBIN="$BIN_DIR" go install www.bamsoftware.com/git/dnstt.git/dnstt-server@latest
  fi
  [[ -x "$BIN_DIR/dnstt-server" ]] || die "dnstt-server unavailable."
  if [[ ! -s /etc/dnstt/server.key || ! -s /etc/dnstt/server.pub ]]; then
    "$BIN_DIR/dnstt-server" -gen-key -privkey-file /etc/dnstt/server.key -pubkey-file /etc/dnstt/server.pub
  fi
  chmod 600 /etc/dnstt/server.key
  chmod 644 /etc/dnstt/server.pub
  printf '%s\n' "$SLOWDNS_NS" > /etc/dnstt/nameserver
  write_service "dnstt-server" "[Unit]
Description=3x-ui Hybrid SlowDNS DNSTT Server
After=network-online.target ssh.service sshd.service dropbear-hybrid.service
Wants=network-online.target

[Service]
Type=simple
ExecStart=${BIN_DIR}/dnstt-server -udp ${SLOWDNS_UDP_LISTEN} -privkey-file /etc/dnstt/server.key ${SLOWDNS_NS} ${SLOWDNS_BACKEND}
Restart=always
RestartSec=3
LimitNOFILE=1048576

[Install]
WantedBy=multi-user.target"
  restart_service "dnstt-server"
}

install_psiphon() {
  log "Installing Psiphon using tipsytux/psiphon baseline and Psiphon core server"
  fetch_repo "$PSIPHON_REPO" "$SRC_DIR/tipsytux-psiphon"
  install -m 0644 "$SRC_DIR/tipsytux-psiphon/run.sh" /etc/psiphon/tipsytux-run.sh 2>/dev/null || true

  if [[ ! -x "$BIN_DIR/psiphond" ]]; then
    fetch_repo "$PSIPHON_CORE_REPO" "$SRC_DIR/psiphon-tunnel-core" || warn "Could not fetch Psiphon core repo."
    if [[ -d "$SRC_DIR/psiphon-tunnel-core" ]]; then
      (cd "$SRC_DIR/psiphon-tunnel-core" && GOBIN="$BIN_DIR" go install ./Server/psiphond 2>/dev/null) || \
      (cd "$SRC_DIR/psiphon-tunnel-core/Server" && go build -o "$BIN_DIR/psiphond" . 2>/dev/null) || true
    fi
  fi

  local ip
  ip="$(public_ip)"
  cat > /etc/psiphon/server.json <<EOF_PSIPHON
{
  "ServerIPAddress": "${ip}",
  "TunnelProtocolPorts": { "SSH": ${PSIPHON_PORT}, "OSSH": 444, "UNFRONTED-MEEK-HTTPS": 8443 },
  "LogFilename": "${LOG_DIR}/psiphon.log",
  "GeoIPDatabaseFilename": "",
  "DiscoveryValueHMACKey": "$(openssl rand -hex 32)",
  "MeekCookieEncryptionPublicKey": "$(openssl rand -base64 32)",
  "MeekCookieEncryptionPrivateKey": "$(openssl rand -base64 32)"
}
EOF_PSIPHON
  chmod 600 /etc/psiphon/server.json

  if [[ -x "$BIN_DIR/psiphond" ]]; then
    "$BIN_DIR/psiphond" --config /etc/psiphon/server.json --server-entry /etc/psiphon/server-entry.dat generate >/dev/null 2>&1 || true
    write_service "psiphon-server" "[Unit]
Description=3x-ui Hybrid Psiphon Server
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=${BIN_DIR}/psiphond --config /etc/psiphon/server.json
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target"
    restart_service "psiphon-server"
  else
    warn "psiphond binary could not be built automatically. Place it at ${BIN_DIR}/psiphond and rerun."
  fi

  if [[ ! -s /etc/psiphon/server-entry.dat ]]; then
    cat > /etc/psiphon/server-entry.dat <<EOF_ENTRY
PSIPHON_SERVER_ENTRY_NOT_GENERATED
source=${PSIPHON_REPO}
server=${ip}:${PSIPHON_PORT}
note=Install a psiphond binary that supports server entry generation, then rerun install_master_vpn.sh.
EOF_ENTRY
    chmod 600 /etc/psiphon/server-entry.dat
  fi
}

print_summary() {
  local ip dnstt_pub psiphon_entry
  ip="$(public_ip)"
  dnstt_pub="$(cat /etc/dnstt/server.pub 2>/dev/null || echo DNSTT_PUBLIC_KEY_NOT_FOUND)"
  psiphon_entry="$(head -c 220 /etc/psiphon/server-entry.dat 2>/dev/null || echo PSIPHON_ENTRY_NOT_FOUND)"
  cat <<EOF_SUMMARY

Hybrid VPN installation complete.

Services:
  OpenSSH:             ${ip}:${SSH_PORT}
  Dropbear:            ${ip}:${DROPBEAR_PORT}
  SSL/Stunnel:         ${ip}:${STUNNEL_PORTS}
  SSH WebSocket:       ${ip}:${SSHWS_PORTS}
  UDP Custom BadVPN:   ${ip}:${UDP_CUSTOM_PORT}
  SlowDNS DNSTT NS:    ${SLOWDNS_NS}
  SlowDNS Public Key:  ${dnstt_pub}
  Psiphon Entry:       ${psiphon_entry}

Backend-readable files:
  /etc/dnstt/server.pub
  /etc/dnstt/server.key
  /etc/dnstt/nameserver
  /etc/psiphon/server-entry.dat
  /etc/psiphon/server.json

systemd services:
  dropbear-hybrid stunnel-hybrid ssh-ws-hybrid udp-custom-badvpn dnstt-server psiphon-server
EOF_SUMMARY
}

main() {
  need_root
  local pm
  pm="$(detect_pkg_manager)"
  ensure_dirs
  bootstrap_packages "$pm"
  install_khai_assets
  configure_openssh
  configure_dropbear
  configure_stunnel
  install_ssh_ws
  install_udp_custom_badvpn
  install_slowdns_dnstt
  install_psiphon
  systemctl daemon-reload
  print_summary
}

main "$@"