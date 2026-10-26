# LocalScribe

macOS（Apple Silicon）向けの、オフライン文字起こしアプリです。音声と認識結果は外部へ送りません。GUI と CLI は同じ Go の処理を使います。

フェーズ 9（Wails 枠・マイク確認）まで完了しています（実機録音の保存を確認済み）。文字起こし GUI（フェーズ 10）は未実装です。

- CLI: `go build -o localscribe ./cmd/localscribe`
- テスト: `make test`
- GUI（録音確認のみ）: `wails dev`（要 `npm install` in `frontend/`）。本番ビルドは `wails build`

| 文書                                             | 役割                                 |
| ------------------------------------------------ | ------------------------------------ |
| [設計書](docs/design/system-design-v1.0.md)      | 構成、機能、保存先、API              |
| [実装計画](docs/plan/mvp-implementation-plan.md) | フェーズ 0 から 12 の作業順          |
| [AGENTS.md](AGENTS.md)                           | 実装を担当するエージェント向けの指示 |

開発基準は M1 MacBook Air（16GB）、動作確認は M3 MacBook Pro（32GB）です。両機は同じ GitHub リポジトリ `ShumaKitayama/LocalScribe` を使います。

実装を始める前に必要なもの:

- Xcode Command Line Tools
- Go 1.25 以上
- FFmpeg（`ffmpeg` と `ffprobe`）

モデルファイル（約 4.4 GiB）は Git に含まれません。各 Mac の初回利用時に取得します。
