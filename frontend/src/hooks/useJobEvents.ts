import { useEffect, useState } from "react";
import { EventsOn } from "../../wailsjs/runtime/runtime";
import { domain } from "../../wailsjs/go/models";

export function useJobEvents(activeJobID: string | null) {
  const [job, setJob] = useState<domain.TranscriptionJob | null>(null);

  useEffect(() => {
    if (!activeJobID) {
      setJob(null);
      return;
    }
    const cancel = EventsOn("job:progress", (payload: domain.TranscriptionJob) => {
      if (payload?.id === activeJobID) {
        setJob(domain.TranscriptionJob.createFrom(payload));
      }
    });
    return () => {
      cancel();
    };
  }, [activeJobID]);

  return job;
}
