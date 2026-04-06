/**
 * MoodAvatar — аватар с пульсирующим ореолом настроения
 */
import { useState, useEffect } from "react";
import { getMoodState, subscribeMood, trackTyping } from "../store/mood";
import type { MoodState } from "../store/mood";

interface MoodAvatarProps {
  username: string;
  size?: number;
  showTooltip?: boolean;
  onClick?: () => void;
}

export default function MoodAvatar({ username, size = 32, showTooltip = true, onClick }: MoodAvatarProps) {
  const [mood, setMood]       = useState<MoodState>(getMoodState);
  const [hovered, setHovered] = useState(false);
  const [pulse, setPulse]     = useState(false);

  useEffect(() => {
    const unsub = subscribeMood(setMood);
    return unsub;
  }, []);

  // Пульс при смене настроения
  useEffect(() => {
    setPulse(true);
    const t = setTimeout(() => setPulse(false), 600);
    return () => clearTimeout(t);
  }, [mood.mood]);

  // Отслеживаем активность мыши
  useEffect(() => {
    const handler = () => trackTyping();
    window.addEventListener("mousemove", handler, { passive: true });
    return () => window.removeEventListener("mousemove", handler);
  }, []);

  const letter = username.charAt(0).toUpperCase();

  return (
    <div
      style={{ position: "relative", width: size, height: size, flexShrink: 0, cursor: onClick ? "pointer" : "default" }}
      onMouseEnter={() => setHovered(true)}
      onMouseLeave={() => setHovered(false)}
      onClick={onClick}
    >
      {/* Outer glow ring */}
      <div style={{
        position: "absolute",
        inset: -3,
        borderRadius: "50%",
        background: `conic-gradient(${mood.color}, ${mood.color}88, ${mood.color})`,
        opacity: pulse ? 1 : 0.7,
        transition: "opacity 0.3s, background 0.8s",
        animation: mood.mood !== "away" ? "moodSpin 4s linear infinite" : "none",
      }} />

      {/* Glow blur */}
      <div style={{
        position: "absolute",
        inset: -4,
        borderRadius: "50%",
        background: mood.glowColor,
        filter: "blur(6px)",
        opacity: hovered ? 0.8 : 0.4,
        transition: "opacity 0.3s, background 0.8s",
      }} />

      {/* Avatar */}
      <div style={{
        position: "relative",
        width: size, height: size, borderRadius: "50%",
        background: `linear-gradient(135deg, ${mood.color}, ${mood.color}99)`,
        display: "flex", alignItems: "center", justifyContent: "center",
        fontSize: size * 0.4, fontWeight: 700, color: "#fff",
        zIndex: 1,
        transition: "background 0.8s",
        border: "2px solid rgba(0,0,0,0.3)",
      }}>
        {letter}
      </div>

      {/* Mood dot */}
      <div style={{
        position: "absolute",
        bottom: -1, right: -1,
        width: size * 0.35, height: size * 0.35,
        borderRadius: "50%",
        background: mood.color,
        border: "2px solid var(--bg-base, #0d0e14)",
        zIndex: 2,
        display: "flex", alignItems: "center", justifyContent: "center",
        fontSize: size * 0.18,
        boxShadow: `0 0 6px ${mood.glowColor}`,
        transition: "background 0.8s",
      }}>
        {mood.emoji}
      </div>

      {/* Tooltip */}
      {showTooltip && hovered && (
        <div style={{
          position: "absolute",
          bottom: "calc(100% + 8px)",
          left: "50%",
          transform: "translateX(-50%)",
          background: "var(--bg-elevated, #1a1b27)",
          border: `1px solid ${mood.color}44`,
          borderRadius: 10,
          padding: "8px 12px",
          zIndex: 100,
          whiteSpace: "nowrap",
          boxShadow: `0 4px 20px rgba(0,0,0,0.5), 0 0 12px ${mood.glowColor}`,
          animation: "fadeUp 0.15s ease",
          pointerEvents: "none",
        }}>
          <div style={{ display: "flex", alignItems: "center", gap: 6 }}>
            <span style={{ fontSize: 16 }}>{mood.emoji}</span>
            <div>
              <div style={{ fontSize: 12, fontWeight: 700, color: mood.color }}>{mood.label}</div>
              <div style={{ fontSize: 10, color: "var(--text-muted, #5a5b70)" }}>{mood.description}</div>
            </div>
          </div>
          {/* Arrow */}
          <div style={{
            position: "absolute",
            top: "100%", left: "50%",
            transform: "translateX(-50%)",
            width: 0, height: 0,
            borderLeft: "5px solid transparent",
            borderRight: "5px solid transparent",
            borderTop: `5px solid ${mood.color}44`,
          }} />
        </div>
      )}
    </div>
  );
}
