# LocalScribe システム設計書 v1.0

- 状態: 確定
- 対象: macOS / Apple Silicon
- 開発基準機: M1 MacBook Air（16GB / 256GB）
- 互換確認機: M3 MacBook Pro（32GB / 1TB）
- 方針: 高精度な日本語音声認識を中心にした、完全無料・オフライン動作のデスクトップアプリ
- 設計の所有者: Grok 4.7
- 実装の担当: Composer 2.5
- 保存日: 2026-10-27
- 配置: `/Users/kitayamashuuma/SystemDevelopment/LocalScribe`
- GitHub: https://github.com/ShumaKitayama/LocalScribe

実装順と完了条件は `docs/plan/mvp-implementation-plan.md` を参照する。文書と既存コードが食い違うときの扱いは `AGENTS.md` に従う。コードがまだ無い機能は、本書の公開 API、状態、保存先、モデル一覧に合わせて作る。

## 1. 基本設計

GUI と CLI は独立し、両方が同じ Go のアプリケーション層を呼ぶ。GUI から CLI を起動しない。

```
Desktop GUI     Wails v2 + React + TypeScript + Vite
CLI             Go（cmd/localscribe）
共通層          Go（internal/application 以下）
音声前処理      FFmpeg / FFprobe
音声認識        whisper.cpp（whisper-cli、Metal）
永続化          ローカル JSON（DB は使わない）
```

| 項目 | 技術 | 方針 |
| --- | --- | --- |
| デスクトップ基盤 | Wails v2.17.0 | 安定版を固定する |
| フロントエンド | React + TypeScript + Vite | Wails の react-ts テンプレートに合わせる |
| ロジック | Go 1.25 系 | GUI と CLI で共有する |
| 音声認識 | whisper.cpp の `whisper-cli` | ローカルプロセス。cgo でライブラリ直結しない |
| 推論 | Metal | Apple GPU。`--no-gpu` は付けない |
| 音声変換 | FFmpeg / FFprobe | 引数配列で実行する |
| 設定・ジョブ・結果 | JSON | `schema_version` を持つ |
| テスト | Go `testing` / Vitest | 大きな音声は Git に入れない |
| バージョン管理 | Git + GitHub | モデル本体はリポジトリに入れない |

### アーキテクチャ原則

1. GUI と CLI は共通の Go サービスを使う。
2. whisper.cpp に依存するコードは `internal/engine/whisper` に閉じる。
3. 音声と認識結果は外部へ送らない。通信はモデルの初回取得だけに限る。
4. モデルは各 Mac の初回利用時に取得し、以降はオフラインで使う。
5. 長い処理はジョブとして非同期実行し、キャンセル、失敗、起動時復旧を扱う。
6. `internal/application` は Wails と React を import しない。

### Wails の版

2026-10-20 時点の v3.0.0-beta.34 を避ける、という当初判断のあと、2026-10-26 に Wails v3.0.0 が安定版として公開されている。v3 はバインディング方式が変わり、v2 からの移行は作り直しになる。MVP は **Wails v2.17.0**（2026-10-18）に固定する。v3 への移行は別の設計改訂であり、実装フェーズでは行わない。

macOS の開発に必要なのは Xcode Command Line Tools である。Xcode 本体は必須にしない。

## 2. 機能要件

MVP だけを最初の完成条件にする。Phase 2 以降は MVP 完了後の設計改訂で着手する。

