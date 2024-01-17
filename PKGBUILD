# Maintainer: Serg Podtynnyi <serg@podtynnyi.com>
pkgname=knotd
pkgver=0.1.r0.648bde0
pkgrel=1
pkgdesc=""
arch=('1686' 'x86_64' 'armv7h' 'armv6h' 'aarch64')
url="https://github.com/shtirlic/knot"
source=("${pkgname}-git::git+ssh://git@github.com/shtirlic/knot")
license=('GPL')
depends=()
makedepends=('git' 'go')
checkdepends=()
optdepends=()
# install=${pkgname}.install
changelog=
noextract=()
md5sums=('SKIP')
validpgpkeys=()

pkgver() {
  cd "${srcdir}/${pkgname}-git"
  (
    set -o pipefail
    git describe --long --tags 2> /dev/null | sed "s/^[A-Za-z\.\-]*//;s/\([^-]*-\)g/r\1/;s/-/./g" ||
    printf "r%s.%s\n" "$(git rev-list --count HEAD)" "$(git rev-parse --short HEAD)"
  )
}

build() {
	cd "${pkgname}-${pkgver}"
	export CGO_CPPFLAGS="${CPPFLAGS}"
  export CGO_CFLAGS="${CFLAGS}"
  export CGO_CXXFLAGS="${CXXFLAGS}"
	export GOFLAGS="-buildmode=pie -trimpath -mod=readonly -modcacherw"
	go build -o knotd -ldflags "-extldflags ${LDFLAGS} -s -w -X main.version=${pkgver}  -X main.date=$(date -u +%Y%m%d.%H%M%S) -X main.commit=$(git rev-parse --short HEAD)" cmd/knotd/*
	go build -o knotctl -ldflags "-extldflags ${LDFLAGS} -s -w -X main.version=${pkgver}  -X main.date=$(date -u +%Y%m%d.%H%M%S) -X main.commit=$(git rev-parse --short HEAD)" cmd/knotctl/*
}

package() {
  install -Dm755 "${pkgname}-${pkgver}/knotd" ${pkgdir}/usr/bin/knotd
	install -Dm755 "${pkgname}-${pkgver}/knotctl" ${pkgdir}/usr/bin/knotctl
}
