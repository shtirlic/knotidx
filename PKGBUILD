# Maintainer: Serg Podtynnyi <serg@podtynnyi.com>
pkgname=knotidx-git
pkgver=0.1.r16.7bdb293
pkgrel=1
pkgdesc=""
arch=('1686' 'x86_64' 'armv7h' 'armv6h' 'aarch64')
url="https://github.com/shtirlic/knotidx"
source=("${pkgname}::git+ssh://git@github.com/shtirlic/knotidx")
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
  cd "${srcdir}/${pkgname}"
  (
    set -o pipefail
    git describe --long --tags 2> /dev/null | sed "s/^[A-Za-z\.\-]*//;s/\([^-]*-\)g/r\1/;s/-/./g" ||
    printf "r%s.%s\n" "$(git rev-list --count HEAD)" "$(git rev-parse --short HEAD)"
  )
}

build() {
  cd "${srcdir}/${pkgname}"
	export CGO_CPPFLAGS="${CPPFLAGS}"
  export CGO_CFLAGS="${CFLAGS}"
  export CGO_CXXFLAGS="${CXXFLAGS}"
	export GOFLAGS="-buildmode=pie -trimpath -mod=readonly -modcacherw"
	go build -o knotidx -ldflags "-s -w -X main.version=${pkgver}  -X main.date=$(date -u +%Y%m%d.%H%M%S) -X main.commit=$(git rev-parse --short HEAD)" cmd/knotidx/*
}

package() {
	cd "${srcdir}/${pkgname}"
  install -Dm755 "knotidx" ${pkgdir}/usr/bin/knotidx
}