| 機能 | 優先度 | 内容 |
| --- | --- | --- |
| 音声ファイル添付 | MVP | ドラッグ＆ドロップ、Finder 選択 |
| マイク録音 | MVP | 開始、停止、その音を文字起こし |
| 高精度文字起こし | MVP | Whisper large-v3（非量子化） |
| 高速モード | MVP | Whisper large-v3-turbo（非量子化） |
| 日本語認識 | MVP | 標準は `ja` |
| 音声形式変換 | MVP | MP3、M4A、WAV、FLAC |
| 動画 | MVP | MP4、MOV から音声を抽出 |
| 進捗とキャンセル | MVP | 段階、経過時間、中断 |
| 結果の編集とコピー | MVP | セグメント単位。全文コピー |
| 自動保存 | MVP | 指定フォルダへ新規ディレクトリ |
| TXT / SRT / JSON | MVP | JSON を正本にする |
| 独立 CLI | MVP | GUI なしで同じサービスを使う |
| モデル管理 | MVP | 初回取得、未完了ファイルは使わない、整合性確認 |
| クラッシュ復旧 | MVP | 未完了ジョブを `interrupted` にし、新規ジョブとして再実行できる |
| 文字起こし履歴の検索 | Phase 2 | 過去結果の検索 UI |
| VAD 調整 | Phase 2 | オンオフとパラメータ |
| Finder 右クリック | Phase 2 | クイックアクション |
| フォルダ監視 | Phase 2 | 投入で自動文字起こし |
| 複数話者の識別 | Phase 3 | 追加モデル |
| リアルタイム逐次表示 | Phase 3 | |
| 他アプリへの直接入力 | Phase 3 | グローバルショートカット |

MVP で実装しないもの: 履歴検索、VAD の調整 UI、Finder 連携、フォルダ監視、話者分離、リアルタイム認識、グローバルショートカット、量子化モデル、独自の音声分割。

## 3. UI

日常利用で操作が追えることを優先する。日本語 UI。ライトとダークは `prefers-color-scheme` に従う。

### メイン画面

- タイトルは LocalScribe
- ドロップ領域に、対応形式 `MP3 / M4A / WAV / FLAC / MP4 / MOV` を書く
- ファイル選択、録音、モデル、言語、保存先、開始ボタン
- 標準モデルは large-v3、言語は日本語、保存先の初期値は `~/Documents/LocalScribe/Transcripts`
- 処理中は同じ画面に、フェーズ名、経過時間、キャンセルを出す
- 進捗率が実測できるときだけパーセンテージを出す。取れないフェーズではバーを不定状態にし、架空のパーセントは出さない
- 完了したら結果画面へ移る
- 複数ファイルは投入順にキューへ入れる。同時に推論するのは 1 件
- 16GB 未満相当のマシンで large-v3 を選んだときは、turbo を案内する。選択自体は止めない

### 結果画面

左に再生、右にタイムスタンプ付きセグメント。

- タイムスタンプでその位置から再生する
- セグメントの編集後テキストだけを更新する。認識直後のテキストは残す
- 全文コピー、TXT / SRT / JSON の再出力、出力フォルダを Finder で開く
- 起動時に `interrupted` のジョブがあれば、元ファイルパスと「新規ジョブとして再実行」を出す。これは履歴検索ではない

### 再生

Wails 既存の AssetServer に、ジョブ ID から元ファイルを読むハンドラを足す。クエリで任意パスを渡す形にはしない。WebKit が再生できない形式では、文字起こし結果は表示したまま「この形式はアプリ内で再生できない」と出す。再生のためだけに元ファイルを変換し直さない。

### 録音

WebView の `MediaRecorder` で録り、停止時にバイト列を Go へ渡してキャッシュへ保存し、通常の文字起こしジョブへ入れる。`NSMicrophoneUsageDescription` は「音声を文字起こしするためにマイクを使います。」とする。GUI の作り込みより先に、実機で許可ダイアログと短い録音が通ることを確認する。ここで失敗したら実装を止めて設計へ戻す。

## 4. ディレクトリ

```
LocalScribe/
├── cmd/localscribe/main.go
├── internal/
│   ├── domain/
│   ├── application/
│   ├── audio/
│   ├── engine/
│   │   ├── engine.go
│   │   └── whisper/
│   ├── storage/
│   ├── export/
│   └── platform/macos/
├── frontend/
├── build/darwin/Info.plist
├── scripts/
│   ├── build-engine.sh
│   └── package-macos.sh
├── tests/fixtures/
├── main.go
├── app.go
├── wails.json
├── go.mod
├── Makefile
├── versions.lock
├── docs/design/system-design-v1.0.md
└── docs/plan/mvp-implementation-plan.md
```

