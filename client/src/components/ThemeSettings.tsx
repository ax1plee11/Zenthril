import React, { useRef, useState } from "react";
import { useTheme, DEFAULT_THEME, ANIMATED_BG_PRESETS, DEFAULT_TOPBAR_ITEMS } from "../store/theme";
import type { Theme, TopbarItem, TopbarItemId } from "../store/theme";

interface Props { onClose: () => void; }

type Tab = "scheme" | "background" | "accent" | "topbar";

const BG_PRESETS = [
  { id: "none",    label: "None",    preview: "var(--bg-base)" },
  { id: "aurora",  label: "Aurora",  preview: "linear-gradient(135deg,#0d1b2a,#1b4332,#0d1b2a,#1a472a)" },
  { id: "galaxy",  label: "Galaxy",  preview: "linear-gradient(135deg,#1a1a2e,#16213e,#0f3460,#533483)" },
  { id: "sunset",  label: "Sunset",  preview: "linear-gradient(135deg,#2d1b69,#fc5c7d,#ff9a44,#fc5c7d)" },
  { id: "ocean",   label: "Ocean",   preview: "linear-gradient(135deg,#0f2027,#203a43,#2c5364,#1a6b8a)" },
  { id: "fire",    label: "Fire",    preview: "linear-gradient(135deg,#1a0000,#7f0000,#cc2200,#ff6600)" },
  { id: "forest",  label: "Forest",  preview: "linear-gradient(135deg,#0a2e0a,#1a5c1a,#0d3b0d,#2d7a2d)" },
  { id: "neon",    label: "Neon",    preview: "linear-gradient(135deg,#0d0221,#7c6af7,#0d0221,#f72585)" },
  { id: "custom",  label: "Accent",  preview: "" },
];

const CHAT_BG_PRESETS = [
  { id: "none",     label: "Default",  style: "var(--bg-base)" },
  { id: "dots",     label: "Dots",     style: "radial-gradient(circle, rgba(255,255,255,0.04) 1px, transparent 1px)", size: "24px 24px" },
  { id: "grid",     label: "Grid",     style: "linear-gradient(rgba(255,255,255,0.03) 1px, transparent 1px), linear-gradient(90deg, rgba(255,255,255,0.03) 1px, transparent 1px)", size: "32px 32px" },
  { id: "diagonal", label: "Lines",    style: "repeating-linear-gradient(45deg, rgba(255,255,255,0.02) 0px, rgba(255,255,255,0.02) 1px, transparent 1px, transparent 12px)" },
  { id: "noise",    label: "Subtle",   style: "var(--bg-surface)" },
];

