import { useCallback, useState } from "react";
import "./App.css";
import { MainPage } from "./pages/MainPage";
import { ResultPage } from "./pages/ResultPage";
import type { domain } from "../wailsjs/go/models";

type Screen =
  | { name: "main"; activeJobID: string | null }
  | { name: "result"; jobID: string; transcriptID: string };

function App() {
  const [screen, setScreen] = useState<Screen>({ name: "main", activeJobID: null });

  const onJobStarted = useCallback((jobID: string) => {
    setScreen({ name: "main", activeJobID: jobID });
  }, []);

  const onJobFinished = useCallback((job: domain.TranscriptionJob) => {
    if (!job.transcript_id) {
      return;
    }
    setScreen({
      name: "result",
      jobID: job.id,
      transcriptID: job.transcript_id,
    });
  }, []);

  const onBack = useCallback(() => {
    setScreen({ name: "main", activeJobID: null });
  }, []);

  if (screen.name === "result") {
    return (
      <ResultPage jobID={screen.jobID} transcriptID={screen.transcriptID} onBack={onBack} />
    );
  }

  return (
    <MainPage
      activeJobID={screen.activeJobID}
      onJobStarted={onJobStarted}
      onJobFinished={onJobFinished}
    />
  );
}

export default App;