| レイヤー | 役割 | 依存 |
| --- | --- | --- |
| `domain` | データ型と不変条件 | 外部ライブラリに依存しない |
| `application` | 手順、キュー、キャンセル、復旧 | domain とインターフェース |
| `audio` | Probe と 16kHz 変換 | FFmpeg |
| `engine` | 認識のインターフェース | domain |
| `engine/whisper` | whisper-cli の起動と JSON 解釈 | whisper-cli |
| `storage` | 設定、ジョブ、結果の JSON | ローカル FS |
| `export` | TXT / SRT / JSON | domain |
| `platform/macos` | パス解決 | macOS |
| `app.go` | Wails から呼ぶ API | application |
| `cmd/localscribe` | CLI | application |

Go module path は `github.com/ShumaKitayama/localscribe` とする。

2026-10-27 追記: `platform/macos` はパス用の `domain.UserError` も返す。依存表の「macOS」に加え、エラー型は domain を使う。

## 5. 音声認識パイプライン

1. 入力検証。拡張子、ファイルであること、読取可能であること、音声トラックがあること。
2. FFprobe で長さ、サンプルレート、チャンネル、音声ストリームの有無を取る。
3. FFmpeg で `16kHz / mono / PCM s16le` の WAV を作業ディレクトリへ書く。
4. whisper-cli で推論する。言語、モデル、初期プロンプトを渡す。
5. whisper の JSON をセグメントへ正規化する。
6. `transcript.json` を正本として原子的に保存し、TXT と SRT を同じ内容から生成する。
7. 作業 WAV を削除する。元ファイルは移動も削除もしない。

変換コマンドの引数は次で固定する。

```
ffmpeg -hide_banner -nostdin -i <input> -ar 16000 -ac 1 -c:a pcm_s16le <work.wav>
ffprobe -v error -print_format json -show_format -show_streams <input>
```

`-y` でユーザーのファイルを上書きしない。出力先は必ず作業ディレクトリ内の新規パスにする。

whisper-cli の引数は次で固定する。

```
whisper-cli -m <model> -f <wav> -l <lang> -oj -of <work/whisper-out> -pp
```

初期プロンプトが空でないときだけ `--prompt` を足す。`--translate`、`--no-gpu`、量子化を促すフラグは付けない。JSON のセグメント時刻は、出力にミリ秒の offset があればそれを優先する。実出力がこの仮定と違う場合、パーサーを想像で増やさず、フィクスチャを添えて設計へ戻す。

### 精度ルール

- 標準は非量子化 `large-v3`。高速は非量子化 `large-v3-turbo`。
- 日本語音声の言語は `ja`。MVP で選べる値は `ja`、`en`、`auto` だけ。
- 要約、意訳、補完をアプリ側でしない。
- 専門用語は初期プロンプトで補助できる。プロンプトは最大 500 文字。
- 無音除去と強いノイズ抑制を既定でかけない。
- VAD のオンオフは MVP に入れない。
- 長時間音声の分割は whisper.cpp に任せる。欠落が実測されたら、その時点で設計を改訂する。

## 6. ジョブ

状態は次だけを使う。

| 状態 | 意味 |
| --- | --- |
| `queued` | キューで待機 |
| `validating` | 入力検証 |
| `converting` | FFmpeg 変換 |
| `transcribing` | whisper 推論 |
| `exporting` | JSON / TXT / SRT 書き出し |
| `completed` | 成功 |
| `failed` | 失敗 |
| `cancelled` | ユーザーがキャンセル |
| `interrupted` | 起動時に未完了だった |

`completed`、`failed`、`cancelled`、`interrupted` は終了状態。終了状態のジョブは再開しない。再実行は同じ入力で新しいジョブを作る。

起動時、終了状態でないジョブはすべて `interrupted` に更新してからキューを空のまま待つ。自動再開しない。

ルール:

- 推論の同時実行は 1。待ちジョブは投入順。
- キャンセルは `context.Context` の取消で、FFmpeg と whisper-cli の両方に届ける。
- 取消で終わったジョブは `failed` にしない。`cancelled` にする。
- 保存は一時ファイルへ書いてから `rename` する。
- 出力ディレクトリ名は `{元ファイル名}_{YYYYMMDD}_{連番3桁}`。既存ディレクトリがあるときは連番を進める。上書きしない。
- 進捗イベントは状態、経過ミリ秒、メッセージを必ず持つ。パーセントは実測できたときだけ数値を入れる。

