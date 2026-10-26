import { useCallback, useEffect, useState } from "react";
import {
  ExportTranscript,
  GetJob,
  GetTranscript,
  RevealOutputFolder,
  UpdateTranscript,
} from "../../wailsjs/go/main/App";
import { AudioPlayer } from "../components/AudioPlayer";
import { TranscriptEditor } from "../components/TranscriptEditor";
import { domain } from "../../wailsjs/go/models";

type Props = {
  jobID: string;
  transcriptID: string;
  onBack: () => void;
};

export function ResultPage({ jobID, transcriptID, onBack }: Props) {
  const [transcript, setTranscript] = useState<domain.Transcript | null>(null);
  const [resultDir, setResultDir] = useState("");
  const [error, setError] = useState("");
  const [status, setStatus] = useState("");

  const reload = useCallback(async () => {
    const [t, job] = await Promise.all([GetTranscript(transcriptID), GetJob(jobID)]);
    setTranscript(domain.Transcript.createFrom(t));
    setResultDir(job.result_dir ?? "");
  }, [jobID, transcriptID]);

  useEffect(() => {
    void reload().catch((err) => {
      setError(err instanceof Error ? err.message : String(err));
    });
  }, [reload]);

  const handleSaveSegment = useCallback(
    async (index: number, editedText: string) => {
      setError("");
      try {
        await UpdateTranscript(transcriptID, [
          domain.SegmentEdit.createFrom({ index, edited_text: editedText }),
        ]);
        await reload();
        setStatus("保存しました");
      } catch (err) {
        setError(err instanceof Error ? err.message : String(err));
      }
    },
    [reload, transcriptID],
  );

  const exportFormats = useCallback(async (formats: string[]) => {
    setError("");
    try {
      await ExportTranscript(transcriptID, formats);
      setStatus("書き出しました");
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    }
  }, [transcriptID]);

  return (
    <div className="result-page">
      <header className="result-header">
        <button type="button" className="btn btn-secondary" onClick={onBack}>
          戻る
        </button>
        <h1>文字起こし結果</h1>
      </header>
      {error ? <p className="error">{error}</p> : null}
      {status ? <p className="status">{status}</p> : null}
      <div className="result-layout">
        <AudioPlayer jobID={jobID} sourcePath={transcript?.source_path ?? ""} />
        {transcript ? (
          <TranscriptEditor transcript={transcript} onSaveSegment={handleSaveSegment} />
        ) : null}
      </div>
      <div className="export-actions">
        <button
          type="button"
          className="btn btn-secondary"
          onClick={() => void exportFormats(["txt"])}
        >
          TXT を再出力
        </button>
        <button
          type="button"
          className="btn btn-secondary"
          onClick={() => void exportFormats(["srt"])}
        >
          SRT を再出力
        </button>
        <button
          type="button"
          className="btn btn-secondary"
          onClick={() => void exportFormats(["json"])}
        >
          JSON を再出力
        </button>
        <button
          type="button"
          className="btn btn-secondary"
          onClick={() => {
            if (resultDir) {
              void RevealOutputFolder(resultDir);
            }
          }}
        >
          出力フォルダを開く
        </button>
      </div>
    </div>
  );
}
