#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
LOCK_FILE="$ROOT/versions.lock"
SRC_DIR="$ROOT/build/whisper-src"
OUT_DIR="$ROOT/build/whisper"
BINARY_NAME="whisper-cli"

read_lock() {
  local key="$1"
  grep "^${key}=" "$LOCK_FILE" | head -n1 | cut -d= -f2-
}

REPO="$(read_lock whisper_cpp_repo)"
REF="$(read_lock whisper_cpp_ref)"

if [[ -z "$REPO" ]]; then
  echo "whisper_cpp_repo missing in versions.lock" >&2
  exit 1
fi

if [[ "$REF" == "unpinned" || -z "$REF" ]]; then
  echo "whisper_cpp_ref is unpinned; fetching latest tag from $REPO" >&2
  REF="$(git ls-remote --tags "$REPO" | awk -F/ '{print $3}' | sed 's/\^{}//' | sort -V | tail -n1)"
  if [[ -z "$REF" ]]; then
    echo "could not resolve whisper.cpp tag" >&2
    exit 1
  fi
fi

if [[ ! -d "$SRC_DIR/.git" ]]; then
  git clone "$REPO" "$SRC_DIR"
fi

(
  cd "$SRC_DIR"
  git fetch --tags origin
  git checkout "$REF"
)

BUILD_DIR="$SRC_DIR/build"
cmake -S "$SRC_DIR" -B "$BUILD_DIR" -DGGML_METAL=ON -DCMAKE_BUILD_TYPE=Release
cmake --build "$BUILD_DIR" --config Release --target "$BINARY_NAME"

mkdir -p "$OUT_DIR"
cp "$BUILD_DIR/bin/$BINARY_NAME" "$OUT_DIR/$BINARY_NAME"

PINNED="$(git -C "$SRC_DIR" rev-parse HEAD)"
if grep -q '^whisper_cpp_ref=' "$LOCK_FILE"; then
  sed -i '' "s|^whisper_cpp_ref=.*|whisper_cpp_ref=${PINNED}|" "$LOCK_FILE"
else
  printf 'whisper_cpp_ref=%s\n' "$PINNED" >>"$LOCK_FILE"
fi

echo "built $OUT_DIR/$BINARY_NAME at revision $PINNED"