export default function ThemeSettings({ onClose }: Props) {
  const { theme, setTheme } = useTheme();
  const fileRef = useRef<HTMLInputElement>(null);
  const [draft, setDraft]   = useState<Theme>({ ...theme });
  const [tab, setTab]       = useState<Tab>("background");
  const [applied, setApplied] = useState(false);
  const [urlInput, setUrlInput] = useState(
    draft.chatBackground && !draft.chatBackground.startsWith("data:") ? draft.chatBackground : ""
  );

  function update(patch: Partial<Theme>) { setDraft(p => ({ ...p, ...patch })); setApplied(false); }

  function handleApply() { setTheme(draft); setApplied(true); }
  function handleReset()  { const r = { ...DEFAULT_THEME }; setDraft(r); setTheme(r); setApplied(false); setUrlInput(""); }

  function handleFile(e: React.ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0];
    if (!file) return;
    if (file.size > 20 * 1024 * 1024) { alert("Max 20 MB"); return; }
    // Предупреждаем что base64 не сохранится после перезагрузки
    const reader = new FileReader();
    reader.onload = () => {
      update({ chatBackground: reader.result as string, animatedPresetId: null, animatedColors: null });
    };
    reader.readAsDataURL(file);
  }

  function handleUrlApply() {
    if (urlInput.trim()) {
      // При установке фона чата — отключаем App Background анимацию
      update({ chatBackground: urlInput.trim(), animatedPresetId: null, animatedColors: null });
    } else {
      update({ chatBackground: null });
    }
  }

  const accentPresets = ["#7c6af7","#3ecf8e","#f5a623","#f04f5e","#06b6d4","#ec4899","#8b5cf6","#14b8a6","#f97316","#ef4444"];

  return (
    <div style={{
      position: "fixed", inset: 0, background: "rgba(0,0,0,0.8)",
      display: "flex", alignItems: "center", justifyContent: "center",
      zIndex: 1000, backdropFilter: "blur(6px)",
    }} onClick={onClose}>
      <div style={{
        background: "var(--bg-surface)", borderRadius: 20,
        width: 520, maxHeight: "88vh", overflow: "hidden",
        border: "1px solid var(--border)", boxShadow: "var(--shadow-lg)",
        display: "flex", flexDirection: "column",
        animation: "fadeUp 0.2s ease",
      }} onClick={e => e.stopPropagation()}>

        {/* Header */}
        <div style={{
          padding: "20px 24px 0", borderBottom: "1px solid var(--border)",
          display: "flex", flexDirection: "column", gap: 16,
        }}>
          <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between" }}>
            <div style={{ fontSize: 17, fontWeight: 700, color: "var(--text-primary)" }}>
              Appearance
            </div>
            <button onClick={onClose} style={{
              background: "none", border: "none", cursor: "pointer",
              color: "var(--text-muted)", fontSize: 18, padding: "2px 6px", borderRadius: 6,
            }}>✕</button>
          </div>

          {/* Tabs */}
          <div style={{ display: "flex", gap: 0 }}>
            {([
              { id: "background", label: "Background" },
              { id: "accent",     label: "Accent Color" },
              { id: "scheme",     label: "Color Scheme" },
              { id: "topbar",     label: "Топбар" },
            ] as { id: Tab; label: string }[]).map(t => (
              <button key={t.id} onClick={() => setTab(t.id)} style={{
                padding: "8px 16px", background: "none", border: "none",
                cursor: "pointer", fontSize: 13, fontWeight: 600,
                color: tab === t.id ? "var(--text-primary)" : "var(--text-muted)",
                borderBottom: `2px solid ${tab === t.id ? "var(--accent)" : "transparent"}`,
                transition: "all 0.15s", marginBottom: -1,
              }}>
                {t.label}
              </button>
            ))}
          </div>
        </div>

        {/* Content */}
        <div style={{ flex: 1, overflowY: "auto", padding: "20px 24px" }}>

          {/* ── BACKGROUND TAB ── */}
          {tab === "background" && (
            <div style={{ display: "flex", flexDirection: "column", gap: 20 }}>

              {/* App background presets */}
              <div>
                <div style={labelS}>App Background</div>
                <div style={{ display: "grid", gridTemplateColumns: "repeat(4, 1fr)", gap: 8 }}>
                  {BG_PRESETS.map(p => {
                    const isSelected = draft.animatedPresetId === p.id || (p.id === "none" && !draft.animatedPresetId);
                    const bg = p.id === "custom"
                      ? `linear-gradient(135deg, ${draft.accentColor}44, ${draft.accentColor}22, var(--bg-base), ${draft.accentColor}11)`
                      : p.preview;
                    return (
                      <div key={p.id} onClick={() => {
                        if (p.id === "none") update({ animatedPresetId: null, animatedColors: null });
                        else {
                          const preset = ANIMATED_BG_PRESETS.find(x => x.id === p.id);
                          update({
                            animatedPresetId: p.id,
                            animatedColors: preset?.colors.length === 4
                              ? [...preset.colors] as [string,string,string,string]
                              : null,
                          });
                        }
                      }} style={{
                        height: 56, borderRadius: 10, cursor: "pointer",
                        background: bg, backgroundSize: "200% 200%",
                        border: `2px solid ${isSelected ? "var(--accent)" : "var(--border)"}`,
                        display: "flex", alignItems: "flex-end", padding: "5px 7px",
                        transition: "all 0.15s",
                        boxShadow: isSelected ? "0 0 0 3px rgba(124,106,247,0.2)" : "none",
                        transform: isSelected ? "scale(1.03)" : "scale(1)",
                      }}>
                        <span style={{ fontSize: 10, fontWeight: 700, color: "#fff", textShadow: "0 1px 3px rgba(0,0,0,0.9)" }}>
                          {p.label}
                        </span>
                      </div>
                    );
                  })}
                </div>
              </div>

              {/* Chat area background */}
              <div>
                <div style={labelS}>Chat Area Background</div>
                <div style={{ display: "grid", gridTemplateColumns: "repeat(5, 1fr)", gap: 6, marginBottom: 12 }}>
                  {CHAT_BG_PRESETS.map(p => {
                    const isNone = p.id === "none";
                    const isSelected = isNone
                      ? !draft.chatBackground
                      : draft.chatBackground === `__pattern__${p.id}`;
                    return (
                      <div key={p.id} onClick={() => {
                        if (isNone) { update({ chatBackground: null }); setUrlInput(""); }
                        else update({ chatBackground: `__pattern__${p.id}`, animatedPresetId: null, animatedColors: null });
                      }} style={{
                        height: 44, borderRadius: 8, cursor: "pointer",
                        background: p.style, backgroundSize: p.size,
                        border: `2px solid ${isSelected ? "var(--accent)" : "var(--border)"}`,
                        display: "flex", alignItems: "center", justifyContent: "center",
                        transition: "all 0.15s",
                        boxShadow: isSelected ? "0 0 0 3px rgba(124,106,247,0.2)" : "none",
                      }}>
                        <span style={{ fontSize: 9, fontWeight: 700, color: "var(--text-muted)", textAlign: "center" as const }}>
                          {p.label}
                        </span>
                      </div>
                    );
                  })}
                </div>

                {/* URL input */}
                <div style={{ display: "flex", gap: 8, marginBottom: 8 }}>
                  <input
                    type="text"
                    placeholder="Image or GIF URL..."
                    value={urlInput}
                    onChange={e => setUrlInput(e.target.value)}
                    onKeyDown={e => e.key === "Enter" && handleUrlApply()}
                    style={{
                      flex: 1, padding: "9px 12px",
                      background: "var(--bg-input)", border: "1px solid var(--border)",
                      borderRadius: "var(--radius-sm)", color: "var(--text-primary)",
                      fontSize: 13, outline: "none", fontFamily: "inherit",
                    }}
                  />
                  <button onClick={handleUrlApply} style={{
                    padding: "9px 14px", background: "var(--accent-dim)",
                    border: "1px solid var(--border-accent)", borderRadius: "var(--radius-sm)",
                    color: "var(--accent)", cursor: "pointer", fontSize: 12, fontWeight: 600,
                  }}>Apply</button>
                  <button onClick={() => fileRef.current?.click()} style={{
                    padding: "9px 12px", background: "var(--bg-elevated)",
                    border: "1px solid var(--border)", borderRadius: "var(--radius-sm)",
                    color: "var(--text-secondary)", cursor: "pointer", fontSize: 13,
                  }} title="Upload image or GIF">
                    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                      <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/>
                      <polyline points="17 8 12 3 7 8"/><line x1="12" y1="3" x2="12" y2="15"/>
                    </svg>
                  </button>
                  <input ref={fileRef} type="file" accept="image/*,.gif" style={{ display: "none" }} onChange={handleFile} />
                </div>

                {/* Preview */}
                {draft.chatBackground && !draft.chatBackground.startsWith("__pattern__") && (
                  <div style={{ position: "relative" }}>
                    <div style={{
                      height: 100, borderRadius: 10, overflow: "hidden",
                      border: "1px solid var(--border)",
                      backgroundImage: `url(${draft.chatBackground})`,
                      backgroundSize: "cover", backgroundPosition: "center",
                    }} />
                    {draft.chatBackground.startsWith("data:") && (
                      <div style={{
                        fontSize: 10, color: "var(--yellow, #f5a623)",
                        marginTop: 4, display: "flex", alignItems: "center", gap: 4,
                      }}>
                        ⚠️ Uploaded files are not saved after page reload. Use a URL instead.
                      </div>
                    )}
                    <button onClick={() => { update({ chatBackground: null }); setUrlInput(""); }} style={{
                      position: "absolute", top: 6, right: 6,
                      background: "rgba(0,0,0,0.7)", border: "none", borderRadius: 6,
                      color: "#fff", cursor: "pointer", padding: "3px 8px", fontSize: 11,
                    }}>Remove</button>
                  </div>
                )}
              </div>

              {/* Overlay Darkness — применяется ко всем панелям когда активен фон */}
              {(draft.animatedPresetId || (draft.chatBackground && !draft.chatBackground.startsWith("__pattern__"))) && (
                <div>
                  <div style={labelS}>
                    Overlay Darkness
                    <span style={{ float: "right" as const, color: "var(--text-secondary)", fontWeight: 400 }}>
                      {draft.overlayDarkness ?? 40}%
                    </span>
                  </div>
                  <input
                    type="range" min={0} max={90}
                    value={draft.overlayDarkness ?? 40}
                    onChange={e => update({ overlayDarkness: Number(e.target.value) })}
                    style={{ width: "100%", accentColor: "var(--accent)" }}
                  />
                  <div style={{ display: "flex", justifyContent: "space-between", fontSize: 10, color: "var(--text-muted)", marginTop: 4 }}>
                    <span>Transparent — все панели прозрачные</span>
                    <span>Dark — непрозрачные</span>
                  </div>
                </div>
              )}
            </div>
          )}

          {/* ── ACCENT TAB ── */}
          {tab === "accent" && (
            <div style={{ display: "flex", flexDirection: "column", gap: 20 }}>
              <div>
                <div style={labelS}>Preset Colors</div>
                <div style={{ display: "flex", flexWrap: "wrap" as const, gap: 8 }}>
                  {accentPresets.map(c => (
                    <div key={c} onClick={() => update({ accentColor: c })} style={{
                      width: 36, height: 36, borderRadius: 10, cursor: "pointer",
                      background: c,
                      border: `3px solid ${draft.accentColor === c ? "#fff" : "transparent"}`,
                      boxShadow: draft.accentColor === c ? `0 0 0 2px ${c}` : "none",
                      transition: "all 0.15s",
                      transform: draft.accentColor === c ? "scale(1.15)" : "scale(1)",
                    }} />
                  ))}
                </div>
              </div>
              <div>
                <div style={labelS}>Custom Color</div>
                <div style={{ display: "flex", alignItems: "center", gap: 12 }}>
                  <input type="color" value={draft.accentColor}
                    onChange={e => update({ accentColor: e.target.value })}
                    style={{ width: 52, height: 44, border: "none", cursor: "pointer", borderRadius: 10, background: "none" }}
                  />
                  <div style={{
                    flex: 1, height: 44, borderRadius: 10,
                    background: `linear-gradient(135deg, ${draft.accentColor}, ${draft.accentColor}88)`,
                    display: "flex", alignItems: "center", justifyContent: "center",
                    fontSize: 13, fontWeight: 600, color: "#fff",
                    boxShadow: `0 4px 16px ${draft.accentColor}44`,
                  }}>
                    {draft.accentColor}
                  </div>
                </div>
              </div>
              {/* Preview */}
              <div style={{
                padding: 16, borderRadius: 12,
                background: "var(--bg-elevated)", border: "1px solid var(--border)",
              }}>
                <div style={{ fontSize: 11, color: "var(--text-muted)", marginBottom: 10, fontWeight: 600, letterSpacing: 0.5, textTransform: "uppercase" as const }}>Preview</div>
                <div style={{ display: "flex", gap: 8, flexWrap: "wrap" as const }}>
                  <button style={{
                    padding: "8px 16px", borderRadius: 8, border: "none",
                    background: `linear-gradient(135deg, ${draft.accentColor}, ${draft.accentColor}cc)`,
                    color: "#fff", fontWeight: 600, fontSize: 13, cursor: "default",
                    boxShadow: `0 4px 12px ${draft.accentColor}44`,
                  }}>Button</button>
                  <div style={{
                    padding: "8px 12px", borderRadius: 8,
                    background: `${draft.accentColor}22`,
                    border: `1px solid ${draft.accentColor}44`,
                    color: draft.accentColor, fontSize: 13, fontWeight: 600,
                  }}>Badge</div>
                  <div style={{
                    width: 10, height: 10, borderRadius: "50%",
                    background: draft.accentColor, alignSelf: "center",
                    boxShadow: `0 0 8px ${draft.accentColor}`,
                  }} />
                </div>
              </div>
            </div>
          )}

          {/* ── SCHEME TAB ── */}
          {tab === "scheme" && (
            <div style={{ display: "flex", flexDirection: "column", gap: 16 }}>
              {([
                { key: "dark",   label: "Dark",   desc: "Easy on the eyes",       icon: "🌙" },
                { key: "light",  label: "Light",  desc: "Bright and clean",        icon: "☀️" },
                { key: "custom", label: "Custom", desc: "Your accent color rules", icon: "🎨" },
              ] as { key: Theme["scheme"]; label: string; desc: string; icon: string }[]).map(s => (
                <div key={s.key} onClick={() => update({ scheme: s.key })} style={{
                  padding: "14px 16px", borderRadius: 12, cursor: "pointer",
                  border: `2px solid ${draft.scheme === s.key ? "var(--accent)" : "var(--border)"}`,
                  background: draft.scheme === s.key ? "var(--accent-dim)" : "var(--bg-elevated)",
                  display: "flex", alignItems: "center", gap: 12, transition: "all 0.15s",
                }}>
                  <span style={{ fontSize: 24 }}>{s.icon}</span>
                  <div>
                    <div style={{ fontWeight: 600, fontSize: 14, color: "var(--text-primary)" }}>{s.label}</div>
                    <div style={{ fontSize: 12, color: "var(--text-muted)" }}>{s.desc}</div>
                  </div>
                  {draft.scheme === s.key && (
                    <div style={{ marginLeft: "auto", color: "var(--accent)" }}>
                      <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5">
                        <polyline points="20 6 9 17 4 12"/>
                      </svg>
                    </div>
                  )}
                </div>
              ))}
            </div>
          )}

          {/* ── TOPBAR TAB ── */}
          {tab === "topbar" && (
            <TopbarCustomizer
              items={draft.topbarItems ?? [...DEFAULT_TOPBAR_ITEMS]}
              onChange={(items) => update({ topbarItems: items })}
            />
          )}
        </div>

        {/* Footer */}
        <div style={{
          padding: "14px 24px", borderTop: "1px solid var(--border)",
          display: "flex", justifyContent: "space-between", alignItems: "center",
          background: "var(--bg-base)",
        }}>
          <button onClick={handleReset} style={{
            padding: "8px 14px", background: "none",
            border: "1px solid var(--border)", borderRadius: "var(--radius-sm)",
            color: "var(--text-muted)", cursor: "pointer", fontSize: 12, fontWeight: 600,
          }}>Reset to Default</button>
          <div style={{ display: "flex", gap: 8 }}>
            <button onClick={onClose} style={{
              padding: "9px 16px", background: "var(--bg-elevated)",
              border: "1px solid var(--border)", borderRadius: "var(--radius-sm)",
              color: "var(--text-secondary)", cursor: "pointer", fontSize: 13, fontWeight: 600,
            }}>Cancel</button>
            <button onClick={handleApply} style={{
              padding: "9px 20px",
              background: applied ? "rgba(62,207,142,0.2)" : "linear-gradient(135deg, #7c6af7, #a78bfa)",
              border: applied ? "1px solid rgba(62,207,142,0.4)" : "none",
              borderRadius: "var(--radius-sm)", color: applied ? "#3ecf8e" : "#fff",
              cursor: "pointer", fontSize: 13, fontWeight: 700,
              boxShadow: applied ? "none" : "0 4px 12px rgba(124,106,247,0.4)",
              transition: "all 0.2s",
            }}>
              {applied ? "✓ Applied" : "Apply"}
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}

