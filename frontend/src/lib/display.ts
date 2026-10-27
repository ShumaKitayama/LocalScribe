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

const terminalStates = new Set(["completed", "failed", "cancelled", "interrupted"]);

/** これ以上進まないジョブ状態なら true。 */
export function isTerminalJobState(state: string | undefined): boolean {
  return state != null && terminalStates.has(state);
}

const stateLabels: Record<string, string> = {
  queued: "待機",
  validating: "確認中",
  converting: "変換中",
  transcribing: "認識中",
  exporting: "書き出し中",
  completed: "完了",
  failed: "失敗",
  cancelled: "キャンセル",
  interrupted: "中断",
};

/** ジョブ状態を画面用の日本語にする。未知の値はそのまま返す。 */
export function jobStateLabel(state: string | undefined): string {
  if (!state) {
    return "待機";
  }
  return stateLabels[state] ?? state;
}

const modelNames: Record<string, string> = {
  "large-v3": "高精度",
  "large-v3-turbo": "高速",
};

/** モデル ID を画面用の短い名前にする。 */
export function modelLabel(modelID: string): string {
  return modelNames[modelID] ?? modelID;
}

const modelStatuses: Record<string, string> = {
  available: "利用できます",
  missing: "未導入",
  partial: "取得が途中です",
};

/** モデルの導入状態を日本語にする。 */
export function modelStatusLabel(status: string): string {
  return modelStatuses[status] ?? status;
}

/** パスからファイル名だけを取り出す。 */
export function fileBaseName(path: string | undefined): string {
  const trimmed = path?.trim() ?? "";
  if (!trimmed) {
    return "無題";
  }
  const parts = trimmed.split(/[/\\]/);
  return parts[parts.length - 1] || trimmed;
}

function readableError(value: string): string {
  if (value.includes("reading 'main'") || value.includes("reading 'go'")) {
    return "アプリとの接続ができませんでした。開き直してください。";
  }
  return value;
}

/** Wails が文字列で返す失敗も、Error も、利用者向けの一文にする。 */
export function errorMessage(err: unknown): string {
  if (typeof err === "string") {
    const trimmed = err.trim();
    if (trimmed) {
      return readableError(trimmed);
    }
  }
  if (err instanceof Error) {
    const trimmed = err.message.trim();
    if (trimmed) {
      return readableError(trimmed);
    }
  }
  if (err && typeof err === "object" && "message" in err) {
    const message = (err as { message?: unknown }).message;
    if (typeof message === "string" && message.trim()) {
      return readableError(message.trim());
    }
  }
  return "処理に失敗しました";
}
