import { useEffect } from "react";
import { OnFileDrop, OnFileDropOff } from "../../wailsjs/runtime/runtime";

type Props = {
  disabled?: boolean;
  onFiles: (paths: string[]) => void;
};

export function DropZone({ disabled, onFiles }: Props) {
  useEffect(() => {
    if (disabled) {
      OnFileDropOff();
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
      <p>音声ファイルをここへドロップ</p>
      <p className="hint">mp3 / m4a / wav / flac / mp4 / mov</p>
    </div>
  );
}
