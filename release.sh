#!/usr/bin/env bash
# Release build script. Cross-compiles the app for every platform and
# architecture reachable from the current host and puts the binaries
# into build/bin/ as translator-<os>-<arch>[.exe].
#
# Wails apps cannot target every OS from a single machine because the
# webview bindings use cgo on linux and macOS:
#   - linux builds need webkit2gtk headers, so they build only on linux,
#     and only for the native arch unless a cross cgo toolchain plus
#     foreign-arch webkit2gtk libraries are installed
#   - darwin builds need the macOS SDK, so they build only on macOS
#   - windows builds are pure Go and cross-compile from any host
# Run the script on linux and on a Mac to cover the full matrix.
#
# Binaries are compressed with upx --best --lzma where upx supports the
# format (linux, windows/amd64). Set UPX=0 to skip compression.

set -euo pipefail
cd "$(dirname "$0")"

OUT_DIR="build/bin"
UPX="${UPX:-1}"
LDFLAGS="-s -w"
# The first wails build compiles the frontend; later targets reuse it.
SKIP_FRONTEND=""

built=()
skipped=()

build_target() {
    local os="$1" arch="$2" upx_ok="$3"
    local out="translator-${os}-${arch}"
    [ "$os" = windows ] && out="${out}.exe"

    local tags=()
    # webkit2gtk-4.1 is the documented linux dependency (see Makefile)
    [ "$os" = linux ] && tags=(-tags webkit2_41)

    echo "==> building ${os}/${arch}"
    wails build -platform "${os}/${arch}" -o "$out" -ldflags "$LDFLAGS" \
        ${tags[@]+"${tags[@]}"} ${SKIP_FRONTEND:+$SKIP_FRONTEND}
    SKIP_FRONTEND="-s"

    if [ "$UPX" = 1 ] && [ "$upx_ok" = yes ]; then
        if command -v upx >/dev/null 2>&1; then
            echo "==> compressing ${out} with upx"
            upx --best --lzma "${OUT_DIR}/${out}"
        else
            echo "==> upx not installed, skipping compression of ${out}"
        fi
    fi
    built+=("${OUT_DIR}/${out}")
}

command -v wails >/dev/null 2>&1 || {
    echo "error: wails CLI not found in PATH" >&2
    exit 1
}

rm -f "${OUT_DIR}"/translator-*

host_os="$(uname -s)"
case "$host_os" in
Linux)
    build_target linux "$(go env GOARCH)" yes
    skipped+=("linux (other arches): needs a cross cgo toolchain and foreign-arch webkit2gtk")
    skipped+=("darwin/amd64, darwin/arm64: need the macOS SDK, run this script on a Mac")
    ;;
Darwin)
    # upx is skipped on darwin: packed binaries are killed by macOS
    # code-signing checks on modern systems
    build_target darwin amd64 no
    build_target darwin arm64 no
    skipped+=("linux: needs webkit2gtk headers, run this script on a linux machine")
    ;;
*)
    echo "error: unsupported host OS: ${host_os}" >&2
    exit 1
    ;;
esac

build_target windows amd64 yes
# upx cannot pack arm64 PE files, so no compression here
build_target windows arm64 no

echo
echo "=== release binaries ==="
ls -lh "${built[@]}"
if [ "${#skipped[@]}" -gt 0 ]; then
    echo
    echo "=== skipped targets ==="
    printf '%s\n' "${skipped[@]}"
fi
