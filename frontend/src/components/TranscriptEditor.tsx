import { useCallback, useState } from "react";
import { segmentDisplayText } from "../lib/display";
import type { domain } from "../../wailsjs/go/models";

type Props = {
  transcript: domain.Transcript;
  onSaveSegment: (index: number, editedText: string) => Promise<void>;
};

function formatTime(ms: number): string {
  const totalSec = Math.floor(ms / 1000);
  const min = Math.floor(totalSec / 60);
  const sec = totalSec % 60;
  return `${min}:${sec.toString().padStart(2, "0")}`;
}

export function TranscriptEditor({ transcript, onSaveSegment }: Props) {
  const [editingIndex, setEditingIndex] = useState<number | null>(null);
  const [draft, setDraft] = useState("");

  const startEdit = useCallback((index: number, current: string) => {
    setEditingIndex(index);
    setDraft(current);
  }, []);

  const saveEdit = useCallback(async () => {
    if (editingIndex == null) {
      return;
    }
    await onSaveSegment(editingIndex, draft);
    setEditingIndex(null);
    setDraft("");
  }, [draft, editingIndex, onSaveSegment]);

  const copyAll = useCallback(() => {
    const text = transcript.segments
      .map((seg) => segmentDisplayText(seg))
      .join("\n");
    void navigator.clipboard.writeText(text);
  }, [transcript.segments]);

  return (
    <section className="transcript-editor">
      <div className="transcript-toolbar">
        <button type="button" className="btn btn-secondary" onClick={copyAll}>
          全文コピー
        </button>
      </div>
      <ol className="segment-list">
        {transcript.segments.map((seg, index) => {
          const display = segmentDisplayText(seg);
          const isEditing = editingIndex === index;
          return (
            <li key={`${seg.start_ms}-${index}`} className="segment-item">
              <button
                type="button"
                className="segment-time"
                data-seek-ms={seg.start_ms}
              >
                {formatTime(seg.start_ms)}
              </button>
              {isEditing ? (
                <div className="segment-edit">
                  <textarea value={draft} onChange={(e) => setDraft(e.target.value)} rows={3} />
                  <div className="segment-edit-actions">
                    <button type="button" className="btn" onClick={() => void saveEdit()}>
                      保存
                    </button>
                    <button
                      type="button"
                      className="btn btn-secondary"
                      onClick={() => setEditingIndex(null)}
                    >
                      キャンセル
                    </button>
                  </div>
                  {seg.text ? <p className="segment-original">認識原文: {seg.text}</p> : null}
                </div>
              ) : (
                <div className="segment-body">
                  <p>{display}</p>
                  {seg.edited_text != null && seg.text ? (
                    <p className="segment-original">認識原文: {seg.text}</p>
                  ) : null}
                  <button
                    type="button"
                    className="btn-link"
                    onClick={() => startEdit(index, display)}
                  >
                    編集
                  </button>
                </div>
              )}
            </li>
          );
        })}
      </ol>
    </section>
  );
}
