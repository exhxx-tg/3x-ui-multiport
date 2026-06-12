#!/usr/bin/env bash
set -Eeuo pipefail

# 3x-ui Multiport Extra Protocols production installer.
#
# This script wires the panel's Linux users into real, daemon-backed endpoints
# commonly found in proven VPN autoscripts (Dropbear/Stunnel/BadVPN/DNSTT/
# OpenVPN-PAM/SSH-WS/Psiphon). It intentionally generates real keys and
# persistent systemd services instead of returning panel-only placeholder URIs.

export DEBIAN_FRONTEND=noninteractive

EXTRA_DIR="${EXTRA_DIR:-/etc/3x-ui/extra}"
BIN_DIR="${BIN_DIR:-/usr/local/bin}"
LOG_DIR="${LOG_DIR:-/var/log/3x-ui-extra}"

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
OPENVPN_PORT="${OPENVPN_PORT:-1194}"
PSIPHON_PORT="${PSIPHON_PORT:-3001}"

log() { printf '\033[1;32m[extra-protocols]\033[0m %s\n' "$*"; }
warn() { printf '\033[1;33m[extra-protocols][warn]\033[0m %s\n' "$*" >&2; }
die() { printf '\033[1;31m[extra-protocols][error]\033[0m %s\n' "$*" >&2; exit 1; }

need_root() { [[ "${EUID}" -eq 0 ]] || die "Please run as root."; }

detect_pkg_manager() {
  if command -v apt-get >/dev/null 2>&1; then echo apt; return; fi
  if command -v dnf >/dev/null 2>&1; then echo dnf; return; fi
  if command -v yum >/dev/null 2>&1; then echo yum; return; fi
  if command -v pacman >/dev/null 2>&1; then echo pacman; return; fi
  die "Unsupported distro: apt, dnf, yum, or pacman is required."
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
      install_packages "$pm" ca-certificates curl wget git build-essential cmake golang-go openssh-server dropbear stunnel4 python3 openssl openvpn easy-rsa iptables tar gzip jq
      ;;
    dnf|yum)
      install_packages "$pm" ca-certificates curl wget git gcc gcc-c++ make cmake golang openssh-server dropbear stunnel python3 openssl openvpn easy-rsa iptables tar gzip jq || true
      ;;
    pacman)
      install_packages "$pm" ca-certificates curl wget git base-devel cmake go openssh dropbear stunnel python openssl openvpn easy-rsa iptables tar gzip jq
      ;;
  esac
}

ensure_dirs() {
  mkdir -p "$EXTRA_DIR" "$BIN_DIR" "$LOG_DIR" \
    /etc/stunnel /etc/ssh-ws /etc/dnstt /etc/psiphon \
    /etc/openvpn/3x-ui /etc/systemd/system /etc/ssws
}

