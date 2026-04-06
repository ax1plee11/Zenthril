import React, { useState, useCallback, useRef } from "react";
import type { MessageAPI } from "../api/index";
import MoodAvatar from "./MoodAvatar";
import UserProfileCard from "./UserProfileCard";

export interface MessageItemProps {
  message: MessageAPI;
  currentUserId: string;
  onEdit: (id: string, text: string) => Promise<void>;
  onDelete: (id: string) => Promise<void>;
  /** Если true — скрываем аватар и имя (продолжение группы) */
  grouped?: boolean;
}

function proxyUrl(url: string): string {
  if (url.startsWith("data:")) return url;
  if (url.startsWith("/") || url.includes("localhost") || url.includes("127.0.0.1")) return url;
  return `https://images.weserv.nl/?url=${encodeURIComponent(url)}&maxage=7d`;
}

function detectMedia(text: string): { url: string; type: "gif" | "image" | "link" | null } {
  const trimmed = text.trim();
  if (!/^https?:\/\//i.test(trimmed)) return { url: trimmed, type: null };
  if (/\.gifv(\?.*)?$/i.test(trimmed)) return { url: trimmed.replace(/\.gifv(\?.*)?$/i, ".gif$1"), type: "gif" };
  if (/\.gif(\?.*)?$/i.test(trimmed)) return { url: trimmed, type: "gif" };
  // Tenor/Giphy media hosts
  if (/(^|\/\/)(media\.tenor\.com|c\.tenor\.com)\//i.test(trimmed)) return { url: trimmed, type: "gif" };
  if (/(^|\/\/)(media\.giphy\.com|i\.giphy\.com)\//i.test(trimmed)) return { url: trimmed, type: "gif" };
  if (/giphy\.com\/media\//i.test(trimmed)) return { url: trimmed, type: "gif" };
  if (/\.(jpg|jpeg|png|webp|bmp|tiff|avif|svg)(\?.*)?$/i.test(trimmed)) return { url: trimmed, type: "image" };
  if (/imgur\.com\//i.test(trimmed)) return { url: trimmed, type: "image" };
  return { url: trimmed, type: "link" };
}

const isGifUrl   = (t: string) => { const { url, type } = detectMedia(t); return type === "gif"   ? url : null; };
const isImageUrl = (t: string) => { const { url, type } = detectMedia(t); return type === "image" ? url : null; };
const isLinkUrl  = (t: string) => { const { url, type } = detectMedia(t); return type === "link"  ? url : null; };

function formatText(text: string): React.ReactNode[] {
  const parts: React.ReactNode[] = [];
  const segments = text.split(/(`[^`]+`|\*\*[^*]+\*\*|\*[^*]+\*|@\w+)/g);
  let key = 0;
  for (const seg of segments) {
    if (!seg) continue;
    if (seg.startsWith("**") && seg.endsWith("**") && seg.length > 4)
      parts.push(<strong key={key++} style={{ color: "var(--text-primary)" }}>{seg.slice(2,-2)}</strong>);
    else if (seg.startsWith("*") && seg.endsWith("*") && seg.length > 2)
      parts.push(<em key={key++}>{seg.slice(1,-1)}</em>);
    else if (seg.startsWith("`") && seg.endsWith("`") && seg.length > 2)
      parts.push(<code key={key++} style={{
        background: "rgba(255,255,255,0.06)", borderRadius: 4,
        padding: "1px 6px", fontFamily: "monospace", fontSize: 13,
        color: "#a78bfa", border: "1px solid rgba(255,255,255,0.08)",
      }}>{seg.slice(1,-1)}</code>);
    else if (seg.startsWith("@"))
      parts.push(<span key={key++} style={{
        background: "rgba(124,106,247,0.2)", color: "#a78bfa",
        borderRadius: 4, padding: "0 4px", fontWeight: 600,
      }}>{seg}</span>);
    else
      parts.push(<React.Fragment key={key++}>{seg}</React.Fragment>);
  }
  return parts;
}

export function formatTime(iso: string): string {
  const d = new Date(iso), now = new Date();
  const isToday = d.toDateString() === now.toDateString();
  return isToday
    ? d.toLocaleTimeString("en", { hour: "2-digit", minute: "2-digit" })
    : d.toLocaleDateString("en", { month: "short", day: "numeric", hour: "2-digit", minute: "2-digit" });
}

export function formatDateDivider(iso: string): string {
  const d = new Date(iso), now = new Date();
  const yesterday = new Date(now); yesterday.setDate(now.getDate() - 1);
  if (d.toDateString() === now.toDateString()) return "Сегодня";
  if (d.toDateString() === yesterday.toDateString()) return "Вчера";
  return d.toLocaleDateString("ru", { day: "numeric", month: "long", year: "numeric" });
}

/** Возвращает true если два сообщения можно сгруппировать */
export function canGroup(a: MessageAPI, b: MessageAPI): boolean {
  if (a.author_id !== b.author_id) return false;
  if (a.deleted || b.deleted) return false;
  const diff = new Date(b.created_at).getTime() - new Date(a.created_at).getTime();
  return diff < 5 * 60 * 1000; // 5 минут
}

const AVATAR_COLORS = ["#7c6af7","#3ecf8e","#f5a623","#f04f5e","#06b6d4","#ec4899","#8b5cf6"];

function MediaMessage({ url, isGif }: { url: string; isGif: boolean }) {
  const [status, setStatus] = useState<"loading" | "proxied" | "direct" | "error">("loading");
  const proxied = proxyUrl(url);
  const src = status === "direct" ? url : proxied;

  return (
    <div style={{ marginTop: 4, display: "inline-block", position: "relative" as const }}>
      {status === "loading" && (
        <div style={{
          width: 240, height: 160, borderRadius: 10,
          background: "var(--bg-elevated)", border: "1px solid var(--border)",
          display: "flex", alignItems: "center", justifyContent: "center",
          flexDirection: "column" as const, gap: 8,
        }}>
          <div style={{
            width: 20, height: 20, border: "2px solid var(--border)",
            borderTopColor: "var(--accent)", borderRadius: "50%",
            animation: "spin 0.7s linear infinite",
          }} />
          <div style={{ fontSize: 10, color: "var(--text-muted)" }}>Loading...</div>
        </div>
      )}
      {status === "error" && (
        <a href={url} target="_blank" rel="noopener noreferrer" style={{
          color: "var(--accent)", fontSize: 13, wordBreak: "break-all" as const,
          textDecoration: "underline", display: "block",
        }}>{url}</a>
      )}
      {status !== "error" && (
        <img key={src} src={src} alt="media" referrerPolicy="no-referrer"
          style={{
            maxWidth: isGif ? 480 : 360, maxHeight: isGif ? 400 : 300, borderRadius: 10,
            display: status === "loading" ? "none" : "block",
            border: "1px solid var(--border)", cursor: "pointer",
            transition: "transform 0.15s",
          }}
          onLoad={() => setStatus(status === "loading" ? "proxied" : status)}
          onError={() => { setStatus(s => s === "loading" || s === "proxied" ? "direct" : "error"); }}
          onClick={() => window.open(url, "_blank")}
          onMouseEnter={e => { (e.target as HTMLImageElement).style.transform = "scale(1.02)"; }}
          onMouseLeave={e => { (e.target as HTMLImageElement).style.transform = "scale(1)"; }}
        />
      )}
      {(status === "proxied" || status === "direct") && isGif && (
        <div style={{
          position: "absolute" as const, bottom: 6, left: 6,
          background: "rgba(0,0,0,0.75)", borderRadius: 4,
          padding: "2px 6px", fontSize: 9, fontWeight: 800, color: "#fff", letterSpacing: 0.5,
        }}>GIF</div>
      )}
    </div>
  );
}

function LinkPreview({ url }: { url: string }) {
  let hostname = url;
  try { hostname = new URL(url).hostname.replace("www.", ""); } catch { /* ignore */ }
  return (
    <a href={url} target="_blank" rel="noopener noreferrer" style={{
      display: "inline-flex", alignItems: "center", gap: 6,
      padding: "6px 10px", borderRadius: 8, marginTop: 2,
      background: "rgba(255,255,255,0.04)", border: "1px solid var(--border)",
      color: "var(--accent)", fontSize: 13, textDecoration: "none",
      maxWidth: 400, overflow: "hidden", transition: "background 0.15s",
    }}
    onMouseEnter={e => { (e.currentTarget as HTMLAnchorElement).style.background = "rgba(124,106,247,0.1)"; }}
    onMouseLeave={e => { (e.currentTarget as HTMLAnchorElement).style.background = "rgba(255,255,255,0.04)"; }}>
      <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" style={{ flexShrink: 0 }}>
        <path d="M10 13a5 5 0 0 0 7.54.54l3-3a5 5 0 0 0-7.07-7.07l-1.72 1.71"/>
        <path d="M14 11a5 5 0 0 0-7.54-.54l-3 3a5 5 0 0 0 7.07 7.07l1.71-1.71"/>
      </svg>
      <span style={{ overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap" as const }}>{hostname}</span>
    </a>
  );
}

export default function MessageItem({ message, currentUserId, onEdit, onDelete, grouped = false }: MessageItemProps) {
  const [hovered, setHovered]   = useState(false);
  const [editing, setEditing]   = useState(false);
  const [editText, setEditText] = useState("");
  const [showCard, setShowCard] = useState(false);
  const avatarRef = useRef<HTMLDivElement>(null);

  const isOwn       = message.author_id === currentUserId;
  const rawText     = message.decryptedContent;
  const displayText = rawText ?? "[encrypted]";
  const isDecrypted = rawText !== undefined;
  const name        = message.author_username ?? "Unknown";
  const color       = AVATAR_COLORS[name.charCodeAt(0) % AVATAR_COLORS.length];

  const startEdit = useCallback(() => { setEditText(message.decryptedContent ?? ""); setEditing(true); }, [message.decryptedContent]);
  const submitEdit = useCallback(async (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      if (editText.trim()) await onEdit(message.id, editText.trim());
      setEditing(false);
    }
    if (e.key === "Escape") setEditing(false);
  }, [editText, message.id, onEdit]);

  return (
    <div
      style={{
        display: "flex", gap: 12,
        padding: grouped ? "1px 20px" : "6px 20px",
        paddingTop: grouped ? 1 : 8,
        background: hovered ? "rgba(255,255,255,0.02)" : "transparent",
        position: "relative" as const, transition: "background 0.1s",
        animation: grouped ? "none" : "fadeUp 0.2s ease",
      }}
      onMouseEnter={() => setHovered(true)}
      onMouseLeave={() => setHovered(false)}
      onDoubleClick={isOwn && !message.deleted ? startEdit : undefined}
    >
      {/* Avatar column — 38px wide always для выравнивания */}
      <div ref={avatarRef} style={{ width: 38, flexShrink: 0, marginTop: grouped ? 0 : 2 }}>
        {grouped ? (
          /* Вместо аватара — время при hover */
          hovered ? (
            <div style={{
              fontSize: 10, color: "var(--text-muted)", textAlign: "right" as const,
              paddingTop: 3, lineHeight: 1.4,
            }}>
              {new Date(message.created_at).toLocaleTimeString("en", { hour: "2-digit", minute: "2-digit" })}
            </div>
          ) : null
        ) : isOwn ? (
          <MoodAvatar username={name} size={38} showTooltip={false} onClick={() => setShowCard(v => !v)} />
        ) : (
          <div style={{
            width: 38, height: 38, borderRadius: 12,
            background: `linear-gradient(135deg, ${color}, ${color}99)`,
            display: "flex", alignItems: "center", justifyContent: "center",
            fontSize: 15, fontWeight: 700, color: "#fff",
            boxShadow: `0 2px 8px ${color}44`, cursor: "pointer",
          }} onClick={() => setShowCard(v => !v)}>
            {name.charAt(0).toUpperCase()}
          </div>
        )}
      </div>

      {showCard && avatarRef.current && (
        <UserProfileCard
          username={name} userId={message.author_id}
          currentUserId={currentUserId}
          anchorEl={avatarRef.current}
          onClose={() => setShowCard(false)}
        />
      )}

      {/* Content */}
      <div style={{ flex: 1, minWidth: 0 }}>
        {/* Header — только для первого в группе */}
        {!grouped && (
          <div style={{ display: "flex", alignItems: "baseline", gap: 8, marginBottom: 3 }}>
            <span style={{ fontWeight: 700, fontSize: 14, color }}>{name}</span>
            <span style={{ fontSize: 11, color: "var(--text-muted)" }}>{formatTime(message.created_at)}</span>
            {message.edited && !message.deleted && (
              <span style={{ fontSize: 10, color: "var(--text-muted)", fontStyle: "italic" }}>(edited)</span>
            )}
          </div>
        )}

        {message.deleted ? (
          <div style={{
            fontSize: 13, color: "var(--text-muted)", fontStyle: "italic",
            background: "rgba(255,255,255,0.03)", borderRadius: "var(--radius-sm)",
            padding: "4px 10px", display: "inline-block", border: "1px solid var(--border)",
          }}>Message deleted</div>
        ) : editing ? (
          <div>
            <textarea style={{
              width: "100%", background: "var(--bg-input)",
              border: "1px solid var(--accent)", borderRadius: "var(--radius-sm)",
              color: "var(--text-primary)", fontSize: 14, padding: "8px 12px",
              outline: "none", resize: "none" as const, fontFamily: "inherit",
              lineHeight: 1.5, boxShadow: "0 0 0 3px rgba(124,106,247,0.15)",
            }}
              value={editText} onChange={e => setEditText(e.target.value)}
              onKeyDown={submitEdit} autoFocus rows={2}
            />
            <div style={{ fontSize: 11, color: "var(--text-muted)", marginTop: 4 }}>
              Enter to save · Escape to cancel
            </div>
          </div>
        ) : (
          <div style={{ fontSize: 14, color: "var(--text-secondary)", lineHeight: 1.6, wordBreak: "break-word" as const }}>
            {(() => {
              if (!isDecrypted) {
                if (!message.payload?.tag) return (
                  <span style={{ color: "var(--text-muted)", fontSize: 13, fontFamily: "monospace" }}>
                    {message.payload?.ciphertext?.slice(0, 40)}...
                  </span>
                );
                return <span style={{ color: "var(--text-muted)", fontStyle: "italic", fontSize: 13 }}>🔒 Decrypting...</span>;
              }
              const gifUrl  = isGifUrl(displayText);
              const imgUrl  = !gifUrl ? isImageUrl(displayText) : null;
              const linkUrl = !gifUrl && !imgUrl ? isLinkUrl(displayText) : null;
              if (gifUrl)  return <MediaMessage url={gifUrl} isGif />;
              if (imgUrl)  return <MediaMessage url={imgUrl} isGif={false} />;
              if (linkUrl) return <div><div style={{ marginBottom: 4 }}>{formatText(displayText)}</div><LinkPreview url={linkUrl} /></div>;
              return formatText(displayText);
            })()}
          </div>
        )}
      </div>

      {/* Actions */}
      {hovered && isOwn && !message.deleted && !editing && (
        <div style={{
          position: "absolute" as const, top: 4, right: 16,
          display: "flex", gap: 2,
          background: "var(--bg-elevated)", border: "1px solid var(--border)",
          borderRadius: "var(--radius-sm)", padding: "3px 4px",
          boxShadow: "var(--shadow-sm)", animation: "fadeIn 0.1s ease",
        }}>
          {[
            { icon: "✏️", label: "Edit",   action: startEdit },
            { icon: "🗑️", label: "Delete", action: () => onDelete(message.id) },
          ].map(btn => (
            <button key={btn.label} title={btn.label} onClick={btn.action} style={{
              background: "none", border: "none", cursor: "pointer",
              color: "var(--text-secondary)", fontSize: 13, padding: "3px 7px",
              borderRadius: 4, transition: "background 0.1s",
            }}>{btn.icon}</button>
          ))}
        </div>
      )}
    </div>
  );
}
