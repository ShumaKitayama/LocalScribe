# LocalScribe MVP 実装計画

作業順の正本はこの計画、仕様の正本は `docs/design/system-design-v1.0.md` である。エージェントの優先規則は `AGENTS.md` に従う。コードが無い事項で設計判断が足りないときは、近い実装で埋めずに作業を止め、差分の提案だけを報告する。

- 設計: Grok 4.7 が `docs/design` を更新する
- 実装: Composer 2.5 が、指定された 1 フェーズだけを実装する
- リポジトリ: `/Users/kitayamashuuma/SystemDevelopment/LocalScribe`
- 基準機: M1 MacBook Air 16GB。確認機: M3 MacBook Pro 32GB

## 使い方

1 回の実装依頼で 1 フェーズだけを渡す。フェーズ末尾の完了条件がテストで揃ってから次へ進む。各フェーズの「依頼文」を Composer 2.5 への最初のメッセージにする。

実装側が守ること:

- 指定フェーズのファイル以外へ機能を広げない
- Phase 2 以降の製品機能（履歴検索、VAD 調整、Finder 連携、フォルダ監視、話者分離、リアルタイム、他アプリ入力）を入れない
- Wails v3 へ上げない
- モデルの `.bin`、作業 WAV、録音、`node_modules`、`build/whisper/` を Git に入れない
- 外部コマンドは `exec.CommandContext` の引数配列にする
- 進捗率が取れない処理に、推測のパーセントを入れない

人手で先に済ませる環境:

- Xcode Command Line Tools（`xcode-select --install`）
- Go 1.25 以上（`go version`。未満ならフェーズ 0 を始めず報告する）
- Homebrew の `ffmpeg`（`ffmpeg` と `ffprobe` が PATH にあること）
- Git

2026-10-27 追記: この開発機では Homebrew の ffmpeg 9.0.2_1 を入れ、`/opt/homebrew/bin/ffmpeg` と `/opt/homebrew/bin/ffprobe` が PATH にある。configure に `--enable-gpl` と libx264 / libx265 が含まれる。開発中の PATH 利用はこれでよい。フェーズ 11 でこのバイナリを `.app` にコピーしない。

`wails` CLI はフェーズ 9 の直前に `go install github.com/wailsapp/wails/v2/cmd/wails@v2.17.0` で入れる。フェーズ 0 ではまだ使わない。

## フェーズ一覧

| 順  | 名前             | 終わるときの状態                                         |
| --- | ---------------- | -------------------------------------------------------- |
| 0   | 骨格             | モジュールと無視規則があり、テストが空で通る             |
| 1   | domain           | 型、状態、不変条件の単体テストが通る                     |
| 2   | パスと保存       | 設定、ジョブ、結果の原子的 JSON 保存が通る               |
| 3   | audio            | FFmpeg 変換と Probe がインターフェース越しにテストできる |
| 4   | export           | JSON 正本から TXT と SRT を再生成できる                  |
| 5   | whisper エンジン | フィクスチャ JSON を解釈し、偽の実行でキャンセルできる   |
| 6   | アプリケーション | 1 並列キュー、復旧、失敗が偽エンジンで通る               |
| 7   | モデル取得       | partial を推論に渡さず、取得と検証ができる               |
| 8   | CLI              | 設計書の 4 コマンドと終了コードが揃う                    |
| 9   | マイク確認       | 空に近い Wails 窓で録音許可と 3 秒録音が実機で通る       |
| 10  | GUI              | ドロップ、進捗、結果編集、再出力ができる                 |
| 11  | パッケージ       | Dock から起動する `.app` と CLI バイナリ                 |
| 12  | 受け入れ         | 設計書 12 節の 10 項目を実機で記録する                   |

フェーズ 9 でマイクが失敗したら、フェーズ 10 へ進まない。

---

## フェーズ 0 — 骨格

作るファイル:

- `go.mod`（module `github.com/ShumaKitayama/localscribe`、Go 1.25）
- `go.sum`（依存を足したときだけ）
- `.gitignore`
- `versions.lock`
- `Makefile`
- `internal/platform/macos/paths.go` はまだ作らない

`.gitignore` に含めるもの:

