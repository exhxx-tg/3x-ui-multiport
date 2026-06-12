#!/usr/bin/env bash
set -Eeuo pipefail

# 3x-ui Multiport Extra Protocols installer for HTTP Custom-compatible clients.
#
# Services installed/configured:
#   - Dropbear SSH: ports 22 and 143
#   - Stunnel4 SSL/TLS: ports 443 and 444 -> Dropbear backend
#   - BadVPN udpgw: UDP gateway port 7300
#   - OpenVPN: TCP/UDP port 1194
#   - SSH over WebSocket bridge: port 80 -> SSH backend
#
# Important HTTP Custom TLS compatibility note:
# /etc/stunnel/stunnel.conf is written with sslVersion = all plus broad cipher
# settings to avoid "Cannot negotiate, proposals do not match" errors on
# clients that propose older/limited TLS suites.

export DEBIAN_FRONTEND=noninteractive

EXTRA_DIR="/etc/3x-ui/extra"
BIN_DIR="/usr/local/bin"
LOG_DIR="/var/log/3x-ui-extra"

DROPBEAR_PORTS="${DROPBEAR_PORTS:-22 143}"
DROPBEAR_BACKEND_PORT="${DROPBEAR_BACKEND_PORT:-143}"
STUNNEL_PORTS="${STUNNEL_PORTS:-443 444}"
SSH_BACKEND_HOST="${SSH_BACKEND_HOST:-127.0.0.1}"
SSH_BACKEND_PORT="${SSH_BACKEND_PORT:-143}"
OPENSSH_PORT="${OPENSSH_PORT:-2222}"
SSHWS_PORT="${SSHWS_PORT:-80}"
UDP_CUSTOM_PORT="${UDP_CUSTOM_PORT:-7300}"
OPENVPN_PORT="${OPENVPN_PORT:-1194}"

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
    apt)
      install_packages "$pm" ca-certificates curl wget git build-essential cmake golang openssh-server dropbear stunnel4 python3 openssl openvpn easy-rsa iptables tar gzip
      ;;
    dnf|yum)
      install_packages "$pm" ca-certificates curl wget git gcc gcc-c++ make cmake golang openssh-server dropbear stunnel python3 openssl openvpn easy-rsa iptables tar gzip || true
      ;;
    pacman)
      install_packages "$pm" ca-certificates curl wget git base-devel cmake go openssh dropbear stunnel python openssl openvpn easy-rsa iptables tar gzip
      ;;
  esac
}

ensure_dirs() {
  mkdir -p "$EXTRA_DIR" "$BIN_DIR" "$LOG_DIR" /etc/stunnel /etc/ssh-ws /etc/openvpn/3x-ui /etc/systemd/system
}

write_service() {
  local name="$1"
  local content="$2"
  printf '%s\n' "$content" > "/etc/systemd/system/${name}.service"
  systemctl daemon-reload
  systemctl enable "${name}.service" >/dev/null 2>&1 || true
}

restart_service() {
  local name="$1"
  systemctl restart "${name}.service" || warn "${name}.service did not start; inspect: systemctl status ${name}.service"
}

configure_openssh_backend() {
  log "Ensuring OpenSSH is available on non-conflicting port ${OPENSSH_PORT}; Dropbear owns ports ${DROPBEAR_PORTS}"
  mkdir -p /etc/ssh/sshd_config.d
  cat > /etc/ssh/sshd_config.d/99-3x-ui-extra.conf <<EOF_SSH
Port ${OPENSSH_PORT}
PasswordAuthentication yes
PubkeyAuthentication yes
PermitTunnel yes
Banner /etc/issue.net
EOF_SSH
  touch /etc/issue.net
  systemctl enable ssh >/dev/null 2>&1 || systemctl enable sshd >/dev/null 2>&1 || true
  systemctl restart ssh >/dev/null 2>&1 || systemctl restart sshd >/dev/null 2>&1 || warn "Could not restart ssh/sshd now."
}

dropbear_binary() {
  command -v dropbear || true
}

configure_dropbear() {
  log "Configuring Dropbear on ports: ${DROPBEAR_PORTS}"
  local dropbear_bin
  dropbear_bin="$(dropbear_binary)"
  [[ -n "$dropbear_bin" ]] || die "dropbear binary was not found after package installation."

  local args="-F -E -b /etc/issue.net"
  local p
  for p in ${DROPBEAR_PORTS}; do
    args="${args} -p 0.0.0.0:${p}"
  done

  cat > /etc/default/dropbear-3x-ui-extra <<EOF_DROPBEAR_ENV
DROPBEAR_ARGS="${args}"
EOF_DROPBEAR_ENV

  write_service "dropbear-extra" "[Unit]
Description=3x-ui Extra Dropbear SSH (ports ${DROPBEAR_PORTS})
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
EnvironmentFile=-/etc/default/dropbear-3x-ui-extra
ExecStart=${dropbear_bin} \$DROPBEAR_ARGS
Restart=always
RestartSec=3

[Install]
WantedBy=multi-user.target"
  restart_service "dropbear-extra"
}

