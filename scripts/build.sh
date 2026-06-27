#!/usr/bin/env bash
set -euo pipefail

GOOS="${GOOS:-$(go env GOOS)}"
GOARCH="${GOARCH:-$(go env GOARCH)}"
OUT_DIR="${OUT_DIR:-dist}"
SKIP_TESTS="${SKIP_TESTS:-0}"

case "$GOOS" in
  windows) EXT=".dll" ;;
  darwin) EXT=".dylib" ;;
  *) EXT=".so" ;;
esac

ARTIFACT="model-fallback-router-${GOOS}-${GOARCH}${EXT}"
mkdir -p "$OUT_DIR"

if [[ "$SKIP_TESTS" != "1" ]]; then
  CGO_ENABLED=1 GOOS="$GOOS" GOARCH="$GOARCH" go test ./...
fi

CGO_ENABLED=1 GOOS="$GOOS" GOARCH="$GOARCH" go build -trimpath -buildmode=c-shared -ldflags="-s -w" -o "$OUT_DIR/$ARTIFACT" .
echo "Built $OUT_DIR/$ARTIFACT"