const labelS: React.CSSProperties = {
  fontSize: 11, fontWeight: 700, color: "var(--text-muted)",
  textTransform: "uppercase", letterSpacing: 0.8, marginBottom: 10,
};

// ── Topbar Customizer ─────────────────────────────────────────────────────────

const TOPBAR_ITEM_META: Record<TopbarItemId, { label: string; icon: React.ReactNode; desc: string }> = {
  search: {
    label: "Поиск",
    desc: "Поиск пользователей",
    icon: <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><circle cx="11" cy="11" r="8"/><line x1="21" y1="21" x2="16.65" y2="16.65"/></svg>,
  },
  notifications: {
    label: "Уведомления",
    desc: "Колокольчик с бейджем",
    icon: <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><path d="M18 8A6 6 0 0 0 6 8c0 7-3 9-3 9h18s-3-2-3-9"/><path d="M13.73 21a2 2 0 0 1-3.46 0"/></svg>,
  },
  friends: {
    label: "Друзья",
    desc: "Список друзей и запросы",
    icon: <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/><path d="M23 21v-2a4 4 0 0 0-3-3.87"/><path d="M16 3.13a4 4 0 0 1 0 7.75"/></svg>,
  },
  settings: {
    label: "Настройки",
    desc: "Оформление и тема",
    icon: <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><circle cx="12" cy="12" r="3"/><path d="M19.07 4.93a10 10 0 0 1 0 14.14M4.93 4.93a10 10 0 0 0 0 14.14"/></svg>,
  },
  divider: {
    label: "Разделитель",
    desc: "Вертикальная линия",
    icon: <div style={{ width: 2, height: 14, background: "var(--border)", borderRadius: 1 }} />,
  },
};

