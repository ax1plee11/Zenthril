/**
 * ThemeSettings — модальное окно настройки темы
 * с анимированным превью фона и кнопкой "Применить"
 */

import React, { useRef, useState } from "react";
import { useTheme, DEFAULT_THEME, saveTheme, applyTheme } from "../store/theme";
import type { Theme } from "../store/theme";

interface Props {
  onClose: () => void;
}

// Предустановленные анимированные фоны
const ANIMATED_PRESETS = [
  {
    id: "aurora",
    label: "Аврора",
    colors: ["#0d1b2a", "#1b4332", "#0d1b2a", "#1a472a"],
    class: "animated-bg",
  },
  {
    id: "galaxy",
    label: "Галактика",
    colors: ["#1a1a2e", "#16213e", "#0f3460", "#533483"],
    class: "animated-bg",
  },
  {
    id: "sunset",
    label: "Закат",
    colors: ["#2d1b69", "#11998e", "#38ef7d", "#fc5c7d"],
    class: "animated-bg",
  },
  {
    id: "ocean",
    label: "Океан",
    colors: ["#0f2027", "#203a43", "#2c5364", "#1a6b8a"],
    class: "animated-bg",
  },
  {
    id: "fire",
    label: "Огонь",
    colors: ["#1a0000", "#7f0000", "#cc2200", "#ff6600"],
    class: "animated-bg",
  },
  {
    id: "custom",
    label: "Акцент",
    colors: [],
    class: "animated-bg-custom",
  },
];

const overlay: React.CSSProperties = {
  position: "fixed",
  inset: 0,
  background: "rgba(0,0,0,0.75)",
  display: "flex",
  alignItems: "center",
  justifyContent: "center",
  zIndex: 1000,
};

const modal: React.CSSProperties = {
  background: "var(--bg-secondary, #2f3136)",
  borderRadius: 12,
  padding: 24,
  width: 460,
  maxHeight: "90vh",
  overflowY: "auto",
  color: "var(--text-primary, #dcddde)",
  display: "flex",
  flexDirection: "column",
  gap: 20,
  boxShadow: "0 16px 48px rgba(0,0,0,0.6)",
};

const labelStyle: React.CSSProperties = {
  fontSize: 11,
  fontWeight: 700,
  textTransform: "uppercase",
  letterSpacing: 0.8,
  color: "var(--text-muted, #72767d)",
  marginBottom: 8,
};

const btnBase: React.CSSProperties = {
  flex: 1,
  padding: "8px 0",
  borderRadius: 6,
  border: "2px solid transparent",
  cursor: "pointer",
  fontWeight: 600,
  fontSize: 13,
  transition: "all 0.15s",
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
  borderRadius: 6,
  padding: "8px 10px",
  color: "var(--text-primary, #dcddde)",
  fontSize: 14,
  outline: "none",
};

