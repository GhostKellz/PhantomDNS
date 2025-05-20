pkgname=phantomdns
pkgver=0.1.0
pkgrel=1
pkgdesc="Blazing-fast, modern Go-native DNS resolver and filter engine"
arch=('x86_64')
url="https://github.com/ghostkellz/PhantomDNS"
license=('MIT')
depends=('glibc')
makedepends=('go')
backup=('etc/pdns/config.yaml')
source=("phantomdns::git+https://github.com/ghostkellz/PhantomDNS.git")
sha256sums=('SKIP')

build() {
  cd "$srcdir/phantomdns"
  go build -o phantomdns main.go
}

package() {
  install -Dm755 "$srcdir/phantomdns/phantomdns" "$pkgdir/usr/bin/phantomdns"
  install -d "$pkgdir/etc/pdns"
  install -m644 "$srcdir/phantomdns/config.yaml" "$pkgdir/etc/pdns/config.yaml"
  # Generate certs and root hints if missing (post_install)
  install -Dm644 "$srcdir/phantomdns/server.crt" "$pkgdir/etc/pdns/server.crt"
  install -Dm600 "$srcdir/phantomdns/server.key" "$pkgdir/etc/pdns/server.key"
  # Systemd service and timer
  install -Dm644 "$srcdir/phantomdns/phantomdns.service" "$pkgdir/usr/lib/systemd/system/phantomdns.service"
  install -Dm644 "$srcdir/phantomdns/refresh-roothints.service" "$pkgdir/usr/lib/systemd/system/refresh-roothints.service"
  install -Dm644 "$srcdir/phantomdns/refresh-roothints.timer" "$pkgdir/usr/lib/systemd/system/refresh-roothints.timer"
}

post_install() {
  # Generate certs if missing
  [ -f /etc/pdns/server.crt ] || /usr/bin/phantomdns --generate-certs
  [ -f /etc/pdns/server.key ] || /usr/bin/phantomdns --generate-certs
  # Generate root hints if missing
  [ -f /etc/pdns/root.hints ] || /usr/bin/phantomdns --refresh-roothints
  systemctl daemon-reload
  systemctl enable --now phantomdns.service
  systemctl enable --now refresh-roothints.timer
}