generate_stunnel_cert() {
  local cert="/etc/stunnel/3x-ui-extra.pem"
  if [[ ! -s "$cert" ]]; then
    openssl req -new -x509 -days 3650 -nodes \
      -out "$cert" -keyout "$cert" -subj "/CN=3x-ui-http-custom" >/dev/null 2>&1
  fi
  chmod 600 "$cert"
  printf '%s' "$cert"
}

configure_stunnel() {
  log "Configuring Stunnel4 on ports ${STUNNEL_PORTS} -> 127.0.0.1:${DROPBEAR_BACKEND_PORT}"
  local stunnel_bin cert
  stunnel_bin="$(command -v stunnel4 || command -v stunnel || true)"
  [[ -n "$stunnel_bin" ]] || die "stunnel/stunnel4 binary was not found after package installation."
  cert="$(generate_stunnel_cert)"

  cat > /etc/stunnel/stunnel.conf <<EOF_STUNNEL
foreground = yes
pid = /run/stunnel-3x-ui-extra.pid
cert = ${cert}

; CRITICAL HTTP Custom compatibility settings:
; Accept broad TLS proposals to prevent: "Cannot negotiate, proposals do not match".
sslVersion = all
ciphers = ALL:@SECLEVEL=0
ciphersuites = TLS_AES_256_GCM_SHA384:TLS_CHACHA20_POLY1305_SHA256:TLS_AES_128_GCM_SHA256:TLS_AES_128_CCM_SHA256
options = NO_SSLv2
options = NO_SSLv3
TIMEOUTclose = 0

EOF_STUNNEL

  local p
  for p in ${STUNNEL_PORTS}; do
    cat >> /etc/stunnel/stunnel.conf <<EOF_STUNNEL_SERVICE
[ssh-tls-${p}]
accept = 0.0.0.0:${p}
connect = 127.0.0.1:${DROPBEAR_BACKEND_PORT}

EOF_STUNNEL_SERVICE
  done

  sed -i 's/^ENABLED=.*/ENABLED=1/' /etc/default/stunnel4 2>/dev/null || true
  write_service "stunnel-extra" "[Unit]
Description=3x-ui Extra Stunnel4 SSL/TLS for HTTP Custom
After=network-online.target dropbear-extra.service
Wants=network-online.target

[Service]
Type=simple
ExecStart=${stunnel_bin} /etc/stunnel/stunnel.conf
Restart=always
RestartSec=3

[Install]
WantedBy=multi-user.target"
  restart_service "stunnel-extra"
}

