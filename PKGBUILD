# Maintainer: Serg Podtynnyi <serg@podtynnyi.com>
pkgname=knotd
pkgver=r15.767cad1
pkgrel=1
pkgdesc=""
arch=('1686' 'x86_64' 'armv7h' 'armv6h' 'aarch64')
url="https://github.com/shtirlic/knot"
license=('GPL')
depends=()
makedepends=('git' 'go')
checkdepends=()
optdepends=()
# install=${pkgname}.install
changelog=
source=("${pkgname}-${pkgver}::git+ssh://git@github.com/shtirlic/knot")
noextract=()
md5sums=('SKIP')
validpgpkeys=()

pkgver() {
  cd "${pkgname}-${pkgver}"
  printf "r%s.%s" "$(git rev-list --count HEAD)" "$(git rev-parse --short HEAD)"
}

build() {
	cd "${pkgname}-${pkgver}"
	export CGO_CPPFLAGS="${CPPFLAGS}"
  export CGO_CFLAGS="${CFLAGS}"
  export CGO_CXXFLAGS="${CXXFLAGS}"
	export GOFLAGS="-buildmode=pie -trimpath -mod=readonly -modcacherw"
	go build -o knotd -ldflags "-extldflags ${LDFLAGS} -s -w -X main.version=${pkgver}" cmd/knotd/*.go
	go build -o knotctl -ldflags "-extldflags ${LDFLAGS} -s -w -X main.version=${pkgver}" cmd/knotctl/*.go
}

package() {
  install -Dm755 "${pkgname}-${pkgver}/knotd" ${pkgdir}/usr/bin/knotd
	install -Dm755 "${pkgname}-${pkgver}/knotctl" ${pkgdir}/usr/bin/knotctl
}