export default function ThemeSettings({ onClose }: Props) {
  const { theme, setTheme } = useTheme();
  const fileRef = useRef<HTMLInputElement>(null);

  // Локальное состояние — применяется только по кнопке "Применить"
  const [draft, setDraft] = useState<Theme>({ ...theme });
  const [selectedPreset, setSelectedPreset] = useState<string | null>(null);
  const [applied, setApplied] = useState(false);

  function updateDraft(patch: Partial<Theme>) {
    setDraft((prev) => ({ ...prev, ...patch }));
    setApplied(false);
  }

  function handleApply() {
    applyTheme(draft);
    saveTheme(draft);
    setTheme(draft);
    setApplied(true);

    // Применяем анимированный фон если выбран пресет
    if (selectedPreset) {
      const preset = ANIMATED_PRESETS.find((p) => p.id === selectedPreset);
      if (preset) {
        const root = document.documentElement;
        if (preset.colors.length > 0) {
          preset.colors.forEach((c, i) => {
            root.style.setProperty(`--anim-color-${i + 1}`, c);
          });
        } else {
          // Для "custom" используем акцентный цвет
          root.style.setProperty("--anim-color-1", draft.accentColor + "44");
          root.style.setProperty("--anim-color-2", draft.accentColor + "22");
          root.style.setProperty("--anim-color-3", "var(--bg-primary)");
          root.style.setProperty("--anim-color-4", draft.accentColor + "11");
        }
        // Применяем класс к body
        document.body.className = preset.class;
      }
    } else if (!selectedPreset) {
      document.body.className = "";
    }
  }

  function handleBgFile(e: React.ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0];
    if (!file) return;
    if (file.size > 10 * 1024 * 1024) {
      alert("Файл слишком большой (макс. 10 МБ)");
      return;
    }
    const reader = new FileReader();
    reader.onload = () => {
      updateDraft({ chatBackground: reader.result as string });
      setSelectedPreset(null);
    };
    reader.readAsDataURL(file);
  }

  function handleReset() {
    setDraft({ ...DEFAULT_THEME });
    setSelectedPreset(null);
    setApplied(false);
    applyTheme(DEFAULT_THEME);
    saveTheme(DEFAULT_THEME);
    setTheme({ ...DEFAULT_THEME });
    document.body.className = "";
  }

  const schemes: Array<{ key: Theme["scheme"]; label: string }> = [
    { key: "dark", label: "🌙 Тёмная" },
    { key: "light", label: "☀️ Светлая" },
    { key: "custom", label: "🎨 Кастомная" },
  ];

  return (
    <div style={overlay} onClick={onClose}>
      <div style={modal} onClick={(e) => e.stopPropagation()}>
        <div style={{ fontSize: 18, fontWeight: 700 }}>🎨 Настройки темы</div>

        {/* Схема */}
        <div>
          <div style={labelStyle}>Цветовая схема</div>
          <div style={{ display: "flex", gap: 8 }}>
            {schemes.map((s) => (
              <button
                key={s.key}
                style={{
                  ...btnBase,
                  background: draft.scheme === s.key ? "var(--accent, #7289da)" : "var(--bg-tertiary, #202225)",
                  color: draft.scheme === s.key ? "#fff" : "var(--text-primary, #dcddde)",
                  borderColor: draft.scheme === s.key ? "var(--accent, #7289da)" : "transparent",
                }}
                onClick={() => updateDraft({ scheme: s.key })}
              >
                {s.label}
              </button>
            ))}
          </div>
        </div>

        {/* Акцентный цвет */}
        <div>
          <div style={labelStyle}>Акцентный цвет</div>
          <div style={row}>
            <input
              type="color"
              value={draft.accentColor}
              onChange={(e) => updateDraft({ accentColor: e.target.value })}
              style={{ width: 44, height: 40, border: "none", cursor: "pointer", borderRadius: 6, background: "none" }}
            />
            <span style={{ fontSize: 14, fontFamily: "monospace" }}>{draft.accentColor}</span>
            {/* Превью цвета */}
            <div style={{
              flex: 1,
              height: 36,
              borderRadius: 6,
              background: `linear-gradient(135deg, ${draft.accentColor}, ${draft.accentColor}88)`,
              transition: "background 0.3s",
            }} />
          </div>
        </div>

        {/* Анимированные фоны */}
        <div>
          <div style={labelStyle}>Анимированный фон</div>
          <div style={{ display: "grid", gridTemplateColumns: "repeat(3, 1fr)", gap: 8 }}>
            {ANIMATED_PRESETS.map((preset) => {
              const isSelected = selectedPreset === preset.id;
              const bgColors = preset.colors.length > 0
                ? preset.colors
                : [draft.accentColor + "44", draft.accentColor + "22", "#36393f", draft.accentColor + "11"];

              return (
                <div
                  key={preset.id}
                  onClick={() => {
                    setSelectedPreset(isSelected ? null : preset.id);
                    setApplied(false);
                  }}
                  style={{
                    height: 60,
                    borderRadius: 8,
                    cursor: "pointer",
                    border: `2px solid ${isSelected ? "var(--accent, #7289da)" : "transparent"}`,
                    background: `linear-gradient(135deg, ${bgColors.join(", ")})`,
                    backgroundSize: "300% 300%",
                    animation: `gradientShift 4s ease infinite`,
                    display: "flex",
                    alignItems: "flex-end",
                    padding: "6px 8px",
                    transition: "border-color 0.15s, transform 0.1s",
                    transform: isSelected ? "scale(1.03)" : "scale(1)",
                    boxShadow: isSelected ? `0 0 12px ${draft.accentColor}66` : "none",
                  }}
                >
                  <span style={{
                    fontSize: 11,
                    fontWeight: 700,
                    color: "#fff",
                    textShadow: "0 1px 3px rgba(0,0,0,0.8)",
                  }}>
                    {preset.label}
                  </span>
                </div>
              );
            })}
            {/* Без анимации */}
            <div
              onClick={() => { setSelectedPreset(null); setApplied(false); }}
              style={{
                height: 60,
                borderRadius: 8,
                cursor: "pointer",
                border: `2px solid ${selectedPreset === null ? "var(--accent, #7289da)" : "transparent"}`,
                background: "var(--bg-tertiary, #202225)",
                display: "flex",
                alignItems: "center",
                justifyContent: "center",
                transition: "border-color 0.15s",
              }}
            >
              <span style={{ fontSize: 11, fontWeight: 700, color: "var(--text-muted, #72767d)" }}>
                Без анимации
              </span>
            </div>
          </div>
        </div>

        {/* Фон чата */}
        <div>
          <div style={labelStyle}>Фоновое изображение чата</div>
          <div style={row}>
            <input
              type="text"
              placeholder="URL изображения..."
              value={draft.chatBackground ?? ""}
              onChange={(e) => {
                updateDraft({ chatBackground: e.target.value || null });
                setSelectedPreset(null);
              }}
              style={inputText}
            />
            <button
              style={{
                padding: "8px 12px",
                borderRadius: 6,
                border: "none",
                cursor: "pointer",
                background: "var(--bg-tertiary, #202225)",
                color: "var(--text-primary, #dcddde)",
                fontSize: 16,
              }}
              onClick={() => fileRef.current?.click()}
              title="Загрузить файл"
            >
              📁
            </button>
            {draft.chatBackground && (
              <button
                style={{
                  padding: "8px 12px",
                  borderRadius: 6,
                  border: "none",
                  cursor: "pointer",
                  background: "#f04747",
                  color: "#fff",
                  fontSize: 14,
                }}
                onClick={() => updateDraft({ chatBackground: null })}
                title="Убрать фон"
              >
                ✕
              </button>
            )}
            <input ref={fileRef} type="file" accept="image/*" style={{ display: "none" }} onChange={handleBgFile} />
          </div>
          {/* Превью фона */}
          {draft.chatBackground && (
            <div style={{
              marginTop: 8,
              height: 80,
              borderRadius: 8,
              backgroundImage: `url(${draft.chatBackground})`,
              backgroundSize: "cover",
              backgroundPosition: "center",
              border: "1px solid var(--bg-tertiary, #202225)",
            }} />
          )}
        </div>

        {/* Кнопки */}
        <div style={{ display: "flex", gap: 8, justifyContent: "space-between", alignItems: "center" }}>
          <button
            style={{
              padding: "8px 14px",
              borderRadius: 6,
              border: "none",
              cursor: "pointer",
              background: "var(--bg-tertiary, #202225)",
              color: "var(--text-muted, #72767d)",
              fontWeight: 600,
              fontSize: 13,
            }}
            onClick={handleReset}
          >
            Сбросить
          </button>

          <div style={{ display: "flex", gap: 8 }}>
            <button
              style={{
                padding: "10px 20px",
                borderRadius: 6,
                border: "none",
                cursor: "pointer",
                background: applied ? "#43b581" : "var(--accent, #7289da)",
                color: "#fff",
                fontWeight: 700,
                fontSize: 14,
                transition: "background 0.3s, transform 0.1s",
                boxShadow: applied ? "0 0 12px #43b58166" : `0 0 12px ${draft.accentColor}44`,
              }}
              onClick={handleApply}
            >
              {applied ? "✓ Применено" : "Применить"}
            </button>
            <button
              style={{
                padding: "10px 16px",
                borderRadius: 6,
                border: "none",
                cursor: "pointer",
                background: "var(--bg-tertiary, #202225)",
                color: "var(--text-primary, #dcddde)",
                fontWeight: 600,
                fontSize: 13,
              }}
              onClick={onClose}
            >
              Закрыть
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}
