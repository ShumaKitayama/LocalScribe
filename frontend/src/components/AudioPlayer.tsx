import { useCallback, useEffect, useRef, useState } from "react";
type Props = {
  jobID: string;
  sourcePath?: string;
};

const WEBM_HINT = "この形式はアプリ内で再生できない場合があります。";

export function AudioPlayer({ jobID, sourcePath }: Props) {
  const audioRef = useRef<HTMLAudioElement>(null);
  const [playbackError, setPlaybackError] = useState(false);
  const src = `/job-audio/${jobID}`;

  useEffect(() => {
    setPlaybackError(false);
  }, [jobID]);

  const handleError = useCallback(() => {
    setPlaybackError(true);
  }, []);

  useEffect(() => {
    const root = document.querySelector(".transcript-editor");
    if (!root || !audioRef.current) {
      return;
    }
    const onClick = (event: Event) => {
      const target = event.target as HTMLElement;
      if (!target.classList.contains("segment-time")) {
        return;
      }
      const ms = Number(target.dataset.seekMs ?? "0");
      const audio = audioRef.current;
      if (audio && Number.isFinite(ms)) {
        audio.currentTime = ms / 1000;
        void audio.play();
      }
    };
    root.addEventListener("click", onClick);
    return () => root.removeEventListener("click", onClick);
  }, [jobID]);

  return (
    <section className="audio-player">
      <audio ref={audioRef} controls src={src} onError={handleError} />
      {sourcePath?.toLowerCase().endsWith(".webm") ? <p className="hint">{WEBM_HINT}</p> : null}
      {playbackError ? <p className="error">この形式はアプリ内で再生できません。</p> : null}
    </section>
  );
}
