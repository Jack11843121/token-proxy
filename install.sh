#!/usr/bin/env bash
# =============================================================================
# token-proxy 一键安装脚本（Linux）
#   sudo bash install.sh
# 下载预构建静态二进制（已内置前端界面与 SQLite 数据），注册 systemd 并启动。
# 支持 Ubuntu / Debian / CentOS / RHEL；无需 Docker、无需数据库、无需 Node。
# =============================================================================
set -euo pipefail

DOWNLOAD_BASE="${TOKEN_PROXY_DOWNLOAD:-http://140.143.248.224:8000/releases}"
INSTALL_DIR="${TOKEN_PROXY_INSTALL_DIR:-/opt/token-proxy}"
PORT="${TOKEN_PROXY_PORT:-3000}"
ADMIN_USER="${TOKEN_PROXY_ADMIN_USER:-root}"
ADMIN_PASS="${TOKEN_PROXY_ADMIN_PASS:-123456}"

info() { echo -e "\033[36m==>\033[0m $*"; }
ok()   { echo -e "\033[32m>> $*\033[0m"; }
err()  { echo -e "\033[31m!! $*\033[0m" >&2; }

[ "$(id -u)" = "0" ] || { err "请用 root 运行：sudo bash install.sh"; exit 1; }
command -v curl >/dev/null 2>&1 || { err "需要 curl 才能下载二进制"; exit 1; }

ARCH="$(uname -m)"
case "$ARCH" in
  x86_64|amd64)    BIN=token-proxy-linux-amd64 ;;
  aarch64|arm64)   BIN=token-proxy-linux-arm64 ;;
  *) err "不支持的架构：$ARCH"; exit 1 ;;
esac

mkdir -p "$INSTALL_DIR"
cd "$INSTALL_DIR"

if [ -f "$INSTALL_DIR/token-proxy" ] && [ "${FORCE_BUILD:-0}" != "1" ]; then
  info "已存在 token-proxy 二进制，跳过下载（设置 FORCE_BUILD=1 强制重新下载）"
else
  info "下载预构建二进制：$DOWNLOAD_BASE/$BIN.tar.gz"
  curl -fsSL "$DOWNLOAD_BASE/$BIN.tar.gz" -o "/tmp/$BIN.tar.gz"
  tar xzf "/tmp/$BIN.tar.gz" -C /tmp
  chmod +x "/tmp/$BIN"
  mv -f "/tmp/$BIN" "$INSTALL_DIR/token-proxy"
  rm -f "/tmp/$BIN.tar.gz"
  ok "二进制已就绪：$INSTALL_DIR/token-proxy"
fi

if [ ! -f "$INSTALL_DIR/token-proxy.env" ]; then
  SECRET="$(tr -dc 'A-Za-z0-9' </dev/urandom | head -c 32)"
  cat > "$INSTALL_DIR/token-proxy.env" <<EOF
TOKEN_PROXY_LISTEN=:${PORT}
TOKEN_PROXY_DATA_DIR=${INSTALL_DIR}/data
TOKEN_PROXY_ADMIN_USER=${ADMIN_USER}
TOKEN_PROXY_ADMIN_PASS=${ADMIN_PASS}
TOKEN_PROXY_JWT_SECRET=${SECRET}
EOF
  ok "已生成 $INSTALL_DIR/token-proxy.env —— 请尽快修改 ADMIN_PASS 与 JWT_SECRET"
fi

cat > /etc/systemd/system/token-proxy.service <<EOF
[Unit]
Description=token-proxy - multi-model LLM token gateway
After=network.target

[Service]
Type=simple
WorkingDirectory=$INSTALL_DIR
EnvironmentFile=$INSTALL_DIR/token-proxy.env
ExecStart=$INSTALL_DIR/token-proxy
Restart=always
RestartSec=3
LimitNOFILE=65535

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable --now token-proxy

IP="$(hostname -I 2>/dev/null | awk '{print $1}')"
[ -z "$IP" ] && IP="<服务器IP>"

cat <<EOF

========================================================
  🌉 token-proxy 安装完成！
--------------------------------------------------------
  管理后台     http://${IP}:${PORT}    账号 ${ADMIN_USER} / ${ADMIN_PASS}
  中继接口     http://${IP}:${PORT}/v1  （OpenAI 兼容）
  健康检查     http://${IP}:${PORT}/health

  三步接入：
    1) 管理后台 → 「渠道管理」新增各厂商渠道（填各自的 API Key）
    2) 「用户管理」新增用户并充值额度
    3) 「令牌管理」生成 sk- API Key，发给使用者即可调用

  查看日志： journalctl -u token-proxy -f
  重启服务： systemctl restart token-proxy
========================================================
EOF