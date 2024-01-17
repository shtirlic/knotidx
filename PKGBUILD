# Maintainer: Serg Podtynnyi <serg@podtynnyi.com>
pkgname=knotd
pkgver=0.1
pkgrel=1
epoch=
pkgdesc=""
arch=(x86_64)
url="https://github.com/shtirlic/knot"
license=('GPL')
groups=()
depends=()
makedepends=('git' 'go')
checkdepends=()
optdepends=()
provides=()
conflicts=()
replaces=()
backup=()
options=()
install=
changelog=
source=("git://github.com/shtirlic/knot")
noextract=()
md5sums=()
validpgpkeys=()

pkgver() {
    cd "$srcdir/"
    printf "r%s.%s" "$(git rev-list --count HEAD)" "$(git rev-parse --short HEAD)"
}

prepare() {
	cd "$pkgname-$pkgver"
	patch -p1 -i "$srcdir/$pkgname-$pkgver.patch"
}

build() {
	cd "$pkgname-$pkgver"
	 go build ./cmd/knotd
}

check() {
	cd "$pkgname-$pkgver"
	# make -k check
}

package() {
	cd "$pkgname-$pkgver"
	install -Dm755 "$srcdir/package/knotd" "$pkgdir/usr/bin/knotd"
}
