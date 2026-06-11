#!/usr/bin/env bash
#
# Rebuilds the macOS app bundle's icns with a complete set of 1x and 2x
# resolutions from build/appicon.png. Wails' own generator emits only @2x
# slots, which can make macOS show a blank/generic icon in some contexts.
# Run automatically after `make build`; safe to run on non-macOS (no-op).
#
set -euo pipefail
cd "$(dirname "$0")/.."

[ "$(uname)" = "Darwin" ] || exit 0

APP="build/bin/mqtt-manager.app"
SRC="build/appicon.png"
[ -d "$APP" ] || { echo "macicon: no bundle at $APP, skipping"; exit 0; }
[ -f "$SRC" ] || { echo "macicon: no $SRC, skipping"; exit 0; }

work="$(mktemp -d)"
set="$work/icon.iconset"
mkdir -p "$set"

# size:name pairs covering every slot iconutil expects.
for spec in "16:16x16" "32:16x16@2x" "32:32x32" "64:32x32@2x" \
            "128:128x128" "256:128x128@2x" "256:256x256" "512:256x256@2x" \
            "512:512x512"; do
  px="${spec%%:*}"; name="${spec##*:}"
  sips -z "$px" "$px" "$SRC" --out "$set/icon_${name}.png" >/dev/null
done
cp "$SRC" "$set/icon_512x512@2x.png" # 1024x1024

iconutil -c icns -o "$APP/Contents/Resources/iconfile.icns" "$set"
rm -rf "$work"
touch "$APP"
echo "macicon: wrote complete multi-resolution icns to the bundle."
