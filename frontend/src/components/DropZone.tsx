import { useEffect } from "react";
import { OnFileDrop, OnFileDropOff } from "../../wailsjs/runtime/runtime";

type Props = {
  disabled?: boolean;
  onFiles: (paths: string[]) => void;
  onPick: () => void;
};

export function DropZone({ disabled, onFiles, onPick }: Props) {
  useEffect(() => {
    const runtime = (window as Window & { runtime?: { OnFileDrop?: unknown } }).runtime;
    if (disabled || !runtime?.OnFileDrop) {
      return;
    }
    OnFileDrop((_x, _y, paths) => {
      if (paths.length > 0) {
        onFiles(paths);
      }
    }, true);
    return () => {
      OnFileDropOff();
    };
  }, [disabled, onFiles]);

  return (
    <div className={`drop-zone${disabled ? " drop-zone--disabled" : ""}`}>
      <p className="drop-title">音声をここにドロップ</p>
      <p className="hint">MP3 / M4A / WAV / FLAC / MP4 / MOV</p>
      <button type="button" className="btn btn-primary" onClick={onPick} disabled={disabled}>
        ファイルを選択
      </button>
    </div>
  );
}
