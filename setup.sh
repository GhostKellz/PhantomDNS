#!/bin/bash
set -e

# Build PhantomDNS
GO111MODULE=on go build -o phantomdns main.go

# Install binary and config
install -Dm755 phantomdns /usr/bin/phantomdns
install -d /etc/pdns
install -m644 config.yaml /etc/pdns/config.yaml

# Generate certs if missing
if [ ! -f /etc/pdns/server.crt ] || [ ! -f /etc/pdns/server.key ]; then
  /usr/bin/phantomdns --generate-certs
fi

# Generate root hints if missing
if [ ! -f /etc/pdns/root.hints ]; then
  /usr/bin/phantomdns --refresh-roothints
fi

# Install systemd service and timer
install -Dm644 phantomdns.service /usr/lib/systemd/system/phantomdns.service
install -Dm644 refresh-roothints.service /usr/lib/systemd/system/refresh-roothints.service
install -Dm644 refresh-roothints.timer /usr/lib/systemd/system/refresh-roothints.timer

# Enable and start services
systemctl daemon-reload
systemctl enable --now phantomdns.service
systemctl enable --now refresh-roothints.timer

echo "PhantomDNS installed and running!"
