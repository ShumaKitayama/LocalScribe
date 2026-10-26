import { describe, expect, it } from "vitest";
import { formatProgressPercent, segmentDisplayText } from "./display";

describe("formatProgressPercent", () => {
  it("returns null when percent is missing", () => {
    expect(formatProgressPercent(undefined)).toBeNull();
    expect(formatProgressPercent(null)).toBeNull();
  });

  it("formats measured percent", () => {
    expect(formatProgressPercent(42.4)).toBe("42%");
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