function TopbarCustomizer({ items, onChange }: {
  items: TopbarItem[];
  onChange: (items: TopbarItem[]) => void;
}) {
  function toggle(id: TopbarItemId) {
    onChange(items.map(it => it.id === id ? { ...it, visible: !it.visible } : it));
  }

  function moveUp(idx: number) {
    if (idx === 0) return;
    const next = [...items];
    [next[idx - 1], next[idx]] = [next[idx], next[idx - 1]];
    onChange(next);
  }

  function moveDown(idx: number) {
    if (idx === items.length - 1) return;
    const next = [...items];
    [next[idx], next[idx + 1]] = [next[idx + 1], next[idx]];
    onChange(next);
  }

  function reset() {
    onChange([...DEFAULT_TOPBAR_ITEMS]);
  }

  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 16 }}>
      <div style={{ fontSize: 12, color: "var(--text-muted)", lineHeight: 1.6 }}>
        Настрой какие кнопки показывать в топбаре и в каком порядке.
        Прозрачность панелей настраивается через вкладку Background → Overlay Darkness.
      </div>

      {/* Preview */}
      <div>
        <div style={labelS}>Предпросмотр</div>
        <div style={{
          display: "flex", alignItems: "center", gap: 4,
          padding: "8px 12px", borderRadius: 10,
          background: "var(--bg-elevated)", border: "1px solid var(--border)",
          height: 40,
        }}>
          <div style={{ flex: 1 }} />
          {items.filter(it => it.visible).map(it => (
            it.id === "divider"
              ? <div key="div" style={{ width: 1, height: 18, background: "var(--border)", margin: "0 2px" }} />
              : (
                <div key={it.id} style={{
                  width: 28, height: 28, borderRadius: 6,
                  background: "rgba(255,255,255,0.06)",
                  display: "flex", alignItems: "center", justifyContent: "center",
                  color: "var(--text-muted)",
                }}>
                  {TOPBAR_ITEM_META[it.id].icon}
                </div>
              )
          ))}
          {/* Profile placeholder */}
          <div style={{
            display: "flex", alignItems: "center", gap: 6,
            padding: "3px 8px 3px 3px", borderRadius: 6,
            background: "rgba(255,255,255,0.06)", marginLeft: 2,
          }}>
            <div style={{ width: 22, height: 22, borderRadius: 6, background: "var(--accent)", opacity: 0.7 }} />
            <div style={{ width: 40, height: 8, borderRadius: 4, background: "var(--border)" }} />
          </div>
        </div>
      </div>

      {/* Items list */}
      <div>
        <div style={labelS}>Элементы</div>
        <div style={{ display: "flex", flexDirection: "column", gap: 6 }}>
          {items.map((item, idx) => {
            const meta = TOPBAR_ITEM_META[item.id];
            return (
              <div key={item.id} style={{
                display: "flex", alignItems: "center", gap: 10,
                padding: "10px 12px", borderRadius: 10,
                background: item.visible ? "var(--bg-elevated)" : "rgba(255,255,255,0.02)",
                border: `1px solid ${item.visible ? "var(--border)" : "rgba(255,255,255,0.04)"}`,
                opacity: item.visible ? 1 : 0.5,
                transition: "all 0.15s",
              }}>
                {/* Icon */}
                <div style={{
                  width: 32, height: 32, borderRadius: 8, flexShrink: 0,
                  background: item.visible ? "var(--accent-dim)" : "rgba(255,255,255,0.04)",
                  display: "flex", alignItems: "center", justifyContent: "center",
                  color: item.visible ? "var(--accent)" : "var(--text-muted)",
                }}>
                  {meta.icon}
                </div>

                {/* Label */}
                <div style={{ flex: 1, minWidth: 0 }}>
                  <div style={{ fontSize: 13, fontWeight: 600, color: "var(--text-primary)" }}>{meta.label}</div>
                  <div style={{ fontSize: 11, color: "var(--text-muted)" }}>{meta.desc}</div>
                </div>

                {/* Move buttons */}
                <div style={{ display: "flex", flexDirection: "column", gap: 2 }}>
                  <button onClick={() => moveUp(idx)} disabled={idx === 0} style={{
                    background: "none", border: "none", cursor: idx === 0 ? "default" : "pointer",
                    color: idx === 0 ? "var(--border)" : "var(--text-muted)",
                    padding: "2px 4px", borderRadius: 4, lineHeight: 1,
                  }}>▲</button>
                  <button onClick={() => moveDown(idx)} disabled={idx === items.length - 1} style={{
                    background: "none", border: "none", cursor: idx === items.length - 1 ? "default" : "pointer",
                    color: idx === items.length - 1 ? "var(--border)" : "var(--text-muted)",
                    padding: "2px 4px", borderRadius: 4, lineHeight: 1,
                  }}>▼</button>
                </div>

                {/* Toggle */}
                <button onClick={() => toggle(item.id)} style={{
                  width: 36, height: 20, borderRadius: 10, border: "none", cursor: "pointer",
                  background: item.visible ? "var(--accent)" : "rgba(255,255,255,0.1)",
                  position: "relative", transition: "background 0.2s", flexShrink: 0,
                }}>
                  <div style={{
                    position: "absolute", top: 2,
                    left: item.visible ? 18 : 2,
                    width: 16, height: 16, borderRadius: "50%",
                    background: "#fff", transition: "left 0.2s",
                    boxShadow: "0 1px 3px rgba(0,0,0,0.3)",
                  }} />
                </button>
              </div>
            );
          })}
        </div>
      </div>

      <button onClick={reset} style={{
        padding: "8px", borderRadius: 8,
        background: "none", border: "1px solid var(--border)",
        color: "var(--text-muted)", cursor: "pointer", fontSize: 12, fontWeight: 600,
      }}>Сбросить по умолчанию</button>
    </div>
  );
}
