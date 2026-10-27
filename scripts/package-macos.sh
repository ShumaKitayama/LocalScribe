#!/usr/bin/env bash
# arm64 の LocalScribe.app と CLI をビルドし、whisper-cli を同梱する。
# FFmpeg はライセンスを確認し、GPL ビルドは同梱せず報告する。
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
LOCK_FILE="$ROOT/versions.lock"
DIST_DIR="$ROOT/build/dist"
APP_PATH="$ROOT/build/bin/LocalScribe.app"
BUNDLE_BIN="$APP_PATH/Contents/Resources/bin"
WHISPER_SRC="$ROOT/build/whisper/whisper-cli"
REPORT="$DIST_DIR/packaging-report.txt"
CLI_OUT="$DIST_DIR/localscribe"

read_lock() {
  local key="$1"
  grep "^${key}=" "$LOCK_FILE" | head -n1 | cut -d= -f2- || true
}

write_lock_kv() {
  local key="$1"
  local value="$2"
  if grep -q "^${key}=" "$LOCK_FILE"; then
    sed -i '' "s|^${key}=.*|${key}=${value}|" "$LOCK_FILE"
  else
    printf '%s=%s\n' "$key" "$value" >>"$LOCK_FILE"
  fi
}

if [[ "$(uname -m)" != "arm64" ]]; then
  echo "package-macos.sh は arm64 の macOS 上で実行してください (got $(uname -m))" >&2
  exit 1
fi

mkdir -p "$DIST_DIR"
: >"$REPORT"

if [[ ! -x "$WHISPER_SRC" ]]; then
  echo "whisper-cli がありません。先に scripts/build-engine.sh を成功させてください。" >&2
  echo "cmake が無いときは brew install cmake が必要です。" >&2
  exit 1
fi

echo "==> frontend build"
(
  cd "$ROOT/frontend"
  npm run build
)

echo "==> wails build (darwin/arm64)"
(
  cd "$ROOT"
  export GOOS=darwin
  export GOARCH=arm64
  export CGO_ENABLED=1
  wails build -platform darwin/arm64
)

if [[ ! -d "$APP_PATH" ]]; then
  echo "app not found at $APP_PATH" >&2
  exit 1
fi

mkdir -p "$BUNDLE_BIN"
install -m 755 "$WHISPER_SRC" "$BUNDLE_BIN/whisper-cli"
echo "bundled whisper-cli -> $BUNDLE_BIN/whisper-cli" | tee -a "$REPORT"

WHISPER_REV="$(read_lock whisper_cpp_ref)"
if [[ -n "$WHISPER_REV" && "$WHISPER_REV" != "unpinned" ]]; then
  echo "whisper.cpp revision: $WHISPER_REV" | tee -a "$REPORT"
fi

FFMPEG_CANDIDATE="${LOCALSCRIBE_PACKAGE_FFMPEG:-}"
if [[ -z "$FFMPEG_CANDIDATE" ]]; then
  FFMPEG_CANDIDATE="$(command -v ffmpeg || true)"
fi

bundle_ffmpeg=no
ffmpeg_source=system-path
if [[ -n "$FFMPEG_CANDIDATE" && -x "$FFMPEG_CANDIDATE" ]]; then
  ffmpeg_source="$FFMPEG_CANDIDATE"
  VERSION_LOG="$DIST_DIR/ffmpeg-version.txt"
  "$FFMPEG_CANDIDATE" -version >"$VERSION_LOG" 2>&1 || true
  echo "ffmpeg -version saved to $VERSION_LOG" | tee -a "$REPORT"
  cat "$VERSION_LOG" >>"$REPORT"

  if grep -Eiq 'enable-gpl|--enable-gpl|libx264|libx265' "$VERSION_LOG"; then
    echo "FFmpeg は GPL 系の構成です。同梱をスキップします: $FFMPEG_CANDIDATE" | tee -a "$REPORT"
    write_lock_kv ffmpeg_bundle_status not_bundled_gpl
  else
    FFPROBE_CANDIDATE="${LOCALSCRIBE_PACKAGE_FFPROBE:-}"
    if [[ -z "$FFPROBE_CANDIDATE" ]]; then
      FFPROBE_CANDIDATE="$(command -v ffprobe || true)"
    fi
    if [[ -n "$FFPROBE_CANDIDATE" && -x "$FFPROBE_CANDIDATE" ]]; then
      install -m 755 "$FFMPEG_CANDIDATE" "$BUNDLE_BIN/ffmpeg"
      install -m 755 "$FFPROBE_CANDIDATE" "$BUNDLE_BIN/ffprobe"
      bundle_ffmpeg=yes
      echo "FFmpeg / FFprobe を同梱しました (LGPL 相当と判断): $FFMPEG_CANDIDATE" | tee -a "$REPORT"
      write_lock_kv ffmpeg_bundle_status bundled
    else
      echo "ffprobe が見つからず FFmpeg を同梱できません" | tee -a "$REPORT"
      write_lock_kv ffmpeg_bundle_status not_bundled_missing_ffprobe
    fi
  fi
  write_lock_kv ffmpeg_probe_source "$ffmpeg_source"
else
  echo "ffmpeg が見つかりません。同梱はスキップします。" | tee -a "$REPORT"
  write_lock_kv ffmpeg_bundle_status not_bundled_missing
  write_lock_kv ffmpeg_probe_source none
fi

echo "==> go build CLI (arm64)"
(
  cd "$ROOT"
  export GOOS=darwin
  export GOARCH=arm64
  go build -o "$CLI_OUT" ./cmd/localscribe
)
chmod +x "$CLI_OUT"
echo "CLI -> $CLI_OUT" | tee -a "$REPORT"

if find "$APP_PATH" \( -name 'ggml-*.bin' -o -name '*.partial' \) -print -quit | grep -q .; then
  echo "モデルファイルが .app に含まれています" >&2
  exit 1
fi

PLIST="$APP_PATH/Contents/Info.plist"
if ! grep -q 'NSMicrophoneUsageDescription' "$PLIST"; then
  echo "生成した Info.plist にマイク用途文がありません" >&2
  exit 1
fi
if ! grep -q '音声を文字起こしするためにマイクを使います。' "$PLIST"; then
  echo "生成した Info.plist のマイク文言が設計と一致しません" >&2
  exit 1
fi
if ! grep -q 'CFBundleExecutable' "$PLIST"; then
  echo "生成した Info.plist に CFBundleExecutable がありません" >&2
  exit 1
fi

rm -rf "$DIST_DIR/LocalScribe.app"
cp -R "$APP_PATH" "$DIST_DIR/LocalScribe.app"
echo "app -> $DIST_DIR/LocalScribe.app" | tee -a "$REPORT"
echo "ffmpeg bundled: $bundle_ffmpeg" | tee -a "$REPORT"
echo "完了。報告: $REPORT"
