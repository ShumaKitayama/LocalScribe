import { formatProgressPercent } from "../lib/display";
import type { domain } from "../../wailsjs/go/models";

type Props = {
  job: domain.TranscriptionJob | null;
  onCancel: () => void;
};

function formatElapsed(ms: number): string {
  const totalSec = Math.floor(ms / 1000);
  const min = Math.floor(totalSec / 60);
  const sec = totalSec % 60;
  return `${min}:${sec.toString().padStart(2, "0")}`;
}

export function JobProgress({ job, onCancel }: Props) {
  if (!job) {
    return null;
  }
  const progress = job.progress;
  const percentLabel = formatProgressPercent(progress?.percent);
  const busy = !job.state || !["completed", "failed", "cancelled", "interrupted"].includes(job.state);

  return (
    <section className="job-progress" aria-live="polite">
      <p className="job-phase">{progress?.message ?? job.state}</p>
      <p className="job-meta">
        <span>{job.state}</span>
        {progress?.elapsed_ms != null && progress.elapsed_ms > 0 ? (
          <span>経過 {formatElapsed(progress.elapsed_ms)}</span>
        ) : null}
        {percentLabel ? <span>{percentLabel}</span> : null}
      </p>
      <div
        className={`progress-bar${percentLabel ? "" : " progress-bar--indeterminate"}`}
        role="progressbar"
        aria-valuemin={0}
        aria-valuemax={100}
        aria-valuenow={percentLabel ? Math.round(progress?.percent ?? 0) : undefined}
      >
        {percentLabel ? (
          <div className="progress-bar-fill" style={{ width: percentLabel }} />
        ) : (
          <div className="progress-bar-indeterminate" />
        )}
      </div>
      {busy ? (
        <button type="button" className="btn btn-secondary" onClick={onCancel}>
          キャンセル
        </button>
      ) : null}
      {job.error?.message ? <p className="error">{job.error.message}</p> : null}
    </section>
  );
}
