/**
 * UserProfileCard — карточка профиля пользователя (показывается при клике на аватар)
 * Отображает тему, которую настроил владелец профиля
 */
import { useEffect, useRef } from "react";
import { getBannerStyle } from "./ProfileModal";
import type { UserProfile } from "./ProfileModal";

interface Props {
  username: string;
  userId: string;
  currentUserId: string;
  anchorEl: HTMLElement;   // элемент относительно которого позиционируемся
  onClose: () => void;
}

const STATUS_COLORS = { online: "#43b581", dnd: "#f04747", offline: "#747f8d" };
const STATUS_LABELS = { online: "Онлайн", dnd: "Не беспокоить", offline: "Офлайн" };
const PROFILE_KEY = "vibrora_profile";

/** Загружает профиль пользователя по userId из localStorage.
 *  В реальном приложении здесь был бы API-запрос.
 *  Для своего профиля — берём напрямую, для чужих — ключ с userId. */
function loadUserProfile(userId: string, currentUserId: string): UserProfile {
  const key = userId === currentUserId ? PROFILE_KEY : `vibrora_profile_${userId}`;
  try {
    const raw = localStorage.getItem(key);
    if (raw) {
      const p = JSON.parse(raw) as UserProfile;
      return {
        avatarBase64: p.avatarBase64 ?? null,
        status: p.status ?? "online",
        bio: p.bio ?? "",
        theme: {
          bannerPresetId: p.theme?.bannerPresetId ?? null,
          bannerColor: p.theme?.bannerColor ?? "#7c6af7",
          bannerUrl: p.theme?.bannerUrl ?? null,
        },
      };
    }
  } catch { /* ignore */ }
  return {
    avatarBase64: null, status: "online", bio: "",
    theme: { bannerPresetId: null, bannerColor: "#7c6af7", bannerUrl: null },
  };
}

export default function UserProfileCard({ username, userId, currentUserId, anchorEl, onClose }: Props) {
  const profile = loadUserProfile(userId, currentUserId);
  const cardRef = useRef<HTMLDivElement>(null);
  const bannerStyle = getBannerStyle(profile.theme);
  const initials = username.charAt(0).toUpperCase();

  // Позиционирование рядом с аватаром
  const rect = anchorEl.getBoundingClientRect();
  const spaceRight = window.innerWidth - rect.right;
  const cardWidth = 280;
  let left = rect.right + 8;
  if (spaceRight < cardWidth + 16) left = rect.left - cardWidth - 8;
  let top = rect.top;
  if (top + 360 > window.innerHeight) top = window.innerHeight - 370;

  // Закрытие по клику вне
  useEffect(() => {
    function handler(e: MouseEvent) {
      if (cardRef.current && !cardRef.current.contains(e.target as Node) &&
          !anchorEl.contains(e.target as Node)) {
        onClose();
      }
    }
    document.addEventListener("mousedown", handler);
    return () => document.removeEventListener("mousedown", handler);
  }, [onClose, anchorEl]);

  return (
    <div ref={cardRef} style={{
      position: "fixed",
      left, top,
      width: cardWidth,
      background: "var(--bg-surface)",
      borderRadius: 16,
      border: "1px solid var(--border)",
      boxShadow: "0 16px 48px rgba(0,0,0,0.7)",
      overflow: "hidden",
      zIndex: 2000,
      animation: "fadeUp 0.15s ease",
    }}>
      {/* Banner */}
      <div style={{
        height: 80,
        ...bannerStyle,
        backgroundSize: "400% 400%",
        animation: profile.theme.bannerPresetId ? "gradientShift 6s ease infinite" : "none",
        position: "relative",
      }}>
        {/* Avatar */}
        <div style={{
          position: "absolute", bottom: -22, left: 14,
          width: 48, height: 48, borderRadius: "50%",
          background: `linear-gradient(135deg, ${profile.theme.bannerColor}, ${profile.theme.bannerColor}99)`,
          border: `3px solid var(--bg-surface)`,
          display: "flex", alignItems: "center", justifyContent: "center",
          fontSize: 18, fontWeight: 700, color: "#fff",
          overflow: "hidden",
          boxShadow: `0 0 16px ${profile.theme.bannerColor}55`,
        }}>
          {profile.avatarBase64
            ? <img src={profile.avatarBase64} alt="av" style={{ width: "100%", height: "100%", objectFit: "cover" }} />
            : initials}
        </div>
      </div>

      {/* Content */}
      <div style={{ padding: "30px 14px 14px" }}>
        {/* Name + status */}
        <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between", marginBottom: 8 }}>
          <div style={{ fontWeight: 700, fontSize: 15, color: "var(--text-primary)" }}>{username}</div>
          <div style={{
            display: "flex", alignItems: "center", gap: 5,
            fontSize: 11, color: STATUS_COLORS[profile.status], fontWeight: 600,
          }}>
            <span style={{
              width: 7, height: 7, borderRadius: "50%",
              background: STATUS_COLORS[profile.status], display: "inline-block",
            }} />
            {STATUS_LABELS[profile.status]}
          </div>
        </div>

        {/* Divider */}
        <div style={{ height: 1, background: "var(--border)", margin: "10px 0" }} />

        {/* Bio */}
        {profile.bio ? (
          <div>
            <div style={{ fontSize: 10, fontWeight: 700, color: "var(--text-muted)", textTransform: "uppercase", letterSpacing: 0.8, marginBottom: 5 }}>
              О себе
            </div>
            <div style={{ fontSize: 13, color: "var(--text-secondary)", lineHeight: 1.5, wordBreak: "break-word" }}>
              {profile.bio}
            </div>
          </div>
        ) : (
          <div style={{ fontSize: 12, color: "var(--text-muted)", fontStyle: "italic" }}>
            Нет информации о пользователе
          </div>
        )}
      </div>
    </div>
  );
}
