import { useCallback, useRef, useState } from "react";
import "./App.css";
import { SaveRecording } from "../wailsjs/go/main/App";
import type { main } from "../wailsjs/go/models";

function App() {
  const [status, setStatus] = useState("マイクの許可後、約3秒録音して停止してください。");
  const [savedPath, setSavedPath] = useState("");
  const [recording, setRecording] = useState(false);
  const recorderRef = useRef<MediaRecorder | null>(null);
  const chunksRef = useRef<BlobPart[]>([]);
  const streamRef = useRef<MediaStream | null>(null);

  const stopTracks = useCallback(() => {
    streamRef.current?.getTracks().forEach((track) => track.stop());
    streamRef.current = null;
  }, []);

  const startRecording = useCallback(async () => {
    setSavedPath("");
    if (!navigator.mediaDevices?.getUserMedia) {
      setStatus("この環境では MediaRecorder を利用できません。");
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
          const result: main.SaveRecordingResult = await SaveRecording(bytes);
          setSavedPath(result.path);
          setStatus(`保存しました（${result.size_bytes} バイト）`);
        } catch (err) {
          const message = err instanceof Error ? err.message : String(err);
          setStatus(`保存に失敗しました: ${message}`);
        }
      };
      recorderRef.current = recorder;
      recorder.start();
      setRecording(true);
      setStatus("録音中です。約3秒後に停止してください。");
    } catch (err) {
      const message = err instanceof Error ? err.message : String(err);
      setStatus(`マイクを開始できません: ${message}`);
    }
  }, [stopTracks]);

  const stopRecording = useCallback(() => {
    const recorder = recorderRef.current;
    if (recorder && recorder.state !== "inactive") {
      recorder.stop();
    }
  }, []);

  return (
    <div id="App">
      <h1>LocalScribe — マイク確認（フェーズ 9）</h1>
      <p id="status">{status}</p>
      <div className="controls">
        <button className="btn" onClick={startRecording} disabled={recording}>
          録音開始
        </button>
        <button className="btn" onClick={stopRecording} disabled={!recording}>
          録音停止
        </button>
      </div>
      {savedPath ? (
        <p className="path">
          保存先: <code>{savedPath}</code>
        </p>
      ) : null}
    </div>
  );
}

export default App;
