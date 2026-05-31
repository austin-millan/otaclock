#!/usr/bin/env bash
# Package a single cross-compiled binary into a distributable archive.
#
#   package-release.sh <platform-label> <version> <binary-path>
#
# Windows binaries (.exe) are zipped; everything else is tar.gz'd. Assets are
# embedded in the binary, so the archive only carries the executable + README.
set -euo pipefail

PLATFORM="${1:?platform label, e.g. linux-amd64}"
VERSION="${2:?release version}"
BIN_SRC="${3:?path to the built binary}"
BINARY_NAME="${BINARY_NAME:-otaclock2}"

if [[ ! -f "$BIN_SRC" ]]; then
  echo "Missing release binary: $BIN_SRC" >&2
  exit 1
fi

release_root="dist/otaclock2-${VERSION}-${PLATFORM}"
mkdir -p "${release_root}" dist

if [[ "$BIN_SRC" == *.exe ]]; then
  cp "$BIN_SRC" "${release_root}/${BINARY_NAME}.exe"
else
  cp "$BIN_SRC" "${release_root}/${BINARY_NAME}"
  chmod +x "${release_root}/${BINARY_NAME}"
fi
cp README.md "${release_root}/" 2>/dev/null || true

if [[ "$PLATFORM" == windows-* ]]; then
  archive="dist/otaclock2-${VERSION}-${PLATFORM}.zip"
  (cd dist && zip -r "$(basename "$archive")" "otaclock2-${VERSION}-${PLATFORM}" >/dev/null)
else
  archive="dist/otaclock2-${VERSION}-${PLATFORM}.tar.gz"
  tar -czf "$archive" -C dist "otaclock2-${VERSION}-${PLATFORM}"
fi

echo "Created ${archive}"