進捗の形:

```json
{
  "job_id": "...",
  "state": "transcribing",
  "elapsed_ms": 12000,
  "message": "音声を認識しています",
  "percent": 42
}
```

`percent` を出せないときはフィールドを省略する。0 や推測値で埋めない。

## 7. データ

タイムスタンプはミリ秒の整数。SRT へ出すときだけ `HH:MM:SS,mmm` にする。時刻の保存は UTC の RFC3339。

2026-10-27 追記: SRT の各字幕のあとには空行を置く。通し番号は 1 から始める。

`schema_version` は 1。

### Transcript JSON

```json
{
  "schema_version": 1,
  "id": "...",
  "job_id": "...",
  "model_id": "large-v3",
  "language": "ja",
  "source_path": "/absolute/path/lecture.m4a",
  "duration_ms": 600000,
  "created_at": "2026-10-27T00:00:00Z",
  "segments": [
    {
      "start_ms": 0,
      "end_ms": 1200,
      "text": "認識直後の文",
      "edited_text": null
    }
  ]
}
```

表示と TXT / SRT の再出力は、`edited_text` があればそれを、なければ `text` を使う。編集 API は `edited_text` だけを変える。空文字への編集は未編集とは区別し、認識文へ戻さない。

2026-10-27 追記: `TranscriptionRequest` に `source_kind` を追加した。空または `file` は MP3 / M4A / WAV / FLAC / MP4 / MOV、`recording` は WebM だけを受け付ける。進捗の `percent` は未計測なら JSON から省き、実測の 0 は 0 として残す。モデル状態の保存値は `missing`、`partial`、`available` である。

### 設定

```json
{
  "schema_version": 1,
  "default_model_id": "large-v3",
  "language": "ja",
  "output_dir": "/Users/.../Documents/LocalScribe/Transcripts",
  "export_formats": ["txt", "srt", "json"],
  "initial_prompt": ""
}
```

### Mac 上のパス

```
~/Library/Application Support/LocalScribe/
├── models/
│   ├── ggml-large-v3.bin
│   ├── ggml-large-v3-turbo.bin
│   └── manifest.json
├── bin/
│   └── whisper-cli
├── jobs/
└── settings.json

~/Library/Caches/LocalScribe/work/
~/Library/Logs/LocalScribe/localscribe.log

~/Documents/LocalScribe/Transcripts/{name}_{YYYYMMDD}_{nnn}/
├── transcript.json
├── transcript.txt
└── transcript.srt
```

作業 WAV はジョブ終了時（成功、失敗、キャンセル）に削除する。アプリ終了時にも `work/` の残留を削除する。元ファイルは触らない。

ID は `crypto/rand` の 16 バイトを 32 文字の hex にしたもの。テストでは生成関数を差し替えられるようにする。

## 8. モデルと資源

| ID | ファイル | 用途 | 目安 |
| --- | --- | --- | --- |
| `large-v3` | `ggml-large-v3.bin` | 高精度 | 2.9 GiB |
| `large-v3-turbo` | `ggml-large-v3-turbo.bin` | 高速 | 1.5 GiB |

取得 URL は公式スクリプトと同じ Hugging Face を使う。

```
https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-large-v3.bin
https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-large-v3-turbo.bin
```

取得ルール:

- ダウンロード先は `*.partial`。完了後にだけ正式名へ `rename` する。`.partial` は推論に渡さない。
- 中断後の再試行は partial を捨てて取り直してよい。
- 下限未満のファイルは未完成として拒否する。下限は `large-v3` が 2_800_000_000 バイト、`large-v3-turbo` が 1_400_000_000 バイト。
- 完了時に SHA-256 を計算して `manifest.json` に保存する。カタログにピンがある場合は一致必須。ピンが空のあいだは、初回取得のハッシュを manifest に記録し、次回以降その値と照合する。計測したハッシュは実装完了報告に含め、設計側がカタログへ固定する。
- 変換前の空き容量確認は `duration_ms / 1000 * 32000 + 64MiB`。長さが不明なときは 2 GiB 以上の空きを要求する。
- モデル取得前は、下限サイズ + 64MiB の空きを要求する。
- 物理メモリが 24 GiB 未満で large-v3 を選んだとき、turbo を案内する。

