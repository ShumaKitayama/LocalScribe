# LocalScribe

macOS（Apple Silicon）向けの、オフライン文字起こしアプリです。音声と認識結果は外部へ送りません。通信はモデルの取得だけです。GUI と CLI は同じ Go の処理を使います。

フェーズ 11（macOS パッケージ）まで完了しています。フェーズ 12 の受け入れは未完了です。M1 の途中結果は `docs/plan/acceptance-m1.md` にあります。M3 ではまだ確認していません。

| 文書                                             | 役割                        |
| ------------------------------------------------ | --------------------------- |
| [設計書](docs/design/system-design-v1.0.md)      | 構成、機能、保存先、API     |
| [実装計画](docs/plan/mvp-implementation-plan.md) | フェーズ 0 から 12 の作業順 |
| [AGENTS.md](AGENTS.md)                           | 実装を担当するエージェント向けの指示 |

開発基準は M1 MacBook Air（16GB）です。動作確認用のもう一台は M3 MacBook Pro（32GB）です。どちらも同じリポジトリ `ShumaKitayama/LocalScribe` を使います。

## 別の Mac への導入

対象は arm64 の macOS です。Intel Mac では `scripts/package-macos.sh` が止まります。アプリとモデルは Git に入っていません。各 Mac でクローンしてからビルドし、モデルを取得します。M1 で作った `build/dist` をコピーする手順ではありません。

プライベートリポジトリなので、その Mac の GitHub アカウントに `ShumaKitayama/LocalScribe` の読み取り権限が必要です。

必要なもの:

- Xcode Command Line Tools（`xcode-select --install`）
- Go 1.25 以上
- Node.js と npm（画面のビルドに使う）
- Wails v2.17.0
- CMake
- FFmpeg の `ffmpeg` と `ffprobe`（PATH に置く）

Homebrew の FFmpeg は GPL です。パッケージには同梱しません。実行時は、その Mac の PATH にある `ffmpeg` と `ffprobe` を使います。

```sh
git clone https://github.com/ShumaKitayama/LocalScribe.git
cd LocalScribe

brew install go node cmake ffmpeg
go install github.com/wailsapp/wails/v2/cmd/wails@v2.17.0

cd frontend && npm install && cd ..
scripts/build-engine.sh
make package-macos
```

`scripts/build-engine.sh` は whisper.cpp `d1be6fde11ac6e0407606b4e42fe72d34add8037` を取得し、Metal 付きの `build/whisper/whisper-cli` を作ります。`make package-macos` の成果物は次の二つです。

- `build/dist/LocalScribe.app`
- `build/dist/localscribe`

初回は診断してから、使うモデルを取得します。`large-v3` は約 2.9 GiB、`large-v3-turbo` は約 1.5 GiB です。モデルは `~/Library/Application Support/LocalScribe/models` に保存されます。

```sh
build/dist/localscribe doctor
build/dist/localscribe models install large-v3
open build/dist/LocalScribe.app
```

文字起こしの例:

```sh
build/dist/localscribe transcribe lecture.m4a --model large-v3 --language ja --format txt,srt,json
```

マイクを使うと、初回に「音声を文字起こしするためにマイクを使います。」の許可を求めます。結果の初期保存先は `~/Documents/LocalScribe/Transcripts` です。

## 開発時のコマンド

- テスト: `make test`
- GUI の開発起動: `wails dev`（先に `frontend` で `npm install`）
- CLI だけ: `go build -o localscribe ./cmd/localscribe`
