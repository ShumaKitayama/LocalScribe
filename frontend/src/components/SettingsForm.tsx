import type { domain } from "../../wailsjs/go/models";

type Props = {
  settings: domain.AppSettings;
  models: domain.ModelInfo[];
  onChange: (settings: domain.AppSettings) => void;
  onPickOutputDir: () => void;
};

export function SettingsForm({ settings, models, onChange, onPickOutputDir }: Props) {
  return (
    <section className="settings-form">
      <h2>設定</h2>
      <label>
        モデル
        <select
          value={settings.default_model_id}
          onChange={(e) => onChange({ ...settings, default_model_id: e.target.value })}
        >
          {models.map((m) => (
            <option key={m.id} value={m.id} disabled={m.status !== "available"}>
              {m.id} ({m.status})
            </option>
          ))}
        </select>
      </label>
      <label>
        言語
        <select
          value={settings.language}
          onChange={(e) =>
            onChange({ ...settings, language: e.target.value })
          }
        >
          <option value="ja">日本語</option>
          <option value="en">英語</option>
          <option value="auto">自動</option>
        </select>
      </label>
      <label className="output-dir">
        保存先
        <div className="output-dir-row">
          <input type="text" readOnly value={settings.output_dir} />
          <button type="button" className="btn btn-secondary" onClick={onPickOutputDir}>
            選択…
          </button>
        </div>
      </label>
      <label>
        初期プロンプト
        <textarea
          value={settings.initial_prompt}
          rows={2}
          onChange={(e) => onChange({ ...settings, initial_prompt: e.target.value })}
        />
      </label>
    </section>
  );
}