開発中の `whisper-cli` 解決順:

1. 環境変数 `LOCALSCRIBE_WHISPER_CLI`
2. `~/Library/Application Support/LocalScribe/bin/whisper-cli`
3. リポジトリの `build/whisper/whisper-cli`（Git 管理外）

2026-10-27 追記: 実装は、上記のあと PATH 上の `whisper-cli` も探す。`engine.Request` の作業ディレクトリは、出力プレフィックスが作業ディレクトリ内かの検査にはまだ使っていない。セグメント本文は前後の空白を除く。`-oj` に `offsets` が無いセグメントは 0 ミリ秒になる。実機の whisper-cli が無いあいだ、フィクスチャの出典は `tests/fixtures/whisper-output.source.md` に置く。JSON にはコメントを書けない。

FFmpeg の解決順:

1. `LOCALSCRIBE_FFMPEG` と `LOCALSCRIBE_FFPROBE`
2. アプリ同梱パス（パッケージ化後）
3. `PATH`

`scripts/build-engine.sh` は `https://github.com/ggml-org/whisper.cpp` を Metal 有効で Release ビルドし、採用した git revision を `versions.lock` に記録する。`master` の追従を固定値にしない。ビルド結果のバイナリは Git に入れない。

## 9. 公開インターフェース

アプリケーションサービス:

| 操作 | 役割 |
| --- | --- |
| `StartTranscription` | 検証できる入力でジョブをキューへ入れ、ID をすぐ返す |
| `GetJob` | 状態、進捗、エラー、結果 ID |
| `CancelJob` | 実行中または待機中をキャンセル |
| `ListRecoverableJobs` | `interrupted` だけを返す |
| `GetTranscript` | 結果とセグメント |
| `UpdateTranscript` | `edited_text` の保存と、JSON からの TXT / SRT 再生成 |
| `ExportTranscript` | 既存の出力先へ指定形式を再生成 |
| `ListModels` | カタログとインストール状態 |
| `InstallModel` | 取得と検証 |
| `GetSettings` / `UpdateSettings` | 設定 |
| `Doctor` | 環境診断。CLI の `doctor` と GUI の初期案内で使う |

`ListRecoverableJobs` と `Doctor` は、クラッシュ復旧と `localscribe doctor` を成立させるための API である。検索用の一覧 API は作らない。

Wails の `app.go` は上記をそのまま束ねる。録音バイトの保存は `app.go` の責務とし、保存したパスで `StartTranscription` を呼ぶ。

進捗は Wails イベント `job:progress` で送る。CLI は進捗を標準エラー、最終結果を標準出力へ出す。

### CLI

```
localscribe transcribe lecture.m4a --model large-v3 --language ja --format txt,srt,json
localscribe models list
localscribe models install large-v3
localscribe doctor
```

終了コード:

| code | 意味 |
| --- | --- |
| 0 | 成功 |
| 1 | 引数または入力検証の失敗 |
| 2 | 実行時失敗 |
| 3 | キャンセル |

`transcribe` の標準出力は、ジョブ ID、状態、出力ディレクトリ、生成ファイルを持つ JSON を 1 つ。`models list` の標準出力は JSON 配列。進捗とログは標準エラー。

診断項目は、OS が darwin/arm64 か、FFmpeg、FFprobe、whisper-cli、各モデルが partial でないか、出力先へ書けるか、Application Support と作業領域の空き容量。

## 10. 非機能とセキュリティ

| 分野 | 要件 |
| --- | --- |
| プライバシー | 音声と認識結果を外部送信しない |
| ネットワーク | モデル取得以外で接続しない |
| 安定性 | ジョブの失敗でプロセス全体を落とさない |
| 資源 | 推論は 1 並列 |
| 元ファイル | 変更も削除もしない |
| 衝突 | 既存の出力ディレクトリを上書きしない |
| エラー | ユーザー向け日本語と、ログ用の詳細を分ける |
| コマンド実行 | `exec.CommandContext` の引数配列。シェル文字列を組み立てない |
| パッケージ | App Sandbox は無効。ユーザー指定パスと Documents を読むため |
| ライセンス | 同梱 FFmpeg が GPL になるビルドなら、同梱を止めて設計へ戻す |

