import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { JobProgress } from "./JobProgress";
import { domain } from "../../wailsjs/go/models";

function makeJob(percent?: number): domain.TranscriptionJob {
  return domain.TranscriptionJob.createFrom({
    id: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
    state: "transcribing",
    progress: {
      job_id: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
      state: "transcribing",
      message: "音声を認識しています",
      elapsed_ms: 1200,
      percent,
    },
  });
}

describe("JobProgress", () => {
  it("does not show percent when progress percent is missing", () => {
    render(<JobProgress job={makeJob()} onCancel={() => {}} />);
    expect(screen.queryByText(/%/)).toBeNull();
  });

  it("shows percent when measured", () => {
    render(<JobProgress job={makeJob(55)} onCancel={() => {}} />);
    expect(screen.getByText("55%")).toBeTruthy();
  });
});
