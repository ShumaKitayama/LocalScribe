# whisper-output.json の出典

開発機に `whisper-cli` が無かったため、2026-10-27 時点の ggml-org/whisper.cpp `examples/cli/cli.cpp` の `-oj` 出力形（`transcription[].offsets` はミリ秒、`text` は先頭空白あり）に合わせた。

参考:

- https://github.com/ggml-org/whisper.cpp/blob/master/examples/cli/cli.cpp （`output_json`）
- セグメント例の値は https://github.com/ggml-org/whisper.cpp/pull/3218 の JSON 抜粋

実機でビルド後に短い無音 WAV を通した出力へ差し替える場合は、このファイルと JSON を同時に更新する。