2026-10-27 追記: 開発機の Homebrew ffmpeg 9.0.2_1 は `--enable-gpl`（libx264、libx265）付きである。PATH 上の開発用としては使い、アプリ同梱の対象にはしない。
| バージョン | `versions.lock` に Go、Wails、whisper.cpp の revision、FFmpeg の入手方法を書く |

ユーザー向けエラーは操作で直せる内容にする（形式が未対応、モデル未導入、空き容量不足、マイクが拒否された、認識プロセスが終了した）。詳細ログは `~/Library/Logs/LocalScribe/localscribe.log` と CLI の標準エラーへ書く。

## 11. テスト

自動テストで見るもの:

- ドメインの状態遷移と、編集が `text` を壊さないこと
- SRT の時刻フォーマットと、JSON 正本から TXT / SRT が再生成されること
- whisper JSON フィクスチャのパーサー
- 偽エンジンを使ったジョブの順次実行、キャンセル、起動時の `interrupted`
- 容量不足や壊れた Probe 結果で元ファイルを消さないこと
- パーセントが無い進捗で数値が埋まらないこと

`integration` ビルドタグのテストだけが、実在する FFmpeg と whisper-cli を呼ぶ。短い WAV はテスト内で生成する。長い日本語音声の CER と RTF は人手の評価セットで測り、音声ファイルは Git に入れない。

必須の実機確認:

| 試験 | 確認 |
| --- | --- |
| 日本語の短い録音 | 認識結果が出る |
| 10 分、30 分、60 分 | 欠落と異常終了がない |
| 日本語と英語の混在 | 専門用語を初期プロンプトあり／なしで比較できる |
| 無音と雑音 | 無関係な長文を作り続けない |
| 壊れたファイル | プロセスが落ちず、失敗になる |
| キャンセル | 子プロセスが止まり、作業 WAV が残らない |
| 容量不足 | 既存の transcript を壊さない |
| M1 と M3 | arm64 と Metal で GUI と CLI が動く |
| モデル取得後 | ネットワークなしで文字起こしできる |

CER は文字誤り率、RTF は処理時間を音声長で割った値。large-v3 と turbo の両方を同じ評価セットで測る。

## 12. MVP 完成判定

次の 10 項目がすべて実機で確認できること。

1. CLI が MP3、M4A、WAV を 16kHz WAV に変換できる
2. large-v3 で日本語の文字起こしができる
3. GUI から音声と動画を選べる
4. マイク録音を文字起こしできる
5. 進捗確認とキャンセルができる
6. TXT、SRT、JSON を保存できる
7. 編集が JSON に残り、再出力と一致する
8. M1 と M3 の両方でアプリと CLI が動く
9. モデル取得後、ネットなしで動く
10. Dock から起動でき、通常利用にターミナルが不要

## 13. 実装時に固定する契約

この節は実装者が埋めない判断である。

- 外部プロセスは `exec.CommandContext`。
- 認識エンジンの差し替え単位は `internal/engine` のインターフェース。MVP の実装は whisper の 1 種類。
- ログは `log/slog`。ユーザー向け文と詳細を分ける。
- 設定の `~` は保存時に絶対パスへ展開する。
- 言語の許容値は `ja`、`en`、`auto`。
- 対応する入力拡張子は `.mp3` `.m4a` `.wav` `.flac` `.mp4` `.mov`。大小文字は区別しない。録音由来の `.webm` はマイク経路だけがジョブに渡せる。
- ディレクトリのドロップは拒否する。
- 出力の連番は同日同名で `001` から空きを探す。
- GUI の画面状態は待機、実行中、結果の 3 つ。実行中はメイン画面の中で表示する。
- フロントの生成物 `frontend/wailsjs` は Wails の生成結果を採用する。手書きでバインディングを模倣しない。
- `wails init` で `internal/` と `cmd/` を上書きしない。空の一時ディレクトリで生成し、`frontend/` と `wails.json` を取り込む。
