#!/bin/bash
set -e

# Download latest root hints from Internic
curl -sSL https://www.internic.net/domain/named.root -o /etc/pdns/root.hints
chown root:root /etc/pdns/root.hints
chmod 644 /etc/pdns/root.hints
