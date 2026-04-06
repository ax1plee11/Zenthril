/**
 * ProfileModal — модальное окно личного профиля
 */
import React, { useState, useRef } from "react";
import { useAuth } from "../store/auth";

const PROFILE_KEY = "vibrora_profile";

export type UserStatus = "online" | "dnd" | "offline";

export interface ProfileTheme {
  bannerPresetId: string | null;
  bannerColor: string;
  bannerUrl: string | null;
}

export interface UserProfile {
  avatarBase64: string | null;
  status: UserStatus;
  bio: string;
  theme: ProfileTheme;
}

const DEFAULT_PROFILE_THEME: ProfileTheme = {
  bannerPresetId: null,
  bannerColor: "#7c6af7",
  bannerUrl: null,
};

export const BANNER_PRESETS = [
  { id: "none",   label: "Нет",    colors: ["#1a1b27","#1a1b27","#1a1b27","#1a1b27"] },
  { id: "aurora", label: "Aurora", colors: ["#0d1b2a","#1b4332","#0d1b2a","#1a472a"] },
  { id: "galaxy", label: "Galaxy", colors: ["#1a1a2e","#16213e","#0f3460","#533483"] },
  { id: "sunset", label: "Sunset", colors: ["#2d1b69","#fc5c7d","#ff9a44","#fc5c7d"] },
  { id: "ocean",  label: "Ocean",  colors: ["#0f2027","#203a43","#2c5364","#1a6b8a"] },
  { id: "fire",   label: "Fire",   colors: ["#1a0000","#7f0000","#cc2200","#ff6600"] },
  { id: "neon",   label: "Neon",   colors: ["#0d0221","#7c6af7","#0d0221","#f72585"] },
];

export function getBannerStyle(theme: ProfileTheme): React.CSSProperties {
  if (theme.bannerUrl) {
    return { backgroundImage: `url(${theme.bannerUrl})`, backgroundSize: "cover", backgroundPosition: "center" };
  }
  if (theme.bannerPresetId) {
    const preset = BANNER_PRESETS.find(p => p.id === theme.bannerPresetId);
    if (preset) return { background: `linear-gradient(135deg, ${preset.colors.join(", ")})` };
  }
  return { background: `linear-gradient(135deg, ${theme.bannerColor}44, ${theme.bannerColor}22, #0d0e14)` };
}

export function loadProfile(): UserProfile {
  try {
    const raw = localStorage.getItem(PROFILE_KEY);
    if (raw) {
      const p = JSON.parse(raw) as UserProfile;
      return { ...p, theme: { ...DEFAULT_PROFILE_THEME, ...(p.theme ?? {}) } };
    }
  } catch { /* ignore */ }
  return { avatarBase64: null, status: "online", bio: "", theme: { ...DEFAULT_PROFILE_THEME } };
}

export function saveProfile(p: UserProfile, userId?: string): void {
  localStorage.setItem(PROFILE_KEY, JSON.stringify(p));
  if (userId) localStorage.setItem(`vibrora_profile_${userId}`, JSON.stringify(p));
}

// ─────────────────────────────────────────────────────────────────────────────

interface Props { onClose: () => void; }
type Tab = "profile" | "appearance";

const STATUS_COLORS: Record<UserStatus, string> = { online: "#43b581", dnd: "#f04747", offline: "#747f8d" };
const STATUS_LABELS: Record<UserStatus, string> = { online: "Онлайн", dnd: "Не беспокоить", offline: "Офлайн" };

const labelS: React.CSSProperties = {
  fontSize: 11, fontWeight: 700, color: "var(--text-muted)",
  textTransform: "uppercase", letterSpacing: 0.8, marginBottom: 10,
};

