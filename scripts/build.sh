#!/usr/bin/env bash
# Builds Van Link for Windows (installer .exe), macOS and Linux into dist/.
# Needs: Go 1.22+, NSIS (makensis) and the MinGW resource compiler
# (x86_64-w64-mingw32-windres) for the Windows icon.
set -euo pipefail
cd "$(dirname "$0")/.."
VERSION="${VERSION:-0.2.1}"
LDFLAGS="-s -w -X main.version=${VERSION}"
mkdir -p dist

echo "== tests"
go test ./... -count=1

echo "== Windows"
if command -v x86_64-w64-mingw32-windres >/dev/null; then
  (cd packaging/windows && x86_64-w64-mingw32-windres --preprocessor=cpp -O coff -o ../../cmd/vanlink/rsrc_windows_amd64.syso vanlink.rc)
fi
GOOS=windows GOARCH=amd64 go build -trimpath -ldflags "${LDFLAGS} -H windowsgui" -o dist/VanLink.exe ./cmd/vanlink
cp cmd/vanlink/rsrc_windows_amd64.syso cmd/vanlink-obs-panel/ 2>/dev/null || true
GOOS=windows GOARCH=amd64 go build -trimpath -ldflags "${LDFLAGS}" -o dist/vanlink-obs-panel.exe ./cmd/vanlink-obs-panel
rm -f cmd/vanlink/rsrc_windows_amd64.syso cmd/vanlink-obs-panel/rsrc_windows_amd64.syso
if command -v makensis >/dev/null; then
  (cd packaging/windows && makensis -V2 -DVERSION="${VERSION}" installer.nsi && makensis -V2 -DVERSION="${VERSION}" obs-panel.nsi)
else
  echo "makensis not found: skipping the installer (dist/VanLink.exe still works on its own)"
fi

echo "== macOS"
GOOS=darwin GOARCH=arm64 go build -trimpath -ldflags "${LDFLAGS}" -o dist/vanlink-macos-apple-silicon ./cmd/vanlink
GOOS=darwin GOARCH=amd64 go build -trimpath -ldflags "${LDFLAGS}" -o dist/vanlink-macos-intel ./cmd/vanlink

echo "== Linux"
GOOS=linux GOARCH=amd64 go build -trimpath -ldflags "${LDFLAGS}" -o dist/vanlink-linux-x64 ./cmd/vanlink

ls -lh dist