- `build/bin/`、`build/whisper/`、`frontend/node_modules/`、`frontend/dist/`
- `*.bin`、`*.partial`、`*.wav`（ただし `tests/fixtures/**/*.json` は追跡する）
- `.DS_Store`

`versions.lock` の初期値:

```
go=1.25
wails=v2.17.0
module=github.com/ShumaKitayama/localscribe
whisper_cpp_repo=https://github.com/ggml-org/whisper.cpp
whisper_cpp_ref=unpinned
ffmpeg=system
```

`Makefile` のターゲットは `test`（`go test ./...`）と `test-integration`（`-tags integration`）。ツールが無くても `test` は成功する。

完了条件: `go test ./...` が成功する。アプリケーションコードはまだ無い。

2026-10-27 追記（フェーズ 0 完了）: `go.mod`、`.gitignore`、`versions.lock`、`Makefile` を追加した。Go 1.27 ではパッケージが無いと `go test ./...` が終了コード 1 になるため、製品ロジックを含まない `tests/smoke/smoke_test.go` だけを置いた。`internal/platform/macos/paths.go` は未作成。

2026-10-27 追記（レビュー）: フェーズ 0 を受け入れて `main` へ入れた。`make test` と `make test-integration` は成功した。`go.mod` の `go 1.25` は最低言語バージョンで、この開発機のツールチェーンは Go 1.27.3 である。作業ブランチは無く、共有は `main` への push で行った。

依頼文:

```
docs/design/system-design-v1.0.md と docs/plan/mvp-implementation-plan.md に従い、フェーズ 0 だけ実装してください。go.mod、.gitignore、versions.lock、Makefile を追加し、go test ./... が通るところまでです。Wails の初期化、ドメイン型、音声処理には進まないでください。
```

## フェーズ 1 — domain

作るファイル:

- `internal/domain/job.go`
- `internal/domain/transcription.go`
- `internal/domain/segment.go`
- `internal/domain/model.go`
- `internal/domain/settings.go`
- `internal/domain/errors.go`
- `internal/domain/domain_test.go`

設計書 6 節と 7 節の型を、JSON タグ付きで置く。状態定数、終了状態の判定、編集適用（`edited_text` だけを変え、`text` は維持）、言語の許容値 `ja` / `en` / `auto`、出力形式 `txt` / `srt` / `json`、拡張子判定をここに置く。I/O とプロセス起動は置かない。

ユーザー向けエラーはメッセージと詳細ログを分けた型にする。

完了条件: 状態を終了状態から実行中へ戻す操作がテストで拒否される。編集テストで認識原文が残る。不正言語が拒否される。`go test ./...` が成功する。

2026-10-27 追記（フェーズ 1 完了）: `internal/domain` に型と不変条件を追加し、`go test ./...` が成功した。終了状態からの遷移は拒否し、元の状態を残す。`source_kind`、進捗パーセントの省略、モデル状態 `missing` / `partial` / `available` は設計書 2026-10-27 追記と同じである。

依頼文:

```
設計書と実装計画のフェーズ 1 だけ実装してください。internal/domain にジョブ、文字起こし、セグメント、モデル、設定の型と不変条件、表形式のテストを追加してください。ファイル保存、FFmpeg、Wails には進まないでください。
```

## フェーズ 2 — パスと保存

作るファイル:

- `internal/platform/macos/paths.go`
- `internal/storage/atomic.go`
- `internal/storage/settings.go`
- `internal/storage/job.go`
- `internal/storage/transcript.go`
- 対応する `*_test.go`

パス関数はホームディレクトリを引数で受け取り、設計書 7 節の配下を返す。`~` の展開はここで行う。

原子的保存は `同名.tmp` に書いて `rename` する。失敗時に一時ファイルを残し続けない。

ジョブ保存は 1 ジョブ 1 JSON、置き場は `jobs/<id>.json`。起動時に全ジョブを読み、終了状態でないものを `interrupted` に更新する関数を storage に置く。自動再開はしない。

文字起こし結果の出力ディレクトリは `{出力先}/{stem}_{YYYYMMDD}_{連番}`。連番は `001` から既存を避けて決める。ディレクトリ作成後に `transcript.json` を原子的に書く。

