/**
 * VoiceChannel — React-компонент голосового канала
 *
 * Требования: 4.1, 4.2, 4.3, 4.4, 4.5, 4.6
 */

import React, { useCallback, useEffect, useRef, useState } from "react";
import { VoiceClient } from "./index";
import { TransportLayer } from "../transport";

// ─── Иконки (SVG inline) ──────────────────────────────────────────────────────

function MicIcon({ muted }: { muted: boolean }) {
  return muted ? (
    <svg width="16" height="16" viewBox="0 0 24 24" fill="currentColor" aria-label="Микрофон выключен">
      <path d="M19 11a7 7 0 0 1-7 7m0 0a7 7 0 0 1-7-7m7 7v3m-4 0h8M2 2l20 20" stroke="currentColor" strokeWidth="2" fill="none" strokeLinecap="round"/>
    </svg>
  ) : (
    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" aria-label="Микрофон включён">
      <rect x="9" y="2" width="6" height="12" rx="3" stroke="currentColor" strokeWidth="2"/>
      <path d="M5 11a7 7 0 0 0 14 0M12 19v3M8 22h8" stroke="currentColor" strokeWidth="2" strokeLinecap="round"/>
    </svg>
  );
}

function QualityIndicator({ quality }: { quality: "good" | "poor" | null }) {
  if (!quality) return null;
  const color = quality === "good" ? "#4caf50" : "#f44336";
  const label = quality === "good" ? "Хорошее соединение" : "Нестабильное соединение";
  return (
    <span
      style={{ display: "inline-block", width: 10, height: 10, borderRadius: "50%", background: color, marginLeft: 6 }}
      title={label}
      aria-label={label}
    />
  );
}

// ─── Props ────────────────────────────────────────────────────────────────────

interface VoiceChannelProps {
  channelId: string;
  channelName: string;
  currentUserId: string;
  transport: TransportLayer;
}

// ─── Компонент ────────────────────────────────────────────────────────────────

export function VoiceChannel({ channelId, channelName, currentUserId, transport }: VoiceChannelProps) {
  const clientRef = useRef<VoiceClient | null>(null);

  const [connected, setConnected] = useState(false);
  const [muted, setMuted] = useState(false);
  const [participants, setParticipants] = useState<string[]>([]);
  const [quality, setQuality] = useState<"good" | "poor" | null>(null);
  const [error, setError] = useState<string | null>(null);

  // Инициализируем VoiceClient один раз
  useEffect(() => {
    clientRef.current = new VoiceClient(transport, currentUserId);
    return () => {
      clientRef.current?.leaveChannel().catch(() => {});
    };
  }, [transport, currentUserId]);

  const handleJoin = useCallback(async () => {
    const client = clientRef.current;
    if (!client) return;
    setError(null);
    try {
      client.onParticipantJoined((userId) => {
        setParticipants((prev) => prev.includes(userId) ? prev : [...prev, userId]);
      });
      client.onParticipantLeft((userId) => {
        setParticipants((prev) => prev.filter((id) => id !== userId));
      });
      client.onConnectionQuality((q) => setQuality(q));

      await client.joinChannel(channelId);
      setConnected(true);
      setMuted(false);
      setParticipants([]);
      setQuality(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Ошибка подключения");
    }
  }, [channelId]);

  const handleLeave = useCallback(async () => {
    const client = clientRef.current;
    if (!client) return;
    await client.leaveChannel();
    setConnected(false);
    setMuted(false);
    setParticipants([]);
    setQuality(null);
  }, []);

  const handleToggleMute = useCallback(async () => {
    const client = clientRef.current;
    if (!client) return;
    const newMuted = await client.toggleMute();
    setMuted(newMuted);
  }, []);

  return (
    <div className="voice-channel" style={{ padding: "12px", border: "1px solid #333", borderRadius: 8, minWidth: 220 }}>
      <div style={{ display: "flex", alignItems: "center", marginBottom: 8 }}>
        <strong>{channelName}</strong>
        {connected && <QualityIndicator quality={quality} />}
      </div>

      {error && (
        <div role="alert" style={{ color: "#f44336", fontSize: 12, marginBottom: 8 }}>
          {error}
        </div>
      )}

      {/* Список участников */}
      {connected && (
        <ul style={{ listStyle: "none", padding: 0, margin: "0 0 10px 0" }}>
          {participants.length === 0 && (
            <li style={{ fontSize: 12, color: "#888" }}>Нет других участников</li>
          )}
          {participants.map((userId) => (
            <li key={userId} style={{ display: "flex", alignItems: "center", gap: 6, fontSize: 13, padding: "2px 0" }}>
              <MicIcon muted={false} />
              <span>{userId}</span>
            </li>
          ))}
        </ul>
      )}

      {/* Кнопки управления */}
      <div style={{ display: "flex", gap: 8 }}>
        {!connected ? (
          <button
            onClick={handleJoin}
            style={{ padding: "6px 14px", borderRadius: 4, background: "#4caf50", color: "#fff", border: "none", cursor: "pointer" }}
          >
            Подключиться
          </button>
        ) : (
          <>
            <button
              onClick={handleLeave}
              style={{ padding: "6px 14px", borderRadius: 4, background: "#f44336", color: "#fff", border: "none", cursor: "pointer" }}
            >
              Отключиться
            </button>
            <button
              onClick={handleToggleMute}
              aria-pressed={muted}
              style={{
                padding: "6px 14px",
                borderRadius: 4,
                background: muted ? "#ff9800" : "#607d8b",
                color: "#fff",
                border: "none",
                cursor: "pointer",
                display: "flex",
                alignItems: "center",
                gap: 4,
              }}
            >
              <MicIcon muted={muted} />
              {muted ? "Снять мут" : "Мут"}
            </button>
          </>
        )}
      </div>
    </div>
  );
}

export default VoiceChannel;
