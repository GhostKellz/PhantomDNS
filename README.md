# PhantomDNS

![Go Version](https://img.shields.io/badge/go-1.22+-blue)
![License](https://img.shields.io/github/license/ghostkellz/PhantomDNS)
![Build](https://img.shields.io/github/actions/workflow/status/ghostkellz/PhantomDNS/go.yml?label=build)
![Status](https://img.shields.io/badge/status-early--dev-yellow)

> A blazing-fast, modern, Go-native DNS resolver and filter engine. Built for speed, simplicity, and control — no unbound or BIND required.

## ✨ Features

- Recursive DNS resolution (forwarding, caching)
- Ad/tracker blocking with hostlist support
- DNS-over-TLS / DNS-over-HTTPS (planned)
- Authoritative DNS support (planned)
- Lightweight Web UI and YAML config
- Written entirely in Go (no system dependencies)

## 🔧 Getting Started

```bash
go run main.go
```

Listens on `:53` UDP by default (requires root or CAP_NET_BIND_SERVICE).

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
