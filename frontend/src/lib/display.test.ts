import { describe, expect, it } from "vitest";
import {
  errorMessage,
  fileBaseName,
  formatProgressPercent,
  isTerminalJobState,
  jobStateLabel,
  modelLabel,
  modelStatusLabel,
  segmentDisplayText,
} from "./display";

describe("formatProgressPercent", () => {
  it("returns null when percent is missing", () => {
    expect(formatProgressPercent(undefined)).toBeNull();
    expect(formatProgressPercent(null)).toBeNull();
  });

  it("formats measured percent", () => {
    expect(formatProgressPercent(42.4)).toBe("42%");
  });
});

describe("errorMessage", () => {
  it("reads Wails string errors and Error objects", () => {
    expect(errorMessage("モデルが導入されていません")).toBe("モデルが導入されていません");
    expect(errorMessage(new Error("FFmpeg が見つかりません"))).toBe("FFmpeg が見つかりません");
    expect(errorMessage({ message: "出力フォルダがありません" })).toBe("出力フォルダがありません");
    expect(errorMessage(null)).toBe("処理に失敗しました");
    expect(errorMessage(new TypeError("Cannot read properties of undefined (reading 'main')"))).toBe(
      "アプリとの接続ができませんでした。開き直してください。",
    );
  });
});

describe("labels", () => {
  it("turns job and model values into Japanese", () => {
    expect(jobStateLabel("transcribing")).toBe("認識中");
    expect(modelLabel("large-v3-turbo")).toBe("高速");
    expect(modelStatusLabel("missing")).toBe("未導入");
    expect(fileBaseName("/tmp/lecture.m4a")).toBe("lecture.m4a");
    expect(isTerminalJobState("failed")).toBe(true);
    expect(isTerminalJobState("converting")).toBe(false);
  });
});

describe("segmentDisplayText", () => {
  it("prefers edited_text over recognition text", () => {
    expect(
      segmentDisplayText({ text: "認識原文", edited_text: "編集後" }),
    ).toBe("編集後");
  });

  it("uses recognition text when not edited", () => {
    expect(segmentDisplayText({ text: "認識原文" })).toBe("認識原文");
  });
});
