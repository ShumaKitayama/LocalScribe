export type SegmentLike = {
  text: string;
  edited_text?: string | null;
};

/** 表示用テキスト。edited_text があればそれを優先する。 */
export function segmentDisplayText(segment: SegmentLike): string {
  if (segment.edited_text != null && segment.edited_text !== undefined) {
    return segment.edited_text;
  }
  return segment.text;
}

/** 進捗率が実測されているときだけパーセント文字列を返す。 */
export function formatProgressPercent(percent: number | null | undefined): string | null {
  if (percent === null || percent === undefined) {
    return null;
  }
  return `${Math.round(percent)}%`;
}