install_ssh_ws() {
  log "Installing SSH WebSocket bridge on port ${SSHWS_PORT} -> ${SSH_BACKEND_HOST}:${SSH_BACKEND_PORT}"
  cat > "${BIN_DIR}/ssh-ws" <<'PY'
#!/usr/bin/env python3
import asyncio, base64, hashlib, os, struct

HOST = os.environ.get("SSH_WS_HOST", "0.0.0.0")
PORT = int(os.environ.get("SSH_WS_PORT", "80"))
BACKEND_HOST = os.environ.get("SSH_WS_BACKEND_HOST", "127.0.0.1")
BACKEND_PORT = int(os.environ.get("SSH_WS_BACKEND_PORT", "22"))
GUID = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"

async def read_http(reader):
    data = b""
    while b"\r\n\r\n" not in data and len(data) < 65536:
        chunk = await reader.read(2048)
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
    opcode = head[0] & 0x0F
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
    return opcode, payload

def ws_frame(payload):
    n = len(payload)
    if n < 126:
        return bytes([0x82, n]) + payload
    if n < 65536:
        return bytes([0x82, 126]) + struct.pack("!H", n) + payload
    return bytes([0x82, 127]) + struct.pack("!Q", n) + payload

async def tcp_to_ws(tcp_reader, ws_writer):
    try:
        while True:
            data = await tcp_reader.read(8192)
            if not data:
                break
            ws_writer.write(ws_frame(data))
            await ws_writer.drain()
    except Exception:
        pass
    finally:
        ws_writer.close()

async def ws_to_tcp(ws_reader, tcp_writer):
    try:
        while True:
            opcode, payload = await ws_recv(ws_reader)
            if opcode == 0x8:
                break
            if opcode in (0x9, 0xA):
                continue
            tcp_writer.write(payload)
            await tcp_writer.drain()
    except Exception:
        pass
    finally:
        tcp_writer.close()

async def handle(reader, writer):
    req = await read_http(reader)
    headers = parse_headers(req)
    key = headers.get("sec-websocket-key", "")
    if not key:
        writer.write(b"HTTP/1.1 400 Bad Request\r\nConnection: close\r\n\r\n")
        await writer.drain(); writer.close(); return
    accept = base64.b64encode(hashlib.sha1((key + GUID).encode()).digest()).decode()
    writer.write((
        "HTTP/1.1 101 Switching Protocols\r\n"
        "Upgrade: websocket\r\n"
        "Connection: Upgrade\r\n"
        "Sec-WebSocket-Accept: " + accept + "\r\n\r\n"
    ).encode())
    await writer.drain()
    tcp_reader, tcp_writer = await asyncio.open_connection(BACKEND_HOST, BACKEND_PORT)
    await asyncio.gather(ws_to_tcp(reader, tcp_writer), tcp_to_ws(tcp_reader, writer))

async def main():
    server = await asyncio.start_server(handle, HOST, PORT)
    async with server:
        await server.serve_forever()

if __name__ == "__main__":
    asyncio.run(main())
PY
  chmod +x "${BIN_DIR}/ssh-ws"
  write_service "ssh-ws-extra" "[Unit]
Description=3x-ui Extra SSH over WebSocket for HTTP Custom
After=network-online.target ssh.service sshd.service
Wants=network-online.target

[Service]
Type=simple
Environment=SSH_WS_PORT=${SSHWS_PORT}
Environment=SSH_WS_BACKEND_HOST=${SSH_BACKEND_HOST}
Environment=SSH_WS_BACKEND_PORT=${SSH_BACKEND_PORT}
ExecStart=${BIN_DIR}/ssh-ws
Restart=always
RestartSec=3

[Install]
WantedBy=multi-user.target"
  restart_service "ssh-ws-extra"
}

install_udp_custom() {
  log "Installing BadVPN udpgw on UDP gateway port ${UDP_CUSTOM_PORT}"
  local tmp
  tmp="$(mktemp -d)"

  if command -v cmake >/dev/null 2>&1 && git clone --depth=1 https://github.com/ambrop72/badvpn "$tmp/badvpn" >/dev/null 2>&1; then
    (cd "$tmp/badvpn" && cmake -DBUILD_NOTHING_BY_DEFAULT=1 -DBUILD_UDPGW=1 . && make -j"$(nproc 2>/dev/null || echo 1)") || warn "BadVPN source build failed."
    if [[ -x "$tmp/badvpn/udpgw/badvpn-udpgw" ]]; then
      install -m 0755 "$tmp/badvpn/udpgw/badvpn-udpgw" "${BIN_DIR}/badvpn-udpgw"
    fi
  fi

  if [[ ! -x "${BIN_DIR}/badvpn-udpgw" ]] && command -v badvpn-udpgw >/dev/null 2>&1; then
    install -m 0755 "$(command -v badvpn-udpgw)" "${BIN_DIR}/badvpn-udpgw"
  fi

  rm -rf "$tmp"
  [[ -x "${BIN_DIR}/badvpn-udpgw" ]] || die "badvpn-udpgw binary unavailable. Install/build BadVPN and re-run."

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
  restart_service "udp-custom-extra"
}

generate_openvpn_pki() {
  local dir="/etc/openvpn/3x-ui"
  local ca_key="${dir}/ca.key"
  local ca_crt="${dir}/ca.crt"
  local srv_key="${dir}/server.key"
  local srv_crt="${dir}/server.crt"
  local dh="${dir}/dh.pem"

  if [[ ! -s "$ca_crt" || ! -s "$ca_key" ]]; then
    openssl genrsa -out "$ca_key" 2048 >/dev/null 2>&1
    openssl req -x509 -new -nodes -key "$ca_key" -sha256 -days 3650 -out "$ca_crt" -subj "/CN=3x-ui-openvpn-ca" >/dev/null 2>&1
  fi
  if [[ ! -s "$srv_crt" || ! -s "$srv_key" ]]; then
    openssl genrsa -out "$srv_key" 2048 >/dev/null 2>&1
    openssl req -new -key "$srv_key" -out "${dir}/server.csr" -subj "/CN=3x-ui-openvpn-server" >/dev/null 2>&1
    openssl x509 -req -in "${dir}/server.csr" -CA "$ca_crt" -CAkey "$ca_key" -CAcreateserial -out "$srv_crt" -days 3650 -sha256 >/dev/null 2>&1
  fi
  [[ -s "$dh" ]] || openssl dhparam -out "$dh" 2048 >/dev/null 2>&1 || true
  chmod 600 "$ca_key" "$srv_key" 2>/dev/null || true
}

