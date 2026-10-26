import { useCallback, useRef, useState } from "react";
import { SaveRecording, StartTranscription } from "../../wailsjs/go/main/App";
import type { domain } from "../../wailsjs/go/models";

type Props = {
  disabled?: boolean;
  buildRequest: (sourcePath: string, kind: string) => domain.TranscriptionRequest;
  onJobStarted: (jobID: string) => void;
  onError: (message: string) => void;
};

export function RecordButton({ disabled, buildRequest, onJobStarted, onError }: Props) {
  const [recording, setRecording] = useState(false);
  const recorderRef = useRef<MediaRecorder | null>(null);
  const chunksRef = useRef<BlobPart[]>([]);
  const streamRef = useRef<MediaStream | null>(null);

  const stopTracks = useCallback(() => {
    streamRef.current?.getTracks().forEach((track) => track.stop());
    streamRef.current = null;
  }, []);

  const startRecording = useCallback(async () => {
    if (!navigator.mediaDevices?.getUserMedia) {
      onError("この環境では MediaRecorder を利用できません。");
      return;
    }
    try {
      const stream = await navigator.mediaDevices.getUserMedia({ audio: true });
      streamRef.current = stream;
      const recorder = new MediaRecorder(stream);
      chunksRef.current = [];
      recorder.ondataavailable = (event) => {
        if (event.data.size > 0) {
          chunksRef.current.push(event.data);
        }
      };
      recorder.onstop = async () => {
        setRecording(false);
        stopTracks();
        const blob = new Blob(chunksRef.current, { type: "audio/webm" });
        const buffer = await blob.arrayBuffer();
        const bytes = Array.from(new Uint8Array(buffer));
        try {
          const saved = await SaveRecording(bytes);
          const req = buildRequest(saved.path, "recording");
          const jobID = await StartTranscription(req);
          onJobStarted(jobID);
        } catch (err) {
          const message = err instanceof Error ? err.message : String(err);
          onError(message);
        }
      };
      recorderRef.current = recorder;
      recorder.start();
      setRecording(true);
    } catch (err) {
      const message = err instanceof Error ? err.message : String(err);
      onError(`マイクを開始できません: ${message}`);
    }
  }, [buildRequest, onError, onJobStarted, stopTracks]);

  const stopRecording = useCallback(() => {
    const recorder = recorderRef.current;
    if (recorder && recorder.state !== "inactive") {
      recorder.stop();
    }
  }, []);

  return (
    <div className="record-controls">
      <button type="button" className="btn" onClick={startRecording} disabled={disabled || recording}>
        録音開始
      </button>
      <button type="button" className="btn" onClick={stopRecording} disabled={!recording}>
        録音停止
      </button>
    </div>
  );
}
