
BOOTLIN_ARCH="x86-64"
PLATFORM="amd64"
echo "Resolving Bootlin musl toolchain for arch=$BOOTLIN_ARCH"
TARBALL_BASE="https://toolchains.bootlin.com/downloads/releases/toolchains/$BOOTLIN_ARCH/tarballs/"
TARBALL_URL=$(curl -fsSL "$TARBALL_BASE" | grep -oE "${BOOTLIN_ARCH}--musl--stable-[^\"]+\\.tar\\.xz" | sort -r | head -n1)
[ -z "$TARBALL_URL" ] && { echo "Failed to locate Bootlin musl toolchain for arch=$BOOTLIN_ARCH" >&2; exit 1; }
echo "Downloading: $TARBALL_URL"
cd /tmp
curl -fL -sS -o "$(basename "$TARBALL_URL")" "$TARBALL_BASE/$TARBALL_URL"
tar -xf "$(basename "$TARBALL_URL")"
TOOLCHAIN_DIR=$(find . -maxdepth 1 -type d -name "${BOOTLIN_ARCH}--musl--stable-*" | head -n1)
export PATH="$(realpath "$TOOLCHAIN_DIR")/bin:$PATH"
export CC=$(realpath "$(find "$TOOLCHAIN_DIR/bin" -name '*-gcc.br_real' -type f -executable | head -n1)")
[ -z "$CC" ] && { echo "No gcc.br_real found in $TOOLCHAIN_DIR/bin" >&2; exit 1; }
cd -
go build -ldflags "-w -s -linkmode external -extldflags '-static'" -o xui-release -v main.go
file xui-release
ldd xui-release || echo "Static binary confirmed"

mkdir x-ui
cp xui-release x-ui/
cp x-ui.service x-ui/
cp x-ui.sh x-ui/
mv x-ui/xui-release x-ui/x-ui
mkdir x-ui/bin
cd x-ui/bin
Xray_URL="https://github.com/XTLS/Xray-core/releases/download/v25.10.15/"
wget -q ${Xray_URL}Xray-linux-64.zip
unzip Xray-linux-64.zip
rm -f Xray-linux-64.zip
rm -f geoip.dat geosite.dat
wget -q https://github.com/Loyalsoldier/v2ray-rules-dat/releases/latest/download/geoip.dat
wget -q https://github.com/Loyalsoldier/v2ray-rules-dat/releases/latest/download/geosite.dat
wget -q -O geoip_IR.dat https://github.com/chocolate4u/Iran-v2ray-rules/releases/latest/download/geoip.dat
wget -q -O geosite_IR.dat https://github.com/chocolate4u/Iran-v2ray-rules/releases/latest/download/geosite.dat
wget -q -O geoip_RU.dat https://github.com/runetfreedom/russia-v2ray-rules-dat/releases/latest/download/geoip.dat
wget -q -O geosite_RU.dat https://github.com/runetfreedom/russia-v2ray-rules-dat/releases/latest/download/geosite.dat
mv xray "xray-linux-$PLATFORM"
cd ../..
tar -zcvf x-ui-linux-${PLATFORM}.tar.gz x-ui
rm -rf x-ui
rm xui-release