# LocalScribe

オフラインの macOS 文字起こし。共通処理は Go。GUI は Wails v2.17.0 + React。CLI は `cmd/localscribe`。モジュールは `github.com/ShumaKitayama/localscribe`。

設計は `docs/design/system-design-v1.0.md`。作業順と完了条件は `docs/plan/mvp-implementation-plan.md`。依頼されたフェーズだけを行い、次へ進まない。

## フェーズの進行

完了の宣言と、次へ進む指示があったら、この順で行う。

1. そのフェーズの差分を、完了条件に照らしてレビューする。
2. 問題があればコミットしない。理由を返し、次のフェーズには進まない。
3. 問題がなければコミットして push する。作業ブランチがあるときは `main` へマージしてから push する。`main` 上の変更は、コミットして push する。
4. 指示どおり、次のフェーズだけを実装する。

## ドキュメントとコード

ドキュメントと既存コードが食い違うときは、コードの書き方に従う。見つけた食い違いを、最後の返答で全て挙げる。コードがまだ無い事項は設計書に従う。コードを古い文書に合わせて書き戻さない。

仕様、手順、バージョン、完了状態に更新があれば、その都度、該当文書へ追記する。対象は `docs/design/system-design-v1.0.md`、`docs/plan/mvp-implementation-plan.md`、内容が利用者向けなら `README.md`、エージェント向けならこのファイル。経過を消して書き換えない。追記した食い違いも、最後の返答に全て残す。

## Go

Go を実装または修正するときは、必ず `/use-modern-go` を使う。編集する `.go` ごとに、そのファイルを対象にガイドラインの `list` を実行し、出力を最後まで読んでから書く。適用する項目だけ `explain` で確認する。`list` の結果を省略して Go を書かない。

## 境界

- 作らないもの: 履歴検索、VAD 調整、Finder 連携、フォルダ監視、話者分離、リアルタイム認識、他アプリ入力、Wails v3、量子化モデル。
- 推論は 1 並列。元ファイルは変更も削除もしない。出力先の既存ディレクトリは上書きしない。
- 外部プロセスは `exec.CommandContext` の引数配列で起動する。
- 進捗率が実測できないとき、パーセントを入れない。
- 通信はモデル取得だけ。音声と認識結果を送らない。
- コミットしないもの: `*.bin`、`*.partial`、作業 WAV、`build/whisper/`、`node_modules`。

## 配置

`internal/domain` 型。`application` 手順。`audio` FFmpeg。`engine/whisper` whisper-cli。`storage` JSON。`export` TXT/SRT/JSON。`platform/macos` パス。`app.go` は Wails の入口で、`application` は Wails を import しない。

## 確認

```sh
go test ./...
go test -tags integration ./...
```

フェーズ 9 以降は、テストに加えて変更した画面を実機で操作する。
