import { useCallback, useEffect, useRef, useState } from "react";
import {
  ExportTranscript,
  GetJob,
  GetTranscript,
  RevealOutputFolder,
  UpdateTranscript,
} from "../../wailsjs/go/main/App";
import { AudioPlayer, type AudioControls } from "../components/AudioPlayer";
import { TranscriptEditor } from "../components/TranscriptEditor";
import { errorMessage, fileBaseName } from "../lib/display";
import { domain } from "../../wailsjs/go/models";

type Props = {
  jobID: string;
  transcriptID: string;
  onBack: () => void;
};

export function ResultPage({ jobID, transcriptID, onBack }: Props) {
  const [transcript, setTranscript] = useState<domain.Transcript | null>(null);
  const [resultDir, setResultDir] = useState("");
  const [loadError, setLoadError] = useState("");
  const [exportError, setExportError] = useState("");
  const [status, setStatus] = useState("");
  const playerRef = useRef<AudioControls>(null);

  const reload = useCallback(async () => {
    const [t, job] = await Promise.all([GetTranscript(transcriptID), GetJob(jobID)]);
    setTranscript(domain.Transcript.createFrom(t));
    setResultDir(job.result_dir ?? "");
  }, [jobID, transcriptID]);

  useEffect(() => {
    void reload().catch((err) => {
      setLoadError(errorMessage(err));
    });
  }, [reload]);

  const handleSaveSegment = useCallback(
    async (index: number, editedText: string) => {
      setStatus("");
      await UpdateTranscript(transcriptID, [
        domain.SegmentEdit.createFrom({ index, edited_text: editedText }),
      ]);
      await reload();
      setStatus("保存しました");
    },
    [reload, transcriptID],
  );

  const exportFormats = useCallback(
    async (formats: string[]) => {
      setExportError("");
      setStatus("");
      try {
        await ExportTranscript(transcriptID, formats);
        setStatus("書き出しました");
      } catch (err) {
        setExportError(errorMessage(err));
      }
    },
    [transcriptID],
  );

  const title = fileBaseName(transcript?.source_path);

  return (
    <div className="result-page">
      <header className="result-header">
        <button type="button" className="btn btn-secondary" onClick={onBack}>
          ホームへ戻る
        </button>
        <div className="result-title">
          <p className="eyebrow">文字起こし結果</p>
          <h1 title={transcript?.source_path}>{title}</h1>
        </div>
      </header>
      {loadError ? (
        <p className="error banner" role="alert">
          {loadError}
        </p>
      ) : null}
      {status ? (
        <p className="status" role="status">
          {status}
        </p>
      ) : null}
      <div className="result-layout">
        <AudioPlayer ref={playerRef} jobID={jobID} sourcePath={transcript?.source_path ?? ""} />
        {transcript ? (
          <TranscriptEditor
            transcript={transcript}
            onSaveSegment={handleSaveSegment}
            onSeek={(ms) => playerRef.current?.seek(ms)}
            onCopyResult={setStatus}
          />
        ) : null}
      </div>
      <div className="export-bar">
        <button type="button" className="btn btn-secondary" onClick={() => void exportFormats(["txt"])}>
          TXT を再出力
        </button>
        <button type="button" className="btn btn-secondary" onClick={() => void exportFormats(["srt"])}>
          SRT を再出力
        </button>
        <button type="button" className="btn btn-secondary" onClick={() => void exportFormats(["json"])}>
          JSON を再出力
        </button>
        <button
          type="button"
          className="btn btn-secondary"
          disabled={!resultDir}
          onClick={() => {
            if (resultDir) {
              void RevealOutputFolder(resultDir).catch((err) => {
                setExportError(errorMessage(err));
              });
            }
          }}
        >
          出力フォルダを開く
        </button>
        {exportError ? (
          <p className="error" role="alert">
            {exportError}
          </p>
        ) : null}
      </div>
    </div>
  );
}
