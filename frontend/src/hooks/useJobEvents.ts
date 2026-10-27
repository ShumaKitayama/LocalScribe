import { useEffect, useRef, useState } from "react";
import { EventsOn } from "../../wailsjs/runtime/runtime";
import { domain } from "../../wailsjs/go/models";

/** この画面で開始したジョブの進捗を、ID ごとに保持する。 */
export function useTrackedJobs(jobIDs: readonly string[]) {
  const [byID, setByID] = useState<Record<string, domain.TranscriptionJob>>({});
  const idsRef = useRef(jobIDs);
  idsRef.current = jobIDs;

  useEffect(() => {
    const runtime = (window as Window & { runtime?: { EventsOnMultiple?: unknown } }).runtime;
    if (!runtime?.EventsOnMultiple) {
      return;
    }
    let cancel = () => {};
    try {
      cancel = EventsOn("job:progress", (payload: domain.TranscriptionJob) => {
        const id = payload?.id;
        if (!id || !idsRef.current.includes(id)) {
          return;
        }
        setByID((prev) => ({
          ...prev,
          [id]: domain.TranscriptionJob.createFrom(payload),
        }));
      });
    } catch {
      return;
    }
    return () => {
      try {
        cancel();
      } catch {
        // ランタイムが既に無いときは購読解除を諦める。
      }
    };
  }, []);

  return jobIDs.map((id) => byID[id] ?? null);
}
