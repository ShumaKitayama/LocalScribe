import { useCallback, useState } from "react";
import { errorMessage, segmentDisplayText } from "../lib/display";
import type { domain } from "../../wailsjs/go/models";

type Props = {
  transcript: domain.Transcript;
  onSaveSegment: (index: number, editedText: string) => Promise<void>;
  onSeek: (ms: number) => void;
  onCopyResult: (message: string) => void;
};

function formatTime(ms: number): string {
  const totalSec = Math.floor(ms / 1000);
  const min = Math.floor(totalSec / 60);
  const sec = totalSec % 60;
  return `${min}:${sec.toString().padStart(2, "0")}`;
}

export function TranscriptEditor({ transcript, onSaveSegment, onSeek, onCopyResult }: Props) {
  const [editingIndex, setEditingIndex] = useState<number | null>(null);
  const [draft, setDraft] = useState("");
  const [editError, setEditError] = useState("");
  const [copyError, setCopyError] = useState("");
  const segments = transcript.segments ?? [];

  const startEdit = useCallback((index: number, current: string) => {
    setEditingIndex(index);
    setDraft(current);
    setEditError("");
  }, []);

  const saveEdit = useCallback(async () => {
    if (editingIndex == null) {
      return;
    }
    setEditError("");
    try {
      await onSaveSegment(editingIndex, draft);
      setEditingIndex(null);
      setDraft("");
    } catch (err) {
      setEditError(errorMessage(err));
    }
  }, [draft, editingIndex, onSaveSegment]);

  const copyAll = useCallback(async () => {
    const text = segments.map((seg) => segmentDisplayText(seg)).join("\n");
    setCopyError("");
    try {
      await navigator.clipboard.writeText(text);
      onCopyResult("全文をコピーしました");
    } catch (err) {
      setCopyError(errorMessage(err));
    }
  }, [onCopyResult, segments]);

  return (
    <section className="transcript-editor">
      <div className="transcript-toolbar">
        <h2>文字起こし</h2>
        <button type="button" className="btn btn-secondary" onClick={() => void copyAll()}>
          全文コピー
        </button>
      </div>
      {copyError ? (
        <p className="error" role="alert">
          {copyError}
        </p>
      ) : null}
      {segments.length === 0 ? <p className="hint">セグメントがありません。</p> : null}
      <ol className="segment-list">
        {segments.map((seg, index) => {
          const display = segmentDisplayText(seg);
          const isEditing = editingIndex === index;
          return (
            <li key={`${seg.start_ms}-${index}`} className="segment-item">
              <button
                type="button"
                className="segment-time tabular"
                aria-label={`${formatTime(seg.start_ms)} から再生`}
                onClick={() => onSeek(seg.start_ms)}
              >
                {formatTime(seg.start_ms)}
              </button>
              {isEditing ? (
                <div className="segment-edit">
                  <textarea
                    value={draft}
                    aria-label="セグメントの編集"
                    onChange={(e) => setDraft(e.target.value)}
                    rows={3}
                  />
                  <div className="segment-edit-actions">
                    <button type="button" className="btn btn-primary" onClick={() => void saveEdit()}>
                      保存
                    </button>
                    <button
                      type="button"
                      className="btn btn-secondary"
                      onClick={() => {
                        setEditingIndex(null);
                        setEditError("");
                      }}
                    >
                      キャンセル
                    </button>
                  </div>
                  {editError ? (
                    <p className="error" role="alert">
                      {editError}
                    </p>
                  ) : null}
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