テストは `t.TempDir()` を使う。

完了条件: 保存の途中失敗で最終パスができない。再起動相当の読み込みで `queued` と `transcribing` が `interrupted` になる。同名出力が上書きされない。

2026-10-27 追記（フェーズ 2 完了）: `internal/platform/macos` と `internal/storage` を追加し、`go test ./...` が成功した。rename 失敗時は最終ファイルと `.tmp` を残さない。再読み込みで `queued` と `transcribing` は `interrupted` になり、`completed` は変わらない。同名の出力は `001` の次に `002` を作る。`platform/macos` は `domain.UserError` を返す。設計の依存表には無いが、エラーの形は domain に合わせてある。

依頼文:

```
設計書と実装計画のフェーズ 2 だけ実装してください。platform/macos のパスと、storage の設定・ジョブ・結果 JSON の原子的保存、中断マークを、一時ディレクトリを使ったテスト付きで追加してください。音声変換や CLI には進まないでください。
```

## フェーズ 3 — audio

作るファイル:

- `internal/audio/probe.go`
- `internal/audio/converter.go`
- `internal/audio/command.go`
- `internal/audio/audio_test.go`
- `internal/audio/integration_test.go`（`//go:build integration`）

Probe と Converter はコマンド実行をインターフェースに分離し、単体テストは偽の実行結果を渡す。実 FFmpeg は integration タグだけ。

Probe は音声ストリームが無い入力をエラーにする。Converter の引数は設計書 5 節の並びにする。出力先が作業ディレクトリ外なら拒否する。元パスに対する削除 API は作らない。

作業 WAV の必要バイト数は `duration_ms / 1000 * 32000 + 64MiB`。長さ不明は 2 GiB。空き容量の取得は macOS の `unix.Statfs` でよい。

完了条件: 引数テストが設計書のフラグ順と一致する。音声ストリーム無し、容量不足、作業ディレクトリ外への出力がエラーになる。偽実装は元ファイルを開いて書き込まない。

2026-10-27 追記（フェーズ 3 完了）: `internal/audio` に FFprobe / FFmpeg の引数組み立て、`CommandRunner`、空き容量確認（`unix.Statfs`）、作業ディレクトリ内出力の検証を追加した。単体テストは偽 `CommandRunner`、`integration` タグだけ実 FFmpeg。依存に `golang.org/x/sys` を追加した。`go test ./...` が成功する。

2026-10-27 追記（レビュー）: 完了条件を確認し、`go test ./...` と `go test -tags integration ./internal/audio/` が成功した。統合テストのサンプル音声生成は `exec.Command` だったので、`exec.CommandContext` と `t.Context()` に直した。アプリ同梱パスの解決はフェーズ 11 のままである。

依頼文:

```
設計書と実装計画のフェーズ 3 だけ実装してください。internal/audio に FFprobe と FFmpeg の引数組み立て、空き容量の事前確認、差し替え可能なコマンド実行を追加してください。単体テストは偽の実行結果で書き、実 FFmpeg は integration タグに分離してください。whisper と CLI には進まないでください。
```

## フェーズ 4 — export

作るファイル:

- `internal/export/json.go`
- `internal/export/txt.go`
- `internal/export/srt.go`
- `internal/export/export_test.go`

JSON は設計書 7 節の形で、`schema_version` が 1。TXT は表示文を改行で連結する。表示文は `edited_text` があればそれ、無ければ `text`。SRT は 1 始まりの通し番号と `HH:MM:SS,mmm`。ミリ秒の桁は 3 桁ゼロ埋め。カンマ区切り。

再出力は常に JSON を読み直した結果から TXT と SRT を生成する。whisper の SRT をそのままユーザー出力にしない。

完了条件: 0 ミリ秒、1 時間超、編集済みセグメントのゴールデンテストが通る。

2026-10-27 追記（フェーズ 4 完了）: `internal/export` で Transcript JSON のエンコード／デコードと、表示文からの TXT / SRT 生成を追加した。再出力は `TXTFromJSON` / `SRTFromJSON` が JSON を読み直す。`go test ./...` が成功する。