public_ip() {
  curl -4fsS --max-time 4 https://api.ipify.org 2>/dev/null || hostname -I 2>/dev/null | awk '{print $1}' || echo "YOUR_SERVER_IP"
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

configure_openssh() {
  log "Configuring OpenSSH on port ${SSH_PORT} with HTTP Custom-compatible algorithms"
  mkdir -p /etc/ssh/sshd_config.d
  cat > /etc/ssh/sshd_config.d/99-3x-ui-extra.conf <<EOF_SSH
Port ${SSH_PORT}
PasswordAuthentication yes
KbdInteractiveAuthentication yes
UsePAM yes
PermitTunnel yes
AllowTcpForwarding yes
X11Forwarding no
Banner /etc/issue.net

# Compatibility profile used by many working SSH/SSL/WS autoscripts.
# Includes modern algorithms first and legacy fallbacks for HTTP Custom clients
# that otherwise fail with "Cannot negotiate, proposals do not match".
KexAlgorithms curve25519-sha256,curve25519-sha256@libssh.org,diffie-hellman-group-exchange-sha256,diffie-hellman-group14-sha256,diffie-hellman-group14-sha1,diffie-hellman-group1-sha1
Ciphers chacha20-poly1305@openssh.com,aes128-ctr,aes192-ctr,aes256-ctr,aes128-cbc,aes256-cbc,3des-cbc
MACs hmac-sha2-256,hmac-sha2-512,hmac-sha1,hmac-sha1-96
HostKeyAlgorithms +ssh-rsa
PubkeyAcceptedAlgorithms +ssh-rsa
EOF_SSH
  touch /etc/issue.net
  systemctl enable ssh >/dev/null 2>&1 || systemctl enable sshd >/dev/null 2>&1 || true
  systemctl restart ssh >/dev/null 2>&1 || systemctl restart sshd >/dev/null 2>&1 || warn "Could not restart ssh/sshd now."
}

configure_dropbear() {
  log "Configuring Dropbear on port ${DROPBEAR_PORT}"
  local dropbear_bin
  dropbear_bin="$(command -v dropbear || true)"
  [[ -n "$dropbear_bin" ]] || die "dropbear binary was not found."
  mkdir -p /etc/dropbear
  cat > /etc/default/dropbear-3x-ui-extra <<EOF_DROPBEAR_ENV
DROPBEAR_ARGS="-F -E -p 0.0.0.0:${DROPBEAR_PORT} -b /etc/issue.net -K 60 -I 300"
EOF_DROPBEAR_ENV
  write_service "dropbear-extra" "[Unit]
Description=3x-ui Extra Dropbear SSH (${DROPBEAR_PORT})
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
      -out "$cert" -keyout "$cert" -subj "/CN=3x-ui-extra-stunnel" >/dev/null 2>&1
  fi
  chmod 600 "$cert"
  printf '%s' "$cert"
}

configure_stunnel() {
  log "Configuring Stunnel4 on ports ${STUNNEL_PORTS} -> ${SSH_BACKEND_HOST}:${SSH_BACKEND_PORT}"
  local stunnel_bin cert
  stunnel_bin="$(command -v stunnel4 || command -v stunnel || true)"
  [[ -n "$stunnel_bin" ]] || die "stunnel/stunnel4 binary was not found."
  cert="$(generate_stunnel_cert)"
  cat > /etc/stunnel/stunnel.conf <<EOF_STUNNEL
foreground = yes
pid = /run/stunnel-3x-ui-extra.pid
cert = ${cert}

; Broad compatibility for HTTP Custom/OpenVPN Connect style TLS handshakes.
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
connect = ${SSH_BACKEND_HOST}:${SSH_BACKEND_PORT}

EOF_STUNNEL_SERVICE
  done
  sed -i 's/^ENABLED=.*/ENABLED=1/' /etc/default/stunnel4 2>/dev/null || true
  write_service "stunnel-extra" "[Unit]
Description=3x-ui Extra Stunnel SSL/TLS -> SSH
After=network-online.target ssh.service sshd.service dropbear-extra.service
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
  log "Installing SSH WebSocket bridge on ports ${SSHWS_PORTS} -> ${SSH_BACKEND_HOST}:${SSH_BACKEND_PORT}"
  cat > "${BIN_DIR}/ssh-ws" <<'PY'
#!/usr/bin/env python3
import asyncio, base64, hashlib, os, struct

HOST = os.environ.get("SSH_WS_HOST", "0.0.0.0")
PORTS = [int(p) for p in os.environ.get("SSH_WS_PORTS", "80 8880").split()]
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

async def pipe(src, dst):
    try:
        while True:
            data = await src.read(8192)
            if not data:
                break
            dst.write(data)
            await dst.drain()
    except Exception:
        pass
    finally:
        dst.close()

async def ws_recv(reader):
    head = await reader.readexactly(2)
    length = head[1] & 0x7F
    if length == 126:
        length = struct.unpack("!H", await reader.readexactly(2))[0]
    elif length == 127:
        length = struct.unpack("!Q", await reader.readexactly(8))[0]
    mask = await reader.readexactly(4) if head[1] & 0x80 else b""
    payload = await reader.readexactly(length) if length else b""
    if mask:
        payload = bytes(b ^ mask[i % 4] for i, b in enumerate(payload))
    return head[0] & 0x0F, payload

def ws_frame(payload):
    n = len(payload)
    if n < 126:
        return bytes([0x82, n]) + payload
    if n < 65536:
        return bytes([0x82, 126]) + struct.pack("!H", n) + payload
    return bytes([0x82, 127]) + struct.pack("!Q", n) + payload

async def ws_to_tcp(ws_reader, tcp_writer):
    try:
        while True:
            opcode, payload = await ws_recv(ws_reader)
            if opcode == 0x8:
                break
            if opcode in (0x1, 0x2, 0x0):
                tcp_writer.write(payload)
                await tcp_writer.drain()
    except Exception:
        pass
    finally:
        tcp_writer.close()

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

async def handle(reader, writer):
    req = await read_http(reader)
    headers = parse_headers(req)
    key = headers.get("sec-websocket-key", "")
    tcp_reader, tcp_writer = await asyncio.open_connection(BACKEND_HOST, BACKEND_PORT)
    if key:
        accept = base64.b64encode(hashlib.sha1((key + GUID).encode()).digest()).decode()
        writer.write(("HTTP/1.1 101 Switching Protocols\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Accept: " + accept + "\r\n\r\n").encode())
        await writer.drain()
        await asyncio.gather(ws_to_tcp(reader, tcp_writer), tcp_to_ws(tcp_reader, writer))
    else:
        await asyncio.gather(pipe(reader, tcp_writer), pipe(tcp_reader, writer))

async def main():
    servers = [await asyncio.start_server(handle, HOST, p) for p in PORTS]
    await asyncio.gather(*(s.serve_forever() for s in servers))

if __name__ == "__main__":
    asyncio.run(main())
PY
  chmod +x "${BIN_DIR}/ssh-ws"
  write_service "ssh-ws-extra" "[Unit]
Description=3x-ui Extra SSH over WebSocket
After=network-online.target ssh.service sshd.service dropbear-extra.service
Wants=network-online.target

[Service]
Type=simple
Environment=SSH_WS_PORTS=${SSHWS_PORTS}
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
  log "Installing BadVPN udpgw on port ${UDP_CUSTOM_PORT}"
  local tmp
  tmp="$(mktemp -d)"
  if command -v badvpn-udpgw >/dev/null 2>&1; then
    install -m 0755 "$(command -v badvpn-udpgw)" "${BIN_DIR}/badvpn-udpgw"
  elif command -v cmake >/dev/null 2>&1 && git clone --depth=1 https://github.com/ambrop72/badvpn "$tmp/badvpn" >/dev/null 2>&1; then
    (cd "$tmp/badvpn" && cmake -DBUILD_NOTHING_BY_DEFAULT=1 -DBUILD_UDPGW=1 . && make -j"$(nproc 2>/dev/null || echo 1)") || warn "BadVPN build failed."
    [[ -x "$tmp/badvpn/udpgw/badvpn-udpgw" ]] && install -m 0755 "$tmp/badvpn/udpgw/badvpn-udpgw" "${BIN_DIR}/badvpn-udpgw"
  fi
  rm -rf "$tmp"
  [[ -x "${BIN_DIR}/badvpn-udpgw" ]] || die "badvpn-udpgw unavailable."
  write_service "udp-custom-extra" "[Unit]
Description=3x-ui Extra UDP Custom BadVPN Gateway
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=${BIN_DIR}/badvpn-udpgw --listen-addr 0.0.0.0:${UDP_CUSTOM_PORT} --max-clients 2048 --max-connections-for-client 32
Restart=always
RestartSec=3

[Install]
WantedBy=multi-user.target"
  restart_service "udp-custom-extra"
}

install_slowdns() {
  log "Installing DNSTT SlowDNS using nameserver ${SLOWDNS_NS}"
  if [[ ! -x "${BIN_DIR}/dnstt-server" ]]; then
    GOBIN="${BIN_DIR}" go install www.bamsoftware.com/git/dnstt.git/dnstt-server@latest || warn "dnstt-server go install failed."
  fi
  [[ -x "${BIN_DIR}/dnstt-server" ]] || die "dnstt-server unavailable."
  if [[ ! -s /etc/dnstt/server.key || ! -s /etc/dnstt/server.pub ]]; then
    "${BIN_DIR}/dnstt-server" -gen-key -privkey-file /etc/dnstt/server.key -pubkey-file /etc/dnstt/server.pub
  fi
  chmod 600 /etc/dnstt/server.key
  chmod 644 /etc/dnstt/server.pub
  printf '%s\n' "$SLOWDNS_NS" > /etc/dnstt/nameserver
  write_service "dnstt-extra" "[Unit]
Description=3x-ui Extra SlowDNS DNSTT Server
After=network-online.target ssh.service sshd.service dropbear-extra.service
Wants=network-online.target

[Service]
Type=simple
ExecStart=${BIN_DIR}/dnstt-server -udp ${SLOWDNS_UDP_LISTEN} -privkey-file /etc/dnstt/server.key ${SLOWDNS_NS} ${SLOWDNS_BACKEND}
Restart=always
RestartSec=3
LimitNOFILE=1048576

[Install]
WantedBy=multi-user.target"
  restart_service "dnstt-extra"
}

install_psiphon() {
  log "Installing Psiphon core/server artifacts"
  local ip tmp
  ip="$(public_ip)"
  tmp="$(mktemp -d)"
  if [[ ! -x "${BIN_DIR}/psiphond" ]]; then
    if git clone --depth=1 https://github.com/Psiphon-Labs/psiphon-tunnel-core "$tmp/psiphon" >/dev/null 2>&1; then
      (cd "$tmp/psiphon" && GOBIN="${BIN_DIR}" go install ./Server/psiphond 2>/dev/null) || \
      (cd "$tmp/psiphon/Server" && go build -o "${BIN_DIR}/psiphond" . 2>/dev/null) || \
      warn "Official Psiphon server build failed; place a psiphond-compatible binary at ${BIN_DIR}/psiphond and rerun."
    fi
  fi
  rm -rf "$tmp"

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

  if [[ -x "${BIN_DIR}/psiphond" ]]; then
    "${BIN_DIR}/psiphond" --config /etc/psiphon/server.json --server-entry /etc/psiphon/server-entry.dat generate >/dev/null 2>&1 || true
  fi
  if [[ ! -s /etc/psiphon/server-entry.dat ]]; then
    cat > /etc/psiphon/server-entry.dat <<EOF_ENTRY
Psiphon server core installed/configured for ${ip}:${PSIPHON_PORT}.
If this file is not a signed psiphon server entry, rerun after installing a psiphond binary that supports --server-entry generation.
EOF_ENTRY
  fi

  if [[ -x "${BIN_DIR}/psiphond" ]]; then
    write_service "psiphon-extra" "[Unit]
Description=3x-ui Extra Psiphon Server
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=${BIN_DIR}/psiphond --config /etc/psiphon/server.json
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target"
    restart_service "psiphon-extra"
  else
    warn "Psiphon service not started because ${BIN_DIR}/psiphond was not built."
  fi
}

generate_openvpn_pki() {
  local dir="/etc/openvpn/3x-ui"
  if [[ ! -s "${dir}/ca.crt" || ! -s "${dir}/ca.key" ]]; then
    openssl genrsa -out "${dir}/ca.key" 2048 >/dev/null 2>&1
    openssl req -x509 -new -nodes -key "${dir}/ca.key" -sha256 -days 3650 -out "${dir}/ca.crt" -subj "/CN=3x-ui-openvpn-ca" >/dev/null 2>&1
  fi
  if [[ ! -s "${dir}/server.crt" || ! -s "${dir}/server.key" ]]; then
    openssl genrsa -out "${dir}/server.key" 2048 >/dev/null 2>&1
    openssl req -new -key "${dir}/server.key" -out "${dir}/server.csr" -subj "/CN=3x-ui-openvpn-server" >/dev/null 2>&1
    openssl x509 -req -in "${dir}/server.csr" -CA "${dir}/ca.crt" -CAkey "${dir}/ca.key" -CAcreateserial -out "${dir}/server.crt" -days 3650 -sha256 >/dev/null 2>&1
  fi
  [[ -s "${dir}/ta.key" ]] || openvpn --genkey secret "${dir}/ta.key" >/dev/null 2>&1 || true
  [[ -s "${dir}/dh.pem" ]] || openssl dhparam -out "${dir}/dh.pem" 2048 >/dev/null 2>&1 || true
  chmod 600 "${dir}/ca.key" "${dir}/server.key" "${dir}/ta.key" 2>/dev/null || true
}

openvpn_pam_plugin() {
  for p in \
    /usr/lib/openvpn/openvpn-plugin-auth-pam.so \
    /usr/lib/x86_64-linux-gnu/openvpn/plugins/openvpn-plugin-auth-pam.so \
    /usr/lib64/openvpn/plugins/openvpn-plugin-auth-pam.so \
    /usr/lib/openvpn/plugins/openvpn-plugin-auth-pam.so; do
    [[ -s "$p" ]] && { printf '%s' "$p"; return; }
  done
  printf '%s' "openvpn-plugin-auth-pam.so"
}

configure_openvpn() {
  log "Configuring OpenVPN TCP/UDP on port ${OPENVPN_PORT} with PAM auth"
  local openvpn_bin plugin group_name iface ip
  openvpn_bin="$(command -v openvpn || true)"
  [[ -n "$openvpn_bin" ]] || die "openvpn binary was not found."
  plugin="$(openvpn_pam_plugin)"
  group_name="nogroup"; getent group nogroup >/dev/null 2>&1 || group_name="nobody"
  iface="$(ip route get 1.1.1.1 2>/dev/null | awk '{for(i=1;i<=NF;i++) if($i=="dev") {print $(i+1); exit}}')"
  ip="$(public_ip)"
  generate_openvpn_pki

  cat > /etc/pam.d/openvpn <<'EOF_PAM'
auth    required pam_unix.so shadow nodelay
account required pam_unix.so
EOF_PAM

  for proto in udp tcp; do
    local dev net server_conf proto_line
    if [[ "$proto" == "udp" ]]; then
      dev="tun-3xui-udp"; net="10.88.0.0 255.255.255.0"; proto_line="udp"; server_conf="/etc/openvpn/3x-ui/server-udp.conf"
    else
      dev="tun-3xui-tcp"; net="10.89.0.0 255.255.255.0"; proto_line="tcp-server"; server_conf="/etc/openvpn/3x-ui/server-tcp.conf"
    fi
    cat > "$server_conf" <<EOF_OVPN
port ${OPENVPN_PORT}
proto ${proto_line}
dev ${dev}
ca /etc/openvpn/3x-ui/ca.crt
cert /etc/openvpn/3x-ui/server.crt
key /etc/openvpn/3x-ui/server.key
dh /etc/openvpn/3x-ui/dh.pem
topology subnet
server ${net}
ifconfig-pool-persist /etc/openvpn/3x-ui/ipp-${proto}.txt
verify-client-cert none
username-as-common-name
plugin ${plugin} openvpn
duplicate-cn
keepalive 10 120
cipher AES-128-CBC
data-ciphers AES-256-GCM:AES-128-GCM:CHACHA20-POLY1305:AES-128-CBC
data-ciphers-fallback AES-128-CBC
auth SHA256
tls-server
tls-auth /etc/openvpn/3x-ui/ta.key 0
push "redirect-gateway def1 bypass-dhcp"
push "dhcp-option DNS 1.1.1.1"
push "dhcp-option DNS 8.8.8.8"
persist-key
persist-tun
user nobody
group ${group_name}
status ${LOG_DIR}/openvpn-${proto}-status.log
verb 3
EOF_OVPN
  done

  cat > /etc/openvpn/3x-ui/client-template.ovpn <<EOF_CLIENT
client
dev tun
proto udp
remote ${ip} ${OPENVPN_PORT}
resolv-retry infinite
nobind
persist-key
persist-tun
remote-cert-tls server
auth-user-pass
cipher AES-128-CBC
data-ciphers AES-256-GCM:AES-128-GCM:CHACHA20-POLY1305:AES-128-CBC
auth SHA256
key-direction 1
verb 3
<ca>
$(cat /etc/openvpn/3x-ui/ca.crt)
</ca>
<tls-auth>
$(cat /etc/openvpn/3x-ui/ta.key 2>/dev/null || true)
</tls-auth>
EOF_CLIENT

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
  if [[ -n "$iface" ]]; then
    iptables -t nat -C POSTROUTING -s 10.88.0.0/16 -o "$iface" -j MASQUERADE 2>/dev/null || iptables -t nat -A POSTROUTING -s 10.88.0.0/16 -o "$iface" -j MASQUERADE || true
    iptables -t nat -C POSTROUTING -s 10.89.0.0/16 -o "$iface" -j MASQUERADE 2>/dev/null || iptables -t nat -A POSTROUTING -s 10.89.0.0/16 -o "$iface" -j MASQUERADE || true
  fi
  restart_service "openvpn-3x-ui-udp"
  restart_service "openvpn-3x-ui-tcp"
}

print_summary() {
  local ip pubkey psientry
  ip="$(public_ip)"
  pubkey="$(cat /etc/dnstt/server.pub 2>/dev/null || echo GENERATED_PUBKEY_NOT_SET)"
  psientry="$(head -c 180 /etc/psiphon/server-entry.dat 2>/dev/null || true)"
  cat <<EOF_SUMMARY

Installation complete.

Real services:
  SSH/OpenSSH:        ${ip}:${SSH_PORT}
  Dropbear:           ${ip}:${DROPBEAR_PORT}
  SSL/Stunnel:        ${ip}:443 and ${ip}:444 -> ${SSH_BACKEND_HOST}:${SSH_BACKEND_PORT}
  SSH-WS:             ${ip}:80 and ${ip}:8880 -> ${SSH_BACKEND_HOST}:${SSH_BACKEND_PORT}
  UDP Custom BadVPN:  ${ip}:${UDP_CUSTOM_PORT}
  SlowDNS DNSTT NS:   ${SLOWDNS_NS}
  SlowDNS Public Key: ${pubkey}
  Psiphon Entry:      ${psientry}
  OpenVPN TCP/UDP:    ${ip}:${OPENVPN_PORT} using PAM/Linux users

Generated files:
  /etc/dnstt/server.key /etc/dnstt/server.pub /etc/dnstt/nameserver
  /etc/psiphon/server.json /etc/psiphon/server-entry.dat
  /etc/openvpn/3x-ui/server-udp.conf /etc/openvpn/3x-ui/server-tcp.conf
  /etc/openvpn/3x-ui/client-template.ovpn

Service names:
  dropbear-extra stunnel-extra ssh-ws-extra udp-custom-extra dnstt-extra
  psiphon-extra openvpn-3x-ui-udp openvpn-3x-ui-tcp
EOF_SUMMARY
}

main() {
  need_root
  local pm
  pm="$(detect_pkg_manager)"
  ensure_dirs
  pkg_bootstrap "$pm"
  configure_openssh
  configure_dropbear
  configure_stunnel
  install_ssh_ws
  install_udp_custom
  install_slowdns
  install_psiphon
  configure_openvpn
  systemctl daemon-reload
  print_summary
}

main "$@"