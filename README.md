# 3x-ui Extended Edition 🚀

A powerful, all-in-one management panel for Xray core and a massive ecosystem of additional tunneling protocols.

## ✨ Features

### 💎 Core Xray Management
Full support for all native Xray protocols (VLESS, VMess, Trojan, Shadowsocks) with advanced routing, load balancing, and real-time traffic monitoring.

### 🌐 All-in-One Multi-Protocol Ecosystem
Unlike standard panels, 3x-ui Extended Edition integrates a unified management system for non-Xray protocols, allowing you to manage users and ports for:

*   **SSH & Dropbear**: High-performance secure shell access.
*   **SSWS**: Secure WebSocket tunnel.
*   **BadVPN (UDPGW)**: Stable UDP forwarding for gaming and VoIP.
*   **Stunnel**: SSL/TLS wrapper for encrypting any TCP protocol.
*   **OpenVPN**: The industry standard for secure VPN tunneling.
*   **Squid Proxy**: High-performance caching and filtering proxy.
*   **SLOW-DNS (dnstt)**: Tunneling over DNS queries for highly restricted networks.
*   **Psiphon**: Robust censorship-circumvention tool.
*   **OHP (Open HTTP Puncher)**: Payload and header injection for net-bypassing.

### 🎨 Server Customization
*   **Dynamic Connection Banner**: Paste your own ASCII art or welcome message in the UI. It is automatically injected into `/etc/issue.net` and loaded by SSH/Dropbear to greet your users upon connection.

## 🚀 Installation

Install the panel and all required dependencies (including standalone binaries and system packages) with a single command:

```bash
bash <(curl -Ls https://raw.githubusercontent.com/YOUR_USERNAME/YOUR_REPO/main/install.sh)
```

## 🛠️ Configuration

1.  **Access the Panel**: Use the URL provided at the end of the installation.
2.  **Extra Protocols**: Navigate to the **Extra Protocols** page to enable specific daemons, set listening ports, and manage users.
3.  **Banner**: Use the **Server Customization** section in the Port Settings tab to set your welcome banner.

## 🛡️ Security
The panel follows security best practices, including:
*   Automatic generation of random administrative credentials.
*   Integration with `acme.sh` for automated Let's Encrypt SSL certificates.
*   Restricted OS user creation for SSH access.
