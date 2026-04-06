/**
 * ProfileModal — модальное окно личного профиля
 */

import React, { useState, useRef } from "react";
import { useAuth } from "../store/auth";

const PROFILE_KEY = "vibrora_profile";

export type UserStatus = "online" | "dnd" | "offline";

export interface UserProfile {
  avatarBase64: string | null;
  status: UserStatus;
  bio: string;
}

export function loadProfile(): UserProfile {
  try {
    const raw = localStorage.getItem(PROFILE_KEY);
    if (raw) return JSON.parse(raw) as UserProfile;
  } catch {
    // ignore
  }
  return { avatarBase64: null, status: "online", bio: "" };
}

export function saveProfile(p: UserProfile): void {
  localStorage.setItem(PROFILE_KEY, JSON.stringify(p));
}

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
  width: 380,
  color: "var(--text-primary, #dcddde)",
  display: "flex",
  flexDirection: "column",
  gap: 20,
};

const labelStyle: React.CSSProperties = {
  fontSize: 12,
  fontWeight: 600,
  textTransform: "uppercase",
  color: "var(--text-muted, #72767d)",
  marginBottom: 8,
};

const STATUS_COLORS: Record<UserStatus, string> = {
  online: "#43b581",
  dnd: "#f04747",
  offline: "#747f8d",
};

const STATUS_LABELS: Record<UserStatus, string> = {
  online: "Онлайн",
  dnd: "Не беспокоить",
  offline: "Офлайн",
};

export default function ProfileModal({ onClose }: Props) {
  const { user } = useAuth();
  const [profile, setProfile] = useState<UserProfile>(loadProfile);
  const fileRef = useRef<HTMLInputElement>(null);

  function update(patch: Partial<UserProfile>) {
    setProfile((prev) => ({ ...prev, ...patch }));
  }

  function handleAvatarFile(e: React.ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0];
    if (!file) return;
    if (file.size > 5 * 1024 * 1024) {
      alert("Файл слишком большой (макс. 5 МБ)");
      return;
    }
    const reader = new FileReader();
    reader.onload = () => update({ avatarBase64: reader.result as string });
    reader.readAsDataURL(file);
  }

  function handleSave() {
    saveProfile(profile);
    onClose();
  }

  const initials = (user?.username ?? "?").charAt(0).toUpperCase();

  return (
    <div style={overlay} onClick={onClose}>
      <div style={modal} onClick={(e) => e.stopPropagation()}>
        <div style={{ fontSize: 18, fontWeight: 700 }}>Профиль</div>

        {/* Аватар */}
        <div style={{ display: "flex", alignItems: "center", gap: 16 }}>
          <div
            style={{
              width: 72,
              height: 72,
              borderRadius: "50%",
              background: "var(--accent, #7289da)",
              display: "flex",
              alignItems: "center",
              justifyContent: "center",
              fontSize: 28,
              fontWeight: 700,
              color: "#fff",
              overflow: "hidden",
              flexShrink: 0,
              cursor: "pointer",
              position: "relative",
            }}
            onClick={() => fileRef.current?.click()}
            title="Загрузить аватар"
          >
            {profile.avatarBase64 ? (
              <img src={profile.avatarBase64} alt="avatar" style={{ width: "100%", height: "100%", objectFit: "cover" }} />
            ) : (
              initials
            )}
          </div>
          <div>
            <div style={{ fontWeight: 600, fontSize: 16 }}>{user?.username}</div>
            <button
              style={{
                marginTop: 8,
                padding: "6px 12px",
                background: "var(--bg-tertiary, #202225)",
                border: "none",
                borderRadius: 4,
                color: "var(--text-primary, #dcddde)",
                cursor: "pointer",
                fontSize: 13,
              }}
              onClick={() => fileRef.current?.click()}
            >
              Загрузить фото
            </button>
            <input
              ref={fileRef}
              type="file"
              accept="image/jpeg,image/png,image/webp"
              style={{ display: "none" }}
              onChange={handleAvatarFile}
            />
          </div>
        </div>

        {/* Статус */}
        <div>
          <div style={labelStyle}>Статус</div>
          <div style={{ display: "flex", gap: 8 }}>
            {(["online", "dnd", "offline"] as UserStatus[]).map((s) => (
              <button
                key={s}
                onClick={() => update({ status: s })}
                style={{
                  flex: 1,
                  padding: "8px 4px",
                  borderRadius: 4,
                  border: `2px solid ${profile.status === s ? STATUS_COLORS[s] : "transparent"}`,
                  background: profile.status === s ? STATUS_COLORS[s] + "22" : "var(--bg-tertiary, #202225)",
                  color: profile.status === s ? STATUS_COLORS[s] : "var(--text-primary, #dcddde)",
                  cursor: "pointer",
                  fontWeight: 600,
                  fontSize: 12,
                  display: "flex",
                  alignItems: "center",
                  justifyContent: "center",
                  gap: 6,
                }}
              >
                <span style={{ width: 8, height: 8, borderRadius: "50%", background: STATUS_COLORS[s], display: "inline-block" }} />
                {STATUS_LABELS[s]}
              </button>
            ))}
          </div>
        </div>

        {/* Биография */}
        <div>
          <div style={labelStyle}>О себе</div>
          <textarea
            maxLength={300}
            value={profile.bio}
            onChange={(e) => update({ bio: e.target.value })}
            placeholder="Расскажите о себе..."
            style={{
              width: "100%",
              minHeight: 80,
              background: "var(--bg-tertiary, #202225)",
              border: "none",
              borderRadius: 4,
              padding: "8px 10px",
              color: "var(--text-primary, #dcddde)",
              fontSize: 14,
              resize: "vertical",
              fontFamily: "inherit",
            }}
          />
          <div style={{ fontSize: 11, color: "var(--text-muted, #72767d)", textAlign: "right", marginTop: 4 }}>
            {profile.bio.length}/300
          </div>
        </div>

        {/* Кнопки */}
        <div style={{ display: "flex", justifyContent: "flex-end", gap: 8 }}>
          <button
            style={{
              padding: "8px 16px",
              borderRadius: 4,
              border: "none",
              cursor: "pointer",
              background: "var(--bg-tertiary, #202225)",
              color: "var(--text-muted, #72767d)",
              fontWeight: 600,
              fontSize: 13,
            }}
            onClick={onClose}
          >
            Отмена
          </button>
          <button
            style={{
              padding: "8px 16px",
              borderRadius: 4,
              border: "none",
              cursor: "pointer",
              background: "var(--accent, #7289da)",
              color: "#fff",
              fontWeight: 600,
              fontSize: 13,
            }}
            onClick={handleSave}
          >
            Сохранить
          </button>
        </div>
      </div>
    </div>
  );
}
