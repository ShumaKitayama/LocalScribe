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
import { useTrackedJobs } from "../hooks/useJobEvents";
import { errorMessage, fileBaseName, isTerminalJobState, jobStateLabel } from "../lib/display";
import { domain } from "../../wailsjs/go/models";

type Props = {
  onJobFinished: (job: domain.TranscriptionJob) => void;
};

export function MainPage({ onJobFinished }: Props) {
  const [settings, setSettings] = useState<domain.AppSettings | null>(null);
  const [models, setModels] = useState<domain.ModelInfo[]>([]);
  const [recoverable, setRecoverable] = useState<domain.TranscriptionJob[]>([]);
  const [loadError, setLoadError] = useState("");
  const [actionError, setActionError] = useState("");
  const [recordError, setRecordError] = useState("");
  const [settingsError, setSettingsError] = useState("");
  const [sessionIDs, setSessionIDs] = useState<string[]>([]);
  const tracked = useTrackedJobs(sessionIDs);

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
        setLoadError(errorMessage(err));
      }
    })();
  }, []);

  const running =
    tracked.find((job) => job && !isTerminalJobState(job.state)) ?? null;
  const single = sessionIDs.length === 1 ? tracked[0] : null;

  useEffect(() => {
    if (single?.state === "completed" && single.transcript_id) {
      onJobFinished(single);
    }
  }, [onJobFinished, single]);

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

  const rememberJob = useCallback((jobID: string) => {
    setSessionIDs((prev) => (prev.includes(jobID) ? prev : [...prev, jobID]));
  }, []);

  const enqueuePaths = useCallback(
    async (paths: string[]) => {
      if (!settings) {
        return;
      }
      setActionError("");
      for (const path of paths) {
        try {
          const req = buildRequest(path, "file");
          const jobID = await StartTranscription(req);
          rememberJob(jobID);
        } catch (err) {
          setActionError(errorMessage(err));
        }
      }
    },
    [buildRequest, rememberJob, settings],
  );

  const handlePickFiles = useCallback(async () => {
    setActionError("");
    try {
      const paths = await PickAudioFiles();
      if (paths?.length) {
        await enqueuePaths(paths);
      }
    } catch (err) {
      setActionError(errorMessage(err));
    }
  }, [enqueuePaths]);

  const persistSettings = useCallback(async (next: domain.AppSettings) => {
    setSettings(next);
    setSettingsError("");
    try {
      await UpdateSettings(next);
    } catch (err) {
      setSettingsError(errorMessage(err));
    }
  }, []);

  const handlePickOutput = useCallback(async () => {
    if (!settings) {
      return;
    }
    setSettingsError("");
    try {
      const dir = await PickOutputDirectory();
      if (dir) {
        await persistSettings({ ...settings, output_dir: dir });
      }
    } catch (err) {
      setSettingsError(errorMessage(err));
    }
  }, [persistSettings, settings]);

  const handleCancel = useCallback(async () => {
    if (!running) {
      return;
    }
    setActionError("");
    try {
      await CancelJob(running.id);
    } catch (err) {
      setActionError(errorMessage(err));
    }
  }, [running]);

  const retryRecoverable = useCallback(
    async (job: domain.TranscriptionJob) => {
      if (!settings) {
        return;
      }
      setActionError("");
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
        rememberJob(id);
      } catch (err) {
        setActionError(errorMessage(err));
      }
    },
    [rememberJob, settings],
  );

  const busy = running != null;
  const ready = Boolean(settings) && !loadError;

  return (
    <div className="app-shell">
      <aside className="sidebar">
        <div className="brand">
          <p className="brand-mark">LocalScribe</p>
          <p className="brand-sub">オフラインで文字起こし</p>
        </div>
        <button
          type="button"
          className="btn btn-primary"
          onClick={() => void handlePickFiles()}
          disabled={!ready || busy}
        >
          ファイルを追加
        </button>
        <RecordButton
          disabled={!ready || busy}
          buildRequest={buildRequest}
          onJobStarted={rememberJob}
          onError={setRecordError}
        />
        {recordError ? (
          <p className="error" role="alert">
            {recordError}
          </p>
        ) : null}
        {settings ? (
          <SettingsForm
            settings={settings}
            models={models}
            error={settingsError}
            onChange={(next) => void persistSettings(next)}
            onPickOutputDir={() => void handlePickOutput()}
          />
        ) : null}
        {recoverable.length > 0 ? (
          <section className="recoverable">
            <h2>中断されたジョブ</h2>
            <ul>
              {recoverable.map((item) => (
                <li key={item.id}>
                  <span className="path-value" title={item.request?.source_path}>
                    {item.request?.source_path}
                  </span>
                  <button
                    type="button"
                    className="btn btn-secondary"
                    onClick={() => void retryRecoverable(item)}
                    disabled={!ready || busy}
                  >
                    新規ジョブとして再実行
                  </button>
                </li>
              ))}
            </ul>
          </section>
        ) : null}
      </aside>
      <main className="workspace">
        {loadError ? (
          <p className="error banner" role="alert">
            設定を読み込めませんでした。{loadError}
          </p>
        ) : null}
        {actionError ? (
          <p className="error banner" role="alert">
            {actionError}
          </p>
        ) : null}
        <JobProgress job={running} onCancel={() => void handleCancel()} />
        {busy ? null : (
          <DropZone
            disabled={!ready}
            onFiles={(paths) => void enqueuePaths(paths)}
            onPick={() => void handlePickFiles()}
          />
        )}
        {sessionIDs.length > 1 ? (
          <section className="session-queue" aria-label="この回のファイル">
            <h2>この回のファイル</h2>
            <ul>
              {sessionIDs.map((id, index) => {
                const job = tracked[index];
                const done = job?.state === "completed" && Boolean(job.transcript_id);
                return (
                  <li key={id}>
                    <span className="queue-name">{fileBaseName(job?.request?.source_path)}</span>
                    <span className="queue-state">{job ? jobStateLabel(job.state) : "待機"}</span>
                    {done ? (
                      <button type="button" className="btn btn-secondary" onClick={() => onJobFinished(job)}>
                        結果を見る
                      </button>
                    ) : null}
                    {job?.error?.message ? <span className="error">{job.error.message}</span> : null}
                  </li>
                );
              })}
            </ul>
          </section>
        ) : null}
      </main>
    </div>
  );
}