configure_openvpn() {
  log "Configuring OpenVPN TCP/UDP on port ${OPENVPN_PORT}"
  local openvpn_bin
  openvpn_bin="$(command -v openvpn || true)"
  [[ -n "$openvpn_bin" ]] || die "openvpn binary was not found after package installation."
  generate_openvpn_pki

  cat > /etc/openvpn/3x-ui/server-udp.conf <<EOF_OVPN_UDP
port ${OPENVPN_PORT}
proto udp
dev tun-3xui-udp
ca /etc/openvpn/3x-ui/ca.crt
cert /etc/openvpn/3x-ui/server.crt
key /etc/openvpn/3x-ui/server.key
dh /etc/openvpn/3x-ui/dh.pem
server 10.88.0.0 255.255.255.0
ifconfig-pool-persist /etc/openvpn/3x-ui/ipp-udp.txt
keepalive 10 120
cipher AES-128-CBC
data-ciphers AES-128-CBC
auth SHA256
persist-key
persist-tun
user nobody
group nogroup
status /var/log/3x-ui-extra/openvpn-udp-status.log
verb 3
EOF_OVPN_UDP

  cat > /etc/openvpn/3x-ui/server-tcp.conf <<EOF_OVPN_TCP
port ${OPENVPN_PORT}
proto tcp-server
dev tun-3xui-tcp
ca /etc/openvpn/3x-ui/ca.crt
cert /etc/openvpn/3x-ui/server.crt
key /etc/openvpn/3x-ui/server.key
dh /etc/openvpn/3x-ui/dh.pem
server 10.89.0.0 255.255.255.0
ifconfig-pool-persist /etc/openvpn/3x-ui/ipp-tcp.txt
keepalive 10 120
cipher AES-128-CBC
data-ciphers AES-128-CBC
auth SHA256
persist-key
persist-tun
user nobody
group nogroup
status /var/log/3x-ui-extra/openvpn-tcp-status.log
verb 3
EOF_OVPN_TCP

  if ! getent group nogroup >/dev/null 2>&1; then
    sed -i 's/^group nogroup/group nobody/' /etc/openvpn/3x-ui/server-*.conf 2>/dev/null || true
  fi

  write_service "openvpn-3x-ui-udp" "[Unit]
Description=3x-ui Extra OpenVPN UDP
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=${openvpn_bin} --config /etc/openvpn/3x-ui/server-udp.conf
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target"

  write_service "openvpn-3x-ui-tcp" "[Unit]
Description=3x-ui Extra OpenVPN TCP
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=${openvpn_bin} --config /etc/openvpn/3x-ui/server-tcp.conf
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target"

  sysctl -w net.ipv4.ip_forward=1 >/dev/null 2>&1 || true
  printf 'net.ipv4.ip_forward=1\n' > /etc/sysctl.d/99-3x-ui-extra.conf
  restart_service "openvpn-3x-ui-udp"
  restart_service "openvpn-3x-ui-tcp"
}

print_summary() {
  cat <<EOF_SUMMARY

Installation complete.

HTTP Custom fields generated by the panel now map to:
  SSH Account:        SERVER:22@USER:PASS or SERVER:143@USER:PASS
  SSL/Stunnel:        SERVER:443@USER:PASS with SNI SERVER/DOMAIN
  SSH-WS RemoteProxy: SERVER:80@USER:PASS plus WebSocket payload
  UDP Custom:         SSH account + UDP Gateway Port ${UDP_CUSTOM_PORT}
  OpenVPN:            remote SERVER ${OPENVPN_PORT}, cipher AES-128-CBC, <ca> tags

Service names:
  dropbear-extra stunnel-extra ssh-ws-extra udp-custom-extra
  openvpn-3x-ui-udp openvpn-3x-ui-tcp

Tune ports before running with env vars:
  DROPBEAR_PORTS, DROPBEAR_BACKEND_PORT, STUNNEL_PORTS, SSHWS_PORT,
  SSH_BACKEND_HOST, SSH_BACKEND_PORT, OPENSSH_PORT, UDP_CUSTOM_PORT, OPENVPN_PORT
EOF_SUMMARY
}

main() {
  need_root
  local pm
  pm="$(detect_pkg_manager)"
  ensure_dirs
  pkg_bootstrap "$pm"

  configure_openssh_backend
  configure_dropbear
  configure_stunnel
  install_ssh_ws
  install_udp_custom
  configure_openvpn

  systemctl daemon-reload
  print_summary
}

main "$@"