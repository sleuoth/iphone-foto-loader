#!/bin/bash
# Smoke test: run iphone-loader against the stub helper.
# Verifies the full Go flow works without a real iPhone.

set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
TMPDIR=$(mktemp -d)
trap "rm -rf $TMPDIR" EXIT

echo "=== Building iphone-loader ==="
cd "$REPO_ROOT"
go build -o "$TMPDIR/iphone-loader" .

echo "=== Setting up stub helper ==="
cp "$SCRIPT_DIR/stub-helper.sh" "$TMPDIR/iphone-ic-helper"
chmod +x "$TMPDIR/iphone-ic-helper"

echo "=== Setting up fake exiftool ==="
cat > "$TMPDIR/exiftool" << 'EXIF'
#!/bin/bash
# Return Apple make for files with IMG_ prefix, empty for others
LAST_ARG="${@: -1}"
if [[ "$LAST_ARG" == *"IMG_"* ]]; then
  echo '[{"Make":"Apple","Model":"iPhone 15 Pro","DateTimeOriginal":"2026:06:27 10:30:00"}]'
else
  echo '[{"Make":"","Model":"","DateTimeOriginal":""}]'
fi
EXIF
chmod +x "$TMPDIR/exiftool"

echo "=== Setting up fake sips ==="
cat > "$TMPDIR/sips" << 'SIPS'
#!/bin/bash
OUT="${@: -1}"
echo "fake-jpeg" > "$OUT"
SIPS
chmod +x "$TMPDIR/sips"

echo "=== Creating config ==="
mkdir -p "$TMPDIR/config"
cat > "$TMPDIR/config/config.toml" << TOML
[helper]
path = "$TMPDIR/iphone-ic-helper"

[devices."stub-uuid-001"]
name = "Stub iPhone"
target = "$TMPDIR/archive"
TOML

echo "=== Running iphone-loader ==="
export PATH="$TMPDIR:$PATH"
export IPHONE_IC_HELPER="$TMPDIR/iphone-ic-helper"
"$TMPDIR/iphone-loader" --config "$TMPDIR/config/config.toml"

echo "=== Checking results ==="
echo "--- Archive structure: ---"
find "$TMPDIR/archive" -type f | sort

echo ""
echo "=== Smoke test complete ==="