2026-10-27 追記（レビュー）: 0 ミリ秒、1 時間超、編集済みセグメントのテストを確認した。提出時の SRT は字幕のあいだに空行が無く、SubRip として区切れなかった。各字幕の末尾に空行を足し、ゴールデンテストもそれに合わせた。負のミリ秒は SRT 整形時に 0 へ丸める。JSON 経路では domain の検証が負の時刻を拒否する。

依頼文:

```
設計書と実装計画のフェーズ 4 だけ実装してください。internal/export で transcript JSON を正本にし、表示文から TXT と SRT を生成するテストを追加してください。whisper-cli の SRT をユーザー向け出力として保存しないでください。
```

## フェーズ 5 — engine / whisper

作るファイル:

- `internal/engine/engine.go`
- `internal/engine/whisper/process.go`
- `internal/engine/whisper/parser.go`
- `internal/engine/whisper/parser_test.go`
- `tests/fixtures/whisper-output.json`
- `scripts/build-engine.sh`

エンジンの入力は WAV パス、モデルパス、言語、初期プロンプト、作業ディレクトリ。出力はドメインのセグメント。進捗コールバックは、実測パーセントがあるときだけ数値を渡す。

`whisper-cli` の引数は設計書 5 節。標準エラーから進捗行を読む。パーセントと解釈できない行はメッセージだけにし、パーセントは渡さない。取消時はプロセスを止め、エラーをキャンセルとして区別する。

パーサーは最初に、手元で一度だけ短い無音 WAV を通した実出力を `tests/fixtures/whisper-output.json` として保存し、その形に合わせる。実出力を取る前にフィールド名を想像で確定しない。実コマンドが動かない環境では、公式 CLI の `-oj` 出力例をフィクスチャにし、その出典をファイル先頭コメントに書く。offset のミリ秒が無い場合はフェーズを完了扱いにせず、フィクスチャを添えて報告する。

`scripts/build-engine.sh` は ggml-org/whisper.cpp を shallow clone ではなく特定 revision で取得し、`cmake -B build -DGGML_METAL=ON` と Release ビルドを行い、`build/whisper/whisper-cli` へコピーする。使った revision を `versions.lock` の `whisper_cpp_ref` に書く。バイナリはコミットしない。

完了条件: フィクスチャのパーサーテストが通る。偽プロセスでキャンセルが `cancelled` 相当のエラーになる。パーセントが無い出力で進捗数値が nil のままである。

2026-10-27 追記（フェーズ 5 完了）: `internal/engine` の `Engine` インターフェース、`internal/engine/whisper`（引数、標準エラー進捗、`ParseOutputJSON`、context 取消）、`tests/fixtures/whisper-output.json`、`scripts/build-engine.sh` を追加した。開発機に whisper-cli が無く、フィクスチャは whisper.cpp 公式 `-oj` 形（出典は `tests/fixtures/whisper-output.source.md`）。`go test ./...` が成功する。

2026-10-27 追記（レビュー）: フィクスチャの `offsets.from` / `offsets.to`（990 / 7800）をパーサーテストが読んでいる。偽プロセスの取消は `engine.ErrCancelled`、パーセントの無い行は `Percent == nil`。`versions.lock` の `whisper_cpp_ref` は未ビルドのため `unpinned` のまま。ビルド用の clone 先 `build/whisper-src/` を Git の対象外にした。実機の `-oj` への差し替えは、`scripts/build-engine.sh` 実行後に行う。

依頼文:

```
設計書と実装計画のフェーズ 5 だけ実装してください。engine のインターフェース、whisper-cli の引数とキャンセル、JSON パーサー、ビルドスクリプトを追加してください。パーサーは tests/fixtures の実出力に合わせ、出力形式が設計と食い違う場合は実装を広げずにフィクスチャと差分を報告してください。アプリケーションサービスと GUI には進まないでください。
```

## フェーズ 6 — application

作るファイル:

- `internal/application/transcribe_service.go`
- `internal/application/job_manager.go`
- `internal/application/service_test.go`

サービスは設計書 9 節の操作を持つ。このフェーズの `InstallModel` は未実装エラーでよい。モデルファイルが所定パスにあり、partial でないことは `StartTranscription` が確認する。