export default function ProfileModal({ onClose }: Props) {
  const { user } = useAuth();
  const [profile, setProfile] = useState<UserProfile>(loadProfile);
  const [tab, setTab] = useState<Tab>("profile");
  const [bannerUrlInput, setBannerUrlInput] = useState(profile.theme.bannerUrl ?? "");
  const fileRef = useRef<HTMLInputElement>(null);

  function update(patch: Partial<UserProfile>) { setProfile(prev => ({ ...prev, ...patch })); }
  function updateTheme(patch: Partial<ProfileTheme>) {
    setProfile(prev => ({ ...prev, theme: { ...prev.theme, ...patch } }));
  }

  function handleAvatarFile(e: React.ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0];
    if (!file) return;
    if (file.size > 5 * 1024 * 1024) { alert("Файл слишком большой (макс. 5 МБ)"); return; }
    const reader = new FileReader();
    reader.onload = () => update({ avatarBase64: reader.result as string });
    reader.readAsDataURL(file);
  }

  function handleSave() { saveProfile(profile, user?.id); onClose(); }

  const initials = (user?.username ?? "?").charAt(0).toUpperCase();
  const bannerStyle = getBannerStyle(profile.theme);

  return (
    <div style={{
      position: "fixed", inset: 0, background: "rgba(0,0,0,0.75)",
      display: "flex", alignItems: "center", justifyContent: "center",
      zIndex: 1000, backdropFilter: "blur(6px)",
    }} onClick={onClose}>
      <div style={{
        background: "var(--bg-surface)", borderRadius: 20,
        width: 440, maxHeight: "90vh", overflow: "hidden",
        border: "1px solid var(--border)", boxShadow: "var(--shadow-lg)",
        display: "flex", flexDirection: "column",
        animation: "fadeUp 0.2s ease",
      }} onClick={e => e.stopPropagation()}>

        {/* Header */}
        <div style={{ padding: "20px 24px 0", borderBottom: "1px solid var(--border)", display: "flex", flexDirection: "column", gap: 16 }}>
          <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between" }}>
            <div style={{ fontSize: 17, fontWeight: 700, color: "var(--text-primary)" }}>Профиль</div>
            <button onClick={onClose} style={{ background: "none", border: "none", cursor: "pointer", color: "var(--text-muted)", fontSize: 18, padding: "2px 6px", borderRadius: 6 }}>✕</button>
          </div>
          <div style={{ display: "flex" }}>
            {([{ id: "profile", label: "Основное" }, { id: "appearance", label: "Оформление" }] as { id: Tab; label: string }[]).map(t => (
              <button key={t.id} onClick={() => setTab(t.id)} style={{
                padding: "8px 16px", background: "none", border: "none", cursor: "pointer",
                fontSize: 13, fontWeight: 600,
                color: tab === t.id ? "var(--text-primary)" : "var(--text-muted)",
                borderBottom: `2px solid ${tab === t.id ? "var(--accent)" : "transparent"}`,
                marginBottom: -1, transition: "all 0.15s",
              }}>{t.label}</button>
            ))}
          </div>
        </div>

        {/* Content */}
        <div style={{ flex: 1, overflowY: "auto", padding: "20px 24px" }}>

          {tab === "profile" && (
            <div style={{ display: "flex", flexDirection: "column", gap: 20 }}>
              {/* Avatar */}
              <div style={{ display: "flex", alignItems: "center", gap: 16 }}>
                <div style={{
                  width: 72, height: 72, borderRadius: "50%",
                  background: `linear-gradient(135deg, ${profile.theme.bannerColor}, ${profile.theme.bannerColor}99)`,
                  display: "flex", alignItems: "center", justifyContent: "center",
                  fontSize: 28, fontWeight: 700, color: "#fff",
                  overflow: "hidden", flexShrink: 0, cursor: "pointer",
                  border: `3px solid ${profile.theme.bannerColor}66`,
                  boxShadow: `0 0 20px ${profile.theme.bannerColor}44`,
                }} onClick={() => fileRef.current?.click()}>
                  {profile.avatarBase64
                    ? <img src={profile.avatarBase64} alt="avatar" style={{ width: "100%", height: "100%", objectFit: "cover" }} />
                    : initials}
                </div>
                <div>
                  <div style={{ fontWeight: 700, fontSize: 16, color: "var(--text-primary)" }}>{user?.username}</div>
                  <button style={{
                    marginTop: 8, padding: "6px 14px",
                    background: "var(--bg-elevated)", border: "1px solid var(--border)",
                    borderRadius: 8, color: "var(--text-secondary)", cursor: "pointer", fontSize: 12, fontWeight: 600,
                  }} onClick={() => fileRef.current?.click()}>Загрузить фото</button>
                  <input ref={fileRef} type="file" accept="image/jpeg,image/png,image/webp" style={{ display: "none" }} onChange={handleAvatarFile} />
                </div>
              </div>

              {/* Status */}
              <div>
                <div style={labelS}>Статус</div>
                <div style={{ display: "flex", gap: 8 }}>
                  {(["online", "dnd", "offline"] as UserStatus[]).map(s => (
                    <button key={s} onClick={() => update({ status: s })} style={{
                      flex: 1, padding: "10px 4px", borderRadius: 10,
                      border: `2px solid ${profile.status === s ? STATUS_COLORS[s] : "var(--border)"}`,
                      background: profile.status === s ? STATUS_COLORS[s] + "22" : "var(--bg-elevated)",
                      color: profile.status === s ? STATUS_COLORS[s] : "var(--text-muted)",
                      cursor: "pointer", fontWeight: 600, fontSize: 12,
                      display: "flex", alignItems: "center", justifyContent: "center", gap: 6,
                      transition: "all 0.15s",
                    }}>
                      <span style={{ width: 8, height: 8, borderRadius: "50%", background: STATUS_COLORS[s], display: "inline-block", flexShrink: 0 }} />
                      {STATUS_LABELS[s]}
                    </button>
                  ))}
                </div>
              </div>

              {/* Bio */}
              <div>
                <div style={labelS}>О себе</div>
                <textarea maxLength={300} value={profile.bio}
                  onChange={e => update({ bio: e.target.value })}
                  placeholder="Расскажите о себе..."
                  style={{
                    width: "100%", minHeight: 90,
                    background: "var(--bg-elevated)", border: "1px solid var(--border)",
                    borderRadius: 10, padding: "10px 12px",
                    color: "var(--text-primary)", fontSize: 14,
                    resize: "vertical", fontFamily: "inherit", outline: "none",
                  }}
                />
                <div style={{ fontSize: 11, color: "var(--text-muted)", textAlign: "right", marginTop: 4 }}>{profile.bio.length}/300</div>
              </div>
            </div>
          )}

          {tab === "appearance" && (
            <div style={{ display: "flex", flexDirection: "column", gap: 20 }}>

              {/* Preview */}
              <div>
                <div style={labelS}>Предпросмотр</div>
                <div style={{ borderRadius: 16, overflow: "hidden", border: "1px solid var(--border)" }}>
                  <div style={{
                    height: 90, ...bannerStyle, backgroundSize: "400% 400%",
                    animation: profile.theme.bannerPresetId ? "gradientShift 6s ease infinite" : "none",
                    position: "relative",
                  }}>
                    <div style={{
                      position: "absolute", bottom: -24, left: 16,
                      width: 52, height: 52, borderRadius: "50%",
                      background: `linear-gradient(135deg, ${profile.theme.bannerColor}, ${profile.theme.bannerColor}99)`,
                      border: "3px solid var(--bg-surface)",
                      display: "flex", alignItems: "center", justifyContent: "center",
                      fontSize: 20, fontWeight: 700, color: "#fff", overflow: "hidden",
                      boxShadow: `0 0 16px ${profile.theme.bannerColor}66`,
                    }}>
                      {profile.avatarBase64
                        ? <img src={profile.avatarBase64} alt="av" style={{ width: "100%", height: "100%", objectFit: "cover" }} />
                        : initials}
                    </div>
                  </div>
                  <div style={{ padding: "32px 16px 16px", background: "var(--bg-elevated)" }}>
                    <div style={{ fontWeight: 700, fontSize: 15, color: "var(--text-primary)" }}>{user?.username}</div>
                    {profile.bio && <div style={{ fontSize: 12, color: "var(--text-muted)", marginTop: 4 }}>{profile.bio}</div>}
                  </div>
                </div>
              </div>

              {/* Banner presets */}
              <div>
                <div style={labelS}>Баннер профиля</div>
                <div style={{ display: "grid", gridTemplateColumns: "repeat(4, 1fr)", gap: 8 }}>
                  {BANNER_PRESETS.map(p => {
                    const isSelected = p.id === "none"
                      ? !profile.theme.bannerPresetId && !profile.theme.bannerUrl
                      : profile.theme.bannerPresetId === p.id;
                    const bg = p.id === "none" ? "var(--bg-elevated)" : `linear-gradient(135deg, ${p.colors.join(", ")})`;
                    return (
                      <div key={p.id} onClick={() => {
                        if (p.id === "none") updateTheme({ bannerPresetId: null, bannerUrl: null });
                        else updateTheme({ bannerPresetId: p.id, bannerUrl: null });
                        setBannerUrlInput("");
                      }} style={{
                        height: 48, borderRadius: 10, cursor: "pointer",
                        background: bg, backgroundSize: "200% 200%",
                        border: `2px solid ${isSelected ? "var(--accent)" : "var(--border)"}`,
                        display: "flex", alignItems: "flex-end", padding: "4px 6px",
                        transition: "all 0.15s",
                        boxShadow: isSelected ? "0 0 0 3px rgba(124,106,247,0.2)" : "none",
                        transform: isSelected ? "scale(1.04)" : "scale(1)",
                      }}>
                        <span style={{ fontSize: 9, fontWeight: 700, color: "#fff", textShadow: "0 1px 3px rgba(0,0,0,0.9)" }}>{p.label}</span>
                      </div>
                    );
                  })}
                </div>
              </div>

              {/* Banner URL */}
              <div>
                <div style={labelS}>URL баннера (фото / GIF)</div>
                <div style={{ display: "flex", gap: 8 }}>
                  <input type="text" placeholder="https://..." value={bannerUrlInput}
                    onChange={e => setBannerUrlInput(e.target.value)}
                    onKeyDown={e => { if (e.key === "Enter") updateTheme({ bannerUrl: bannerUrlInput.trim() || null, bannerPresetId: null }); }}
                    style={{
                      flex: 1, padding: "9px 12px",
                      background: "var(--bg-input)", border: "1px solid var(--border)",
                      borderRadius: 8, color: "var(--text-primary)", fontSize: 13, outline: "none", fontFamily: "inherit",
                    }}
                  />
                  <button onClick={() => updateTheme({ bannerUrl: bannerUrlInput.trim() || null, bannerPresetId: null })} style={{
                    padding: "9px 14px", background: "var(--accent-dim)",
                    border: "1px solid var(--border-accent)", borderRadius: 8,
                    color: "var(--accent)", cursor: "pointer", fontSize: 12, fontWeight: 600,
                  }}>OK</button>
                </div>
              </div>

              {/* Accent color */}
              <div>
                <div style={labelS}>Цвет акцента профиля</div>
                <div style={{ display: "flex", alignItems: "center", gap: 12 }}>
                  <input type="color" value={profile.theme.bannerColor}
                    onChange={e => updateTheme({ bannerColor: e.target.value })}
                    style={{ width: 48, height: 40, border: "none", cursor: "pointer", borderRadius: 8, background: "none" }}
                  />
                  <div style={{
                    flex: 1, height: 40, borderRadius: 10,
                    background: `linear-gradient(135deg, ${profile.theme.bannerColor}, ${profile.theme.bannerColor}88)`,
                    display: "flex", alignItems: "center", justifyContent: "center",
                    fontSize: 12, fontWeight: 600, color: "#fff",
                    boxShadow: `0 4px 12px ${profile.theme.bannerColor}44`,
                  }}>{profile.theme.bannerColor}</div>
                </div>
              </div>
            </div>
          )}
        </div>

        {/* Footer */}
        <div style={{
          padding: "14px 24px", borderTop: "1px solid var(--border)",
          display: "flex", justifyContent: "flex-end", gap: 8, background: "var(--bg-base)",
        }}>
          <button onClick={onClose} style={{
            padding: "9px 16px", background: "var(--bg-elevated)",
            border: "1px solid var(--border)", borderRadius: 8,
            color: "var(--text-secondary)", cursor: "pointer", fontSize: 13, fontWeight: 600,
          }}>Отмена</button>
          <button onClick={handleSave} style={{
            padding: "9px 20px",
            background: "linear-gradient(135deg, var(--accent), var(--accent-hover))",
            border: "none", borderRadius: 8, color: "#fff",
            cursor: "pointer", fontSize: 13, fontWeight: 700,
            boxShadow: "0 4px 12px rgba(124,106,247,0.4)",
          }}>Сохранить</button>
        </div>
      </div>
    </div>
  );
}
