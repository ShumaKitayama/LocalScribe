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
- FFmpeg の `ffmpeg` と `ffprobe`（下の手順で PATH に置く）

### FFmpeg

Apple Silicon の Homebrew なら、次の 1 本で `ffmpeg` と `ffprobe` の両方が入ります。置き場所は `/opt/homebrew/bin` です。

```sh
brew install ffmpeg
command -v ffmpeg
command -v ffprobe
```

両方とも `/opt/homebrew/bin/...` と出れば、ターミナルの PATH に載っています。`command -v` が空なら、新しいターミナルを開くか、`eval "$(brew shellenv)"` を実行してからもう一度確認します。

この FFmpeg は GPL です。`.app` には同梱しません。実行時は、そのプロセスの PATH にある `ffmpeg` と `ffprobe` を使います。

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

## トラブルシューティング

アプリが起動することと、文字起こしに必要なファイルが揃っていることは別です。モデルの `ggml-*.bin` は `.app` に入っていません。初回の取得は、ネットワークがある状態で、その Mac ごとに 1 回行います。GUI の設定画面にモデル取得のボタンはありません。取得は CLI で行います。

### モデルが導入されていません

`doctor` に次が出るのは、まだ取得していないときの表示です。`missing` だけでは `doctor` は失敗にしません。

```text
Model large-v3: missing
Model large-v3-turbo: missing
```

文字起こしを始めると「モデルが導入されていません」になります。リポジトリのディレクトリで、パッケージした CLI を使います。

```sh
build/dist/localscribe models list
build/dist/localscribe models install large-v3
```

高速側は `build/dist/localscribe models install large-v3-turbo` です。保存先は GUI と CLI で同じ `~/Library/Application Support/LocalScribe/models/` です。取得後に確認します。

```sh
build/dist/localscribe doctor
build/dist/localscribe models list
```

`large-v3` が `available OK` になってから、`build/dist/LocalScribe.app` でファイル選択と文字起こしを試します。

### FFmpeg が見つかりません

`doctor` の該当行は、見つかれば `FFmpeg: /opt/homebrew/bin/ffmpeg OK` と `FFprobe: /opt/homebrew/bin/ffprobe OK`、無ければ `FFmpeg: 見つかりません` です。上の「FFmpeg」の手順で入れます。

ターミナルから実行した `doctor` と、Dock から起動した GUI は、同じ探し方を使います。プロセスの PATH は別です。Dock 起動のアプリには、ターミナルで有効な `/opt/homebrew/bin` が入らないことがあります。そのときは GUI だけが「FFmpeg が見つかりません」になります。PATH が通っているターミナルから、アプリ本体を直接起動すると、その PATH を使います。

```sh
build/dist/LocalScribe.app/Contents/MacOS/LocalScribe
```

whisper-cli は `.app` の `Contents/Resources/bin/whisper-cli` に入っています。FFmpeg とは別です。

### cmake がありません

`scripts/build-engine.sh` が `cmake: command not found` で止まったときは、エンジンのビルドがその時点で終わっていません。

```sh
brew install cmake
scripts/build-engine.sh
make package-macos
```

## 開発時のコマンド

- テスト: `make test`
- GUI の開発起動: `wails dev`（先に `frontend` で `npm install`）
- CLI だけ: `go build -o localscribe ./cmd/localscribe`