`StartTranscription` はジョブを `queued` で保存して ID を返し、ワーカーが 1 本だけ次を取る。処理順は validating、converting、transcribing、exporting、completed。どれで失敗しても `failed` とユーザー向けメッセージを保存し、プロセスは落とさない。キャンセルは待機中なら子プロセスを起動せず `cancelled`。実行中なら context を取消する。終了時は作業 WAV を削除する。

ワーカーと保存層はテストで偽エンジン、偽コンバータ、一時ディレクトリに差し替える。

`ListRecoverableJobs` は `interrupted` だけを返す。

完了条件: 2 件連続投入で同時に transcribing にならない。キャンセルで作業ファイルが残らない。失敗で 2 件目が続行する。再起動処理のあと自動では推論が始まらない。

2026-10-27 追記（フェーズ 6 完了）: `internal/application` に 1 並列ワーカーと設計書 9 節のサービス API を追加した。`InstallModel` は未実装。ジョブ JSON に結果ディレクトリ `result_dir` を保存する（`GetTranscript` / 再出力用）。偽 Prober / Converter / Engine のテストで完了条件を確認した。`go test ./...` が成功する。

2026-10-27 追記（レビュー）: `go test -race ./internal/application/` を含め完了条件を確認した。テスト用 ID 生成はワーカーと呼び出し元が同時に使うため、mutex で保護した。`Doctor` は large-v3 が使えるかだけを見る。`Close` はワーカーの待ち受けを止めるが、実行中ジョブの context は取消しない。

依頼文:

```
設計書と実装計画のフェーズ 6 だけ実装してください。application に 1 並列のジョブマネージャーと文字起こしサービスを、偽の音声変換と偽の認識エンジンでテストしてください。モデルのダウンロード、CLI、Wails には進まないでください。
```

## フェーズ 7 — モデル取得

作るファイル:

- `internal/application/model_service.go`
- `internal/application/catalog.go`
- `internal/application/model_service_test.go`

カタログは設計書 8 節の 2 モデルだけ。HTTP クライアントはインターフェースにし、テストは偽サーバー（`httptest`）を使う。

手順は、空き容量確認、`.partial` へ取得、下限サイズ、SHA-256、manifest 更新、`rename`。失敗時に正式ファイル名へ残さない。カタログの SHA-256 が空なら、初回のハッシュを manifest に書き、2 回目以降はそれと照合する。計測値は完了報告に書く。

`ListModels` は未導入、取得途中、利用可能を区別する。

完了条件: 途中で切れた応答が `.partial` のままになり、`ListModels` が利用可能にしない。下限未満を拒否する。正式ファイルの上書きをしない。

2026-10-27 追記（フェーズ 7 完了）: `catalog.go` と `model_service.go` を追加し、`InstallModel` / `ListModels` を実装した。取得は `.partial` → サイズ・SHA-256 → `manifest.json` → `rename`。httptest のペイロード `model-payload-for-test-install` の SHA-256 は `05bda8466ff17f066afec451d325befa7bb01ba3c214e90af1c66d769f38af10`（カタログのピンは未固定）。`go test ./...` が成功する。

2026-10-27 追記（レビュー）: 中断で `.partial` が残ること、下限未満とハッシュ不一致で `.partial` を消すこと、正式ファイルを上書きしないこと、空き容量不足では HTTP を呼ばないこと、正式ファイルを消した再取得は manifest の SHA-256 と照合することを確認した。実モデルの SHA-256 は未計測で、カタログのピンは空のまま。導入済みファイルの照合と `ListModels` は manifest の読み取り失敗を別扱いする。文字起こし開始時の確認は標準カタログ固定。空き容量の要求は下限 + 64MiB。物理メモリ 24 GiB 未満の案内は未実装。

依頼文:

```
設計書と実装計画のフェーズ 7 だけ実装してください。2 モデルのカタログ、partial を使った取得、サイズと SHA-256 の検証を httptest で確認してください。実ネットワークへの大きなダウンロードはテストに含めないでください。CLI と GUI には進まないでください。
```

## フェーズ 8 — CLI

作るファイル:

- `cmd/localscribe/main.go`
- `cmd/localscribe/cli_test.go`

