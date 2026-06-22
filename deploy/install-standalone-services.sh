#!/bin/bash
# install-standalone-services.sh
# Installs prerequisites for X-UI PRO standalone services (OpenVPN, WireGuard, Dropbear)
# Usage: bash install-standalone-services.sh [openvpn|wireguard|dropbear|all]

set -euo pipefail

red='\033[0;31m'; green='\033[0;32m'; blue='\033[0;34m'; yellow='\033[0;33m'; plain='\033[0m'

[[ $EUID -ne 0 ]] && echo -e "${red}Please run as root${plain}" && exit 1

detect_os() {
  if [[ -f /etc/os-release ]]; then
    source /etc/os-release
    echo "$ID"
  elif [[ -f /usr/lib/os-release ]]; then
    source /usr/lib/os-release
    echo "$ID"
  else
    echo "unknown"
  fi
}

OS=$(detect_os)
PKG_MGR=""
INSTALL_CMD=""

case "$OS" in
  ubuntu|debian)
    PKG_MGR="apt-get"
    INSTALL_CMD="apt-get install -y"
    ;;
  centos|rhel|rocky|almalinux)
    PKG_MGR="yum"
    INSTALL_CMD="yum install -y"
    ;;
  fedora)
    PKG_MGR="dnf"
    INSTALL_CMD="dnf install -y"
    ;;
  alpine)
    PKG_MGR="apk"
    INSTALL_CMD="apk add"
    ;;
  *)
    echo -e "${red}Unsupported OS: $OS${plain}"
    exit 1
    ;;
esac

echo -e "${blue}Detected OS: $OS | Package manager: $PKG_MGR${plain}"

install_openvpn() {
  echo -e "${green}Installing OpenVPN...${plain}"
  case "$OS" in
    ubuntu|debian) $INSTALL_CMD openvpn easy-rsa ;;
    centos|rhel|rocky|almalinux|fedora) $INSTALL_CMD openvpn easy-rsa ;;
    alpine) $INSTALL_CMD openvpn easy-rsa ;;
  esac
  mkdir -p /etc/openvpn /var/log/x-ui
  echo -e "${green}OpenVPN installed. Use X-UI PRO panel to configure.${plain}"
}

install_wireguard() {
  echo -e "${green}Installing WireGuard...${plain}"
  case "$OS" in
    ubuntu|debian)
      $INSTALL_CMD wireguard wireguard-tools
      ;;
    centos|rhel|rocky|almalinux|fedora)
      $INSTALL_CMD wireguard-tools
      ;;
    alpine)
      $INSTALL_CMD wireguard-tools
      ;;
  esac
  mkdir -p /etc/wireguard /var/log/x-ui
  echo -e "${green}WireGuard installed. Use X-UI PRO panel to configure.${plain}"
}

install_dropbear() {
  echo -e "${green}Installing Dropbear...${plain}"
  case "$OS" in
    ubuntu|debian) $INSTALL_CMD dropbear ;;
    centos|rhel|rocky|almalinux|fedora) $INSTALL_CMD dropbear ;;
    alpine) $INSTALL_CMD dropbear ;;
  esac
  mkdir -p /etc/dropbear /var/log/x-ui
  echo -e "${green}Dropbear installed. Use X-UI PRO panel to configure.${plain}"
}

install_all() {
  install_openvpn
  install_wireguard
  install_dropbear
}

case "${1:-all}" in
  openvpn) install_openvpn ;;
  wireguard) install_wireguard ;;
  dropbear) install_dropbear ;;
  all) install_all ;;
  *)
    echo "Usage: $0 [openvpn|wireguard|dropbear|all]"
    exit 1
    ;;
esac

echo -e "${green}Done.${plain}"
