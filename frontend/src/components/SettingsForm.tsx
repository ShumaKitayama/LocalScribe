import { modelLabel, modelStatusLabel } from "../lib/display";
import type { domain } from "../../wailsjs/go/models";

type Props = {
  settings: domain.AppSettings;
  models: domain.ModelInfo[];
  error?: string;
  onChange: (settings: domain.AppSettings) => void;
  onPickOutputDir: () => void;
};

export function SettingsForm({ settings, models, error, onChange, onPickOutputDir }: Props) {
  const current = models.find((model) => model.id === settings.default_model_id);
  const unavailable = current != null && current.status !== "available";

  return (
    <section className="settings-form">
      <h2>文字起こしの設定</h2>
      <label>
        モデル
        <select
          value={settings.default_model_id}
          onChange={(e) => onChange({ ...settings, default_model_id: e.target.value })}
        >
          {models.map((model) => (
            <option key={model.id} value={model.id} disabled={model.status !== "available"}>
              {modelLabel(model.id)}（{modelStatusLabel(model.status)}）
            </option>
          ))}
        </select>
      </label>
      {unavailable ? (
        <p className="error" role="alert">
          このモデルは未導入です。ターミナルで models install を実行してください。
        </p>
      ) : null}
      <label>
        言語
        <select
          value={settings.language}
          onChange={(e) => onChange({ ...settings, language: e.target.value })}
        >
          <option value="ja">日本語</option>
          <option value="en">英語</option>
          <option value="auto">自動</option>
        </select>
      </label>
      <div className="output-dir">
        <span className="field-label">保存先</span>
        <p className="path-value" title={settings.output_dir}>
          {settings.output_dir}
        </p>
        <button type="button" className="btn btn-secondary" onClick={onPickOutputDir}>
          保存先を変更
        </button>
      </div>
      <details className="prompt-details">
        <summary>初期プロンプト</summary>
        <label>
          固有名詞など、認識の手がかり
          <textarea
            value={settings.initial_prompt}
            rows={3}
            onChange={(e) => onChange({ ...settings, initial_prompt: e.target.value })}
          />
        </label>
      </details>
      {error ? (
        <p className="error" role="alert">
          {error}
        </p>
      ) : null}
    </section>
  );
}