サブコマンドは `transcribe`、`models list`、`models install`、`doctor`。フラグ解析は標準ライブラリで行う。追加の CLI フレームワークは入れない。

標準出力と標準エラー、終了コードは設計書 9 節。`transcribe` はサービスを呼び、終了まで待ってから結果 JSON を 1 つ出す。進捗は標準エラー。SIGINT で `CancelJob` し、終了コード 3。

`doctor` は設計書の診断項目を人間可読で標準出力へ書き、問題があれば終了コード 2。

完了条件: フラグ不足が終了コード 1。偽サービスで成功 JSON が標準出力だけに出る。進捗行が標準出力へ混ざらない。

2026-10-27 追記（フェーズ 8 完了）: `cmd/localscribe` に `transcribe` / `models list` / `models install` / `doctor` を追加した。終了コード 0–3、進捗は標準エラー、結果 JSON は標準出力。`go test ./...` が成功する。

2026-10-27 追記（レビュー）: フラグ不足と `doctor` は、FFmpeg や whisper-cli が無くても文字起こしサービスを起動しない。`models` はモデル取得サービスだけを使う。`transcribe` の結果 JSON の `files` は、出力先を読まずに組み立てたファイル名。入力パスはフラグより前に置く。`models install` は SIGINT で取得を取消しない。既にインストール済みは終了コード 1。長すぎる初期プロンプトは終了コード 2。実機の `doctor` は whisper-cli が無いため終了コード 2。

依頼文:

```
設計書と実装計画のフェーズ 8 だけ実装してください。cmd/localscribe に transcribe、models list、models install、doctor を、標準ライブラリのフラグ解析で追加してください。終了コードと標準出力・標準エラーの分離をテストしてください。Wails には進まないでください。
```

## フェーズ 9 — Wails の枠とマイク確認

ここだけは実機確認が完了条件に入る。

手順:

1. 空の一時ディレクトリで `wails init -n localscribe -t react-ts` を実行する。
2. `frontend/` と `wails.json` を本リポジトリへコピーする。`internal/` と `cmd/` は上書きしない。
3. ルートの `main.go` と `app.go` を、既存モジュールを呼ぶ最小構成で書く。
4. `Info.plist` に `NSMicrophoneUsageDescription` を設計書の文面で入れる。
5. 画面は録音開始と停止だけ。停止後にバイト列を Go のキャッシュへ保存し、保存パスを画面へ出す。文字起こしへはまだ繋がない。
6. `wails dev` で起動し、許可ダイアログ、3 秒録音、ファイル生成を確認する。

マイクが録音できない、許可が出ない、WKWebView に `MediaRecorder` が無い、のいずれかならフェーズ 10 を始めず、症状と macOS のバージョンを報告する。

完了条件: 生成パスの音声ファイルが 0 バイトでない。App Sandbox を有効にしていない。

2026-10-27 追記（フェーズ 9 実装）: Wails v2.17.0 の `react-ts` から `frontend/` と `wails.json` を取り込み、ルートに `main.go` / `app.go`、`build/darwin/Info.plist`（マイク文言）、録音開始・停止 UI と `SaveRecording` を追加した。`wails build` で `.app` を生成し `NSMicrophoneUsageDescription` を確認済み。

2026-10-27 追記（フェーズ 9 実機確認完了）: macOS 27.0.1 上でマイク許可・録音停止後に `~/Library/Caches/LocalScribe/work/recordings/recording_1793047549472.webm`（157593 バイト）が生成された。フェーズ 10 に進める。

2026-10-27 追記（レビュー）: 同じファイルが 157593 バイトであることを確認した。画面は録音開始と停止だけで、文字起こしには繋いでいない。`frontend/dist` は `main.go` の埋め込みに必要なので Git に含める。`Info.plist` はマイク文言だけで、CFBundle キーは無い。App Sandbox は無効。`go test ./...` は成功する。フェーズ 10 は未着手。

依頼文:

```
設計書と実装計画のフェーズ 9 だけ実装してください。一時ディレクトリで Wails v2.17.0 の react-ts を生成し、frontend と wails.json だけを取り込み、マイクの使用目的文と 3 秒録音の保存までを実機で確認してください。録音できなければ文字起こし UI へ進まず、症状を報告してください。internal と cmd は上書きしないでください。
```

