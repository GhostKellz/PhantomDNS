# PhantomDNS

![Go Version](https://img.shields.io/badge/go-1.22+-blue)
![Docker Ready](https://img.shields.io/badge/docker-image--planned-blue)
![DNSSEC Planned](https://img.shields.io/badge/DNSSEC-support--planned-lightgrey)
[![Go Report Card](https://goreportcard.com/badge/github.com/ghostkellz/PhantomDNS)](https://goreportcard.com/report/github.com/ghostkellz/PhantomDNS)
![Last Commit](https://img.shields.io/github/last-commit/ghostkellz/PhantomDNS)
![Status](https://img.shields.io/badge/status-early--dev-yellow)

> A blazing-fast, modern, Go-native DNS resolver and filter engine. Built for speed, simplicity, and control — no unbound or BIND required.
--- 
## ✨ Features

- Recursive DNS resolution (forwarding, caching)
- Ad/tracker blocking with hostlist support
- DNS-over-TLS / DNS-over-HTTPS (planned)
- Authoritative DNS support (planned)
- Lightweight Web UI and YAML config
- Written entirely in Go (no system dependencies)
- Optional Web UI served on port 5380 — designed for performance-conscious deployments.

---

## 📦 Installation

### Arch Linux (PKGBUILD)

1. Clone the repo and build the package:
   ```sh
   git clone https://github.com/ghostkellz/PhantomDNS.git
   cd PhantomDNS
   makepkg -si
   ```
2. Configuration, certs, and root hints will be installed to `/etc/pdns/`.
3. Systemd services and timers for PhantomDNS and daily root hints refresh will be installed and enabled.
4. On first install, `server.crt` and `server.key` will be generated if missing.

### Debian/Ubuntu (.deb)

1. Clone the repo and build the .deb package:
   ```sh
   git clone https://github.com/ghostkellz/PhantomDNS.git
   cd PhantomDNS
   ./build-deb.sh
   sudo dpkg -i phantomdns_*.deb
   ```
2. Configuration, certs, and root hints will be installed to `/etc/pdns/`.
3. Systemd services and timers for PhantomDNS and daily root hints refresh will be installed and enabled.
4. On first install, `server.crt` and `server.key` will be generated if missing.

### Manual/Portable Install (Optional)

1. Run the provided `setup.sh` script:
   ```sh
   ./setup.sh
   ```
2. This will build PhantomDNS, copy files to `/etc/pdns/`, generate certs/root hints if missing, and enable/start the systemd service/timer.

---

## 🔄 Root Hints Auto-Refresh
- A script and systemd timer/cronjob will keep root hints up to date daily in `/etc/pdns/`.

---

## 🚀 Quick Start

```bash
go run main.go
```

Listens on `:53` UDP by default (requires root or CAP_NET_BIND_SERVICE).
---
## 🧪 Commands 

```bash
phantom -r   # Restart and reload configuration
phantom -u   # Update blocklists from URLs
phantom -s   # Show DNS server status
phantom -c   # Clear in-memory DNS cache
```

---
## 🧩 Roadmap

- [x] UDP resolver w/ upstream fallback
- [ ] Config loader
- [ ] Cache layer (go-cache or ristretto)
- [ ] Blocking engine
- [ ] HTTP API + Web UI
- [ ] DoH / DoT support
- [ ] Auth DNS zone files
- [ ] Docker + systemd service

---

Built by [@ghostkellz](https://github.com/ghostkellz) — part of the **CK Technology Stack**, maintained by **CK Technology LLC**.
