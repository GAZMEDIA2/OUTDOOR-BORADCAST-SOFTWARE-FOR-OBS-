#!/usr/bin/env bash
# Installs (or updates) the Van Link relay on a Linux server and starts it.
#   curl -fsSL https://raw.githubusercontent.com/GAZMEDIA2/OUTDOOR-BORADCAST-SOFTWARE-FOR-OBS-/downloads/relay/install.sh | sudo bash
# It only adds its own program, user and service, and opens UDP port 47800.
# Anything else on the server (other apps, web servers) is left alone.
set -euo pipefail
PORT=47800
BASE="https://raw.githubusercontent.com/GAZMEDIA2/OUTDOOR-BORADCAST-SOFTWARE-FOR-OBS-/downloads/relay"

if [ "$(id -u)" != 0 ]; then echo "Run it with sudo (or as root)."; exit 1; fi
case "$(uname -m)" in
  x86_64|amd64) ARCH=amd64 ;;
  aarch64|arm64) ARCH=arm64 ;;
  *) echo "Unsupported processor: $(uname -m)"; exit 1 ;;
esac

echo "== Downloading the Van Link relay ($ARCH)"
curl -fsSL "$BASE/vanlink-relay-linux-$ARCH" -o /usr/local/bin/vanlink-relay.new
chmod 755 /usr/local/bin/vanlink-relay.new
mv -f /usr/local/bin/vanlink-relay.new /usr/local/bin/vanlink-relay

id vanlink-relay >/dev/null 2>&1 || useradd --system --no-create-home --shell /usr/sbin/nologin vanlink-relay 2>/dev/null \
  || useradd --system --no-create-home --shell /sbin/nologin vanlink-relay

cat > /etc/systemd/system/vanlink-relay.service <<UNIT
[Unit]
Description=Van Link relay
After=network-online.target
Wants=network-online.target

[Service]
User=vanlink-relay
ExecStart=/usr/local/bin/vanlink-relay -listen :$PORT
Restart=always
RestartSec=2
NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=true
PrivateTmp=true

[Install]
WantedBy=multi-user.target
UNIT

echo "== Opening UDP port $PORT"
if command -v ufw >/dev/null && ufw status 2>/dev/null | grep -q "Status: active"; then
  ufw allow $PORT/udp
elif command -v firewall-cmd >/dev/null && firewall-cmd --state >/dev/null 2>&1; then
  firewall-cmd --permanent --add-port=$PORT/udp && firewall-cmd --reload
elif command -v csf >/dev/null; then
  echo "CSF firewall found: add $PORT to UDP_IN in /etc/csf/csf.conf, then run: csf -r"
fi

systemctl daemon-reload
systemctl enable vanlink-relay >/dev/null 2>&1
systemctl restart vanlink-relay
sleep 1
if systemctl is-active --quiet vanlink-relay; then
  IP=$(curl -fsS4 --max-time 5 https://api.ipify.org || echo "this server's IP")
  echo
  echo "Van Link relay is running on UDP port $PORT."
  echo "Server address: $IP"
  echo "Status: systemctl status vanlink-relay    Log: journalctl -u vanlink-relay -f"
else
  echo "The relay did not start. Log:"; journalctl -u vanlink-relay -n 30 --no-pager; exit 1
fi