## フェーズ 10 — GUI

作るファイルの置き場:

- `frontend/src/pages/MainPage.tsx`（待機と実行中）
- `frontend/src/pages/ResultPage.tsx`
- `frontend/src/components/DropZone.tsx`
- `frontend/src/components/RecordButton.tsx`
- `frontend/src/components/SettingsForm.tsx`
- `frontend/src/components/JobProgress.tsx`
- `frontend/src/components/TranscriptEditor.tsx`
- `frontend/src/components/AudioPlayer.tsx`
- `frontend/src/hooks/useJobEvents.ts`
- `app.go` の公開メソッドを設計書 9 節まで広げる

操作は Wails バインディング経由。進捗はイベント `job:progress`。パーセントが無いイベントでは数値を表示しない。

再生は設計書 3 節の AssetServer ハンドラ。ジョブ ID 以外でファイルを読めないテストを Go 側に置く。

編集の保存は `UpdateTranscript` を呼び、成功後の表示が再取得結果と一致する。

複数ドロップは投入順に `StartTranscription` する。

Vitest では、パーセント欠落時に `%` 表示が出ないことと、`edited_text` の表示優先を見る。

完了条件: `go test ./...` とフロントのユニットテストが成功する。実機で、短い音声の選択、進捗、キャンセル、結果表示、1 セグメント修正、TXT の再出力までを操作して確認する。

依頼文:

```
設計書と実装計画のフェーズ 10 だけ実装してください。メイン画面と結果画面を、共通 Go サービスへの Wails バインディングで接続してください。進捗率が無いときパーセントを表示しないこと、セグメント編集が認識原文を残すこと、再生ハンドラがジョブ ID 以外のパスを拒否することをテストしてください。パッケージ化と Phase 2 機能には進まないでください。
```

## フェーズ 11 — パッケージ

作るファイル:

- `scripts/package-macos.sh`
- `build/darwin/Info.plist` のマイク文面の確認

スクリプトは arm64 の `wails build` と、`cmd/localscribe` の `go build` を行う。whisper-cli はフェーズ 5 の成果物を `.app` 内の決まった相対パスへコピーする。FFmpeg は `ffmpeg -version` のライセンス表示を記録し、GPL のライブラリを含むビルドならコピーを止めて報告する。LGPL 相当と確認できたときだけ同梱する。

`versions.lock` に whisper.cpp revision と FFmpeg の入手元を書く。

完了条件: `.app` を Dock から起動してファイル選択ができる。ターミナルを開かずにその操作ができる。CLI バイナリが `doctor` を実行できる。モデルファイルは `.app` に bake しない。

依頼文:

```
設計書と実装計画のフェーズ 11 だけ実装してください。arm64 の .app と CLI を作る scripts/package-macos.sh を追加し、whisper-cli の同梱と FFmpeg のライセンス確認を入れてください。GPL になる FFmpeg は同梱せず報告してください。モデルの bin はアプリに含めないでください。
```

## フェーズ 12 — 受け入れ

コードを足すフェーズではない。設計書 12 節の 10 項目と、11 節の実機表を、M1 と M3 で実行し、結果を `docs/plan/acceptance-m1.md` と `docs/plan/acceptance-m3.md` に日付付きで残す。失敗項目は修正フェーズを新しく切る前に、設計上の問題か実装欠陥かを報告する。

CER と RTF は、リポジトリ外の評価用音声で測る。音声自体はコミットしない。数値と条件（モデル、言語、マシン、音声長）だけを記録する。

依頼文:

```
設計書 12 節の受け入れ 10 項目を、この Mac で確認し、docs/plan/acceptance に結果だけ記録してください。失敗した項目を別機能の実装で迂回しないでください。評価用の音声ファイルはコミットしないでください。
```

## フェーズをまたぐ禁止

- 公開メソッドの改名、状態名の追加、保存先の変更
- データベースの導入
- 認識結果の要約や言い換え
- 2 本目の推論ワーカー
- 量子化モデルの追加
- モデル取得以外の HTTP
- シェル経由の FFmpeg / whisper 起動
- 元ファイルの削除と上書き
- 架空の進捗率
