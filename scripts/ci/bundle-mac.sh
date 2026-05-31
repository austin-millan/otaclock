#!/usr/bin/env bash
# Wrap a built otaclock2 binary in a macOS .app bundle using Resources/Info.plist.
#
#   bundle-mac.sh <binary-path> <output.app>
set -euo pipefail

BIN_SRC="${1:?path to the built binary}"
APP_DIR="${2:?output .app path}"
APP_NAME="$(basename "${APP_DIR%.app}")"

if [[ ! -f "$BIN_SRC" ]]; then
  echo "Missing binary: $BIN_SRC" >&2
  exit 1
fi

rm -rf "$APP_DIR"
mkdir -p "$APP_DIR/Contents/MacOS" "$APP_DIR/Contents/Resources"
cp "$BIN_SRC" "$APP_DIR/Contents/MacOS/$APP_NAME"
chmod +x "$APP_DIR/Contents/MacOS/$APP_NAME"
cp Resources/Info.plist "$APP_DIR/Contents/Info.plist"

echo "App bundle → $APP_DIR"
