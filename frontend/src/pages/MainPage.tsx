import { useCallback, useEffect, useState } from "react";
import {
  CancelJob,
  GetSettings,
  ListModels,
  ListRecoverableJobs,
  PickAudioFiles,
  PickOutputDirectory,
  StartTranscription,
  UpdateSettings,
} from "../../wailsjs/go/main/App";
import { DropZone } from "../components/DropZone";
import { JobProgress } from "../components/JobProgress";
import { RecordButton } from "../components/RecordButton";
import { SettingsForm } from "../components/SettingsForm";
import { useJobEvents } from "../hooks/useJobEvents";
import { domain } from "../../wailsjs/go/models";

type Props = {
  activeJobID: string | null;
  onJobStarted: (jobID: string) => void;
  onJobFinished: (job: domain.TranscriptionJob) => void;
};

export function MainPage({ activeJobID, onJobStarted, onJobFinished }: Props) {
  const [settings, setSettings] = useState<domain.AppSettings | null>(null);
  const [models, setModels] = useState<domain.ModelInfo[]>([]);
  const [recoverable, setRecoverable] = useState<domain.TranscriptionJob[]>([]);
  const [error, setError] = useState("");
  const job = useJobEvents(activeJobID);

  useEffect(() => {
    void (async () => {
      try {
        const [s, m, r] = await Promise.all([
          GetSettings(),
          ListModels(),
          ListRecoverableJobs(),
        ]);
        setSettings(domain.AppSettings.createFrom(s));
        setModels(m.map((item) => domain.ModelInfo.createFrom(item)));
        setRecoverable(r.map((item) => domain.TranscriptionJob.createFrom(item)));
      } catch (err) {
        setError(err instanceof Error ? err.message : String(err));
      }
    })();
  }, []);

  useEffect(() => {
    if (job?.state === "completed" && job.transcript_id) {
      onJobFinished(job);
    }
  }, [job, onJobFinished]);

  const buildRequest = useCallback(
    (sourcePath: string, kind: string): domain.TranscriptionRequest => {
      if (!settings) {
        throw new Error("設定を読み込んでいます");
      }
      return domain.TranscriptionRequest.createFrom({
        source_path: sourcePath,
        source_kind: kind,
        model_id: settings.default_model_id,
        language: settings.language,
        output_dir: settings.output_dir,
        initial_prompt: settings.initial_prompt,
        export_formats: settings.export_formats,
      });
    },
    [settings],
  );

  const enqueuePaths = useCallback(
    async (paths: string[]) => {
      if (!settings) {
        return;
      }
      setError("");
      for (const path of paths) {
        try {
          const req = buildRequest(path, "file");
          const jobID = await StartTranscription(req);
          onJobStarted(jobID);
        } catch (err) {
          setError(err instanceof Error ? err.message : String(err));
        }
      }
    },
    [buildRequest, onJobStarted, settings],
  );

  const handlePickFiles = useCallback(async () => {
    try {
      const paths = await PickAudioFiles();
      if (paths?.length) {
        await enqueuePaths(paths);
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    }
  }, [enqueuePaths]);

  const persistSettings = useCallback(async (next: domain.AppSettings) => {
    setSettings(next);
    try {
      await UpdateSettings(next);
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    }
  }, []);

  const handlePickOutput = useCallback(async () => {
    if (!settings) {
      return;
    }
    try {
      const dir = await PickOutputDirectory();
      if (dir) {
        await persistSettings({ ...settings, output_dir: dir });
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    }
  }, [persistSettings, settings]);

  const handleCancel = useCallback(async () => {
    if (!activeJobID) {
      return;
    }
    try {
      await CancelJob(activeJobID);
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    }
  }, [activeJobID]);

  const retryRecoverable = useCallback(
    async (job: domain.TranscriptionJob) => {
      if (!settings) {
        return;
      }
      try {
        const req = domain.TranscriptionRequest.createFrom({
          source_path: job.request.source_path,
          source_kind: job.request.source_kind,
          model_id: settings.default_model_id,
          language: settings.language,
          output_dir: settings.output_dir,
          initial_prompt: settings.initial_prompt,
          export_formats: settings.export_formats,
        });
        const id = await StartTranscription(req);
        onJobStarted(id);
      } catch (err) {
        setError(err instanceof Error ? err.message : String(err));
      }
    },
    [onJobStarted, settings],
  );

  const busy = Boolean(
    activeJobID &&
      job &&
      !["completed", "failed", "cancelled", "interrupted"].includes(job.state),
  );

  return (
    <div className="main-page">
      <header>
        <h1>LocalScribe</h1>
        <p>音声をオフラインで文字起こしします。</p>
      </header>
      {error ? <p className="error">{error}</p> : null}
      {settings ? (
        <SettingsForm
          settings={settings}
          models={models}
          onChange={persistSettings}
          onPickOutputDir={() => void handlePickOutput()}
        />
      ) : null}
      <div className="main-actions">
        <button type="button" className="btn" onClick={() => void handlePickFiles()} disabled={busy}>
          ファイルを選択
        </button>
        <RecordButton
          disabled={busy || !settings}
          buildRequest={buildRequest}
          onJobStarted={onJobStarted}
          onError={setError}
        />
      </div>
      <DropZone disabled={busy || !settings} onFiles={(paths) => void enqueuePaths(paths)} />
      <JobProgress job={job} onCancel={() => void handleCancel()} />
      {recoverable.length > 0 ? (
        <section className="recoverable">
          <h2>中断されたジョブ</h2>
          <ul>
            {recoverable.map((item) => (
              <li key={item.id}>
                <span>{item.request?.source_path}</span>
                <button type="button" className="btn btn-secondary" onClick={() => void retryRecoverable(item)}>
                  新規ジョブとして再実行
                </button>
              </li>
            ))}
          </ul>
        </section>
      ) : null}
    </div>
  );
}
