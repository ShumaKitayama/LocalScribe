import { forwardRef, useCallback, useEffect, useImperativeHandle, useRef, useState } from "react";

type Props = {
  jobID: string;
  sourcePath?: string;
};

export type AudioControls = {
  seek: (ms: number) => void;
};

const WEBM_HINT = "この形式はアプリ内で再生できない場合があります。";

export const AudioPlayer = forwardRef<AudioControls, Props>(function AudioPlayer(
  { jobID, sourcePath },
  ref,
) {
  const audioRef = useRef<HTMLAudioElement>(null);
  const [playbackError, setPlaybackError] = useState(false);
  const src = `/job-audio/${jobID}`;
  const webm = sourcePath?.toLowerCase().endsWith(".webm") ?? false;

  useEffect(() => {
    setPlaybackError(false);
  }, [jobID]);

  const handleError = useCallback(() => {
    setPlaybackError(true);
  }, []);

  useImperativeHandle(
    ref,
    () => ({
      seek(ms: number) {
        const audio = audioRef.current;
        if (!audio || !Number.isFinite(ms)) {
          return;
        }
        audio.currentTime = ms / 1000;
        void audio.play().catch(() => {
          setPlaybackError(true);
        });
      },
    }),
    [],
  );

  return (
    <section className="audio-player">
      <h2>再生</h2>
      <audio ref={audioRef} controls src={src} onError={handleError} />
      {webm ? <p className="hint">{WEBM_HINT}</p> : null}
      {playbackError ? (
        <p className="error" role="alert">
          この形式はアプリ内で再生できません。
        </p>
      ) : null}
    </section>
  );
});
