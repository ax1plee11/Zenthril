/**
 * ThemeSettings — модальное окно настройки темы
 */

import React, { useRef } from "react";
import { useTheme, DEFAULT_THEME, saveTheme, applyTheme } from "../store/theme";
import type { Theme } from "../store/theme";

interface Props {
  onClose: () => void;
}

const overlay: React.CSSProperties = {
  position: "fixed",
  inset: 0,
  background: "rgba(0,0,0,0.7)",
  display: "flex",
  alignItems: "center",
  justifyContent: "center",
  zIndex: 1000,
};

const modal: React.CSSProperties = {
  background: "var(--bg-secondary, #2f3136)",
  borderRadius: 8,
  padding: 24,
  width: 400,
  color: "var(--text-primary, #dcddde)",
  display: "flex",
  flexDirection: "column",
  gap: 20,
};

const title: React.CSSProperties = {
  fontSize: 18,
  fontWeight: 700,
  marginBottom: 4,
};

const label: React.CSSProperties = {
  fontSize: 12,
  fontWeight: 600,
  textTransform: "uppercase",
  color: "var(--text-muted, #72767d)",
  marginBottom: 8,
};

const schemeRow: React.CSSProperties = {
  display: "flex",
  gap: 8,
};

const btnBase: React.CSSProperties = {
  flex: 1,
  padding: "8px 0",
  borderRadius: 4,
  border: "2px solid transparent",
  cursor: "pointer",
  fontWeight: 600,
  fontSize: 13,
  transition: "border-color 0.15s",
};

const row: React.CSSProperties = {
  display: "flex",
  alignItems: "center",
  gap: 10,
};

const inputText: React.CSSProperties = {
  flex: 1,
  background: "var(--bg-tertiary, #202225)",
  border: "none",
  borderRadius: 4,
  padding: "8px 10px",
  color: "var(--text-primary, #dcddde)",
  fontSize: 14,
};

const actionBtn: React.CSSProperties = {
  padding: "8px 16px",
  borderRadius: 4,
  border: "none",
  cursor: "pointer",
  fontWeight: 600,
  fontSize: 13,
};

export default function ThemeSettings({ onClose }: Props) {
  const { theme, setTheme } = useTheme();
  const fileRef = useRef<HTMLInputElement>(null);

  function update(patch: Partial<Theme>) {
    const next = { ...theme, ...patch };
    applyTheme(next);
    saveTheme(next);
    setTheme(next);
  }

  function handleBgFile(e: React.ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0];
    if (!file) return;
    const reader = new FileReader();
    reader.onload = () => update({ chatBackground: reader.result as string });
    reader.readAsDataURL(file);
  }

  function handleReset() {
    applyTheme(DEFAULT_THEME);
    saveTheme(DEFAULT_THEME);
    setTheme({ ...DEFAULT_THEME });
  }

  const schemes: Array<{ key: Theme["scheme"]; label: string }> = [
    { key: "dark", label: "Тёмная" },
    { key: "light", label: "Светлая" },
    { key: "custom", label: "Кастомная" },
  ];

  return (
    <div style={overlay} onClick={onClose}>
      <div style={modal} onClick={(e) => e.stopPropagation()}>
        <div style={title}>Настройки темы</div>

        {/* Схема */}
        <div>
          <div style={label}>Схема</div>
          <div style={schemeRow}>
            {schemes.map((s) => (
              <button
                key={s.key}
                style={{
                  ...btnBase,
                  background: theme.scheme === s.key ? "var(--accent, #7289da)" : "var(--bg-tertiary, #202225)",
                  color: theme.scheme === s.key ? "#fff" : "var(--text-primary, #dcddde)",
                  borderColor: theme.scheme === s.key ? "var(--accent, #7289da)" : "transparent",
                }}
                onClick={() => update({ scheme: s.key })}
              >
                {s.label}
              </button>
            ))}
          </div>
        </div>

        {/* Акцентный цвет */}
        <div>
          <div style={label}>Акцентный цвет</div>
          <div style={row}>
            <input
              type="color"
              value={theme.accentColor}
              onChange={(e) => update({ accentColor: e.target.value })}
              style={{ width: 40, height: 36, border: "none", cursor: "pointer", borderRadius: 4, background: "none" }}
            />
            <span style={{ fontSize: 14 }}>{theme.accentColor}</span>
          </div>
        </div>

        {/* Фон чата */}
        <div>
          <div style={label}>Фон чата</div>
          <div style={row}>
            <input
              type="text"
              placeholder="URL изображения..."
              value={theme.chatBackground ?? ""}
              onChange={(e) => update({ chatBackground: e.target.value || null })}
              style={inputText}
            />
            <button
              style={{ ...actionBtn, background: "var(--bg-tertiary, #202225)", color: "var(--text-primary, #dcddde)" }}
              onClick={() => fileRef.current?.click()}
            >
              📁
            </button>
            <input ref={fileRef} type="file" accept="image/*" style={{ display: "none" }} onChange={handleBgFile} />
          </div>
        </div>

        {/* Кнопки */}
        <div style={{ display: "flex", justifyContent: "space-between" }}>
          <button
            style={{ ...actionBtn, background: "var(--bg-tertiary, #202225)", color: "var(--text-muted, #72767d)" }}
            onClick={handleReset}
          >
            Сбросить к умолчаниям
          </button>
          <button
            style={{ ...actionBtn, background: "var(--accent, #7289da)", color: "#fff" }}
            onClick={onClose}
          >
            Закрыть
          </button>
        </div>
      </div>
    </div>
  );
}
