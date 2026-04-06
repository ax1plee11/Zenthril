/**
 * MessageItem — один элемент сообщения
 * Требования: 3.4 (форматирование), 3.5, 3.6 (редактирование/удаление)
 */

import React, { useState, useCallback } from "react";
import type { MessageAPI } from "../api/index";

interface MessageItemProps {
  message: MessageAPI;
  currentUserId: string;
  onEdit: (messageId: string, newText: string) => Promise<void>;
  onDelete: (messageId: string) => Promise<void>;
}

const styles = {
  wrapper: (hovered: boolean): React.CSSProperties => ({
    display: "flex",
    gap: 12,
    padding: "4px 16px",
    background: hovered ? "rgba(4,4,5,0.07)" : "transparent",
    position: "relative",
  }),

  avatar: {
    width: 40,
    height: 40,
    borderRadius: "50%",
    background: "#7289da",
    display: "flex",
    alignItems: "center",
    justifyContent: "center",
    fontSize: 16,
    fontWeight: 700,
    color: "#fff",
    flexShrink: 0,
    marginTop: 2,
  } as React.CSSProperties,

  content: {
    flex: 1,
    minWidth: 0,
  } as React.CSSProperties,

  header: {
    display: "flex",
    alignItems: "baseline",
    gap: 8,
    marginBottom: 2,
  } as React.CSSProperties,

  author: {
    fontWeight: 600,
    fontSize: 15,
    color: "#fff",
  } as React.CSSProperties,

  timestamp: {
    fontSize: 11,
    color: "#72767d",
  } as React.CSSProperties,

  editedBadge: {
    fontSize: 10,
    color: "#72767d",
    fontStyle: "italic",
  } as React.CSSProperties,

  text: {
    fontSize: 15,
    color: "#dcddde",
    lineHeight: 1.5,
    wordBreak: "break-word" as const,
  } as React.CSSProperties,

  deletedText: {
    fontSize: 14,
    color: "#72767d",
    fontStyle: "italic",
  } as React.CSSProperties,

  actions: {
    position: "absolute" as const,
    top: 0,
    right: 16,
    display: "flex",
    gap: 4,
    background: "#36393f",
    border: "1px solid #202225",
    borderRadius: 4,
    padding: "2px 4px",
  } as React.CSSProperties,

  actionBtn: {
    background: "none",
    border: "none",
    cursor: "pointer",
    color: "#b9bbbe",
    fontSize: 14,
    padding: "2px 6px",
    borderRadius: 3,
    transition: "color 0.1s, background 0.1s",
  } as React.CSSProperties,

  editInput: {
    width: "100%",
    background: "#40444b",
    border: "1px solid #7289da",
    borderRadius: 4,
    color: "#dcddde",
    fontSize: 15,
    padding: "8px 12px",
    outline: "none",
    boxSizing: "border-box" as const,
    resize: "none" as const,
    fontFamily: "inherit",
    lineHeight: 1.5,
  } as React.CSSProperties,

  editHint: {
    fontSize: 12,
    color: "#72767d",
    marginTop: 4,
  } as React.CSSProperties,
};

/**
 * Форматирование текста: **жирный**, *курсив*, `код`, @mention
 * Требование 3.4
 */
function formatText(text: string, currentUsername?: string): React.ReactNode[] {
  // Разбиваем по блокам кода сначала
  const parts: React.ReactNode[] = [];
  let remaining = text;
  let key = 0;

  // Обрабатываем inline-форматирование
  const segments = remaining.split(/(`[^`]+`|\*\*[^*]+\*\*|\*[^*]+\*|@\w+)/g);

  for (const seg of segments) {
    if (!seg) continue;

    if (seg.startsWith("**") && seg.endsWith("**") && seg.length > 4) {
      parts.push(<strong key={key++}>{seg.slice(2, -2)}</strong>);
    } else if (seg.startsWith("*") && seg.endsWith("*") && seg.length > 2) {
      parts.push(<em key={key++}>{seg.slice(1, -1)}</em>);
    } else if (seg.startsWith("`") && seg.endsWith("`") && seg.length > 2) {
      parts.push(
        <code
          key={key++}
          style={{
            background: "#2f3136",
            borderRadius: 3,
            padding: "1px 4px",
            fontFamily: "monospace",
            fontSize: 13,
            color: "#e3e5e8",
          }}
        >
          {seg.slice(1, -1)}
        </code>,
      );
    } else if (seg.startsWith("@")) {
      const mentionName = seg.slice(1);
      const isMe = currentUsername && mentionName === currentUsername;
      parts.push(
        <span
          key={key++}
          style={{
            background: isMe ? "rgba(114,137,218,0.3)" : "rgba(114,137,218,0.15)",
            color: "#7289da",
            borderRadius: 3,
            padding: "0 2px",
            fontWeight: 600,
          }}
        >
          {seg}
        </span>,
      );
    } else {
      parts.push(<React.Fragment key={key++}>{seg}</React.Fragment>);
    }
  }

  return parts;
}

function formatTime(iso: string): string {
  const d = new Date(iso);
  const now = new Date();
  const isToday =
    d.getDate() === now.getDate() &&
    d.getMonth() === now.getMonth() &&
    d.getFullYear() === now.getFullYear();

  if (isToday) {
    return `Сегодня в ${d.toLocaleTimeString("ru-RU", { hour: "2-digit", minute: "2-digit" })}`;
  }
  return d.toLocaleDateString("ru-RU", {
    day: "2-digit",
    month: "2-digit",
    year: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  });
}

export default function MessageItem({
  message,
  currentUserId,
  onEdit,
  onDelete,
}: MessageItemProps) {
  const [hovered, setHovered] = useState(false);
  const [editing, setEditing] = useState(false);
  const [editText, setEditText] = useState("");

  const isOwn = message.author_id === currentUserId;
  const displayText = message.decryptedContent ?? `[зашифровано: ${message.payload.ciphertext.slice(0, 20)}...]`;
  const authorLetter = (message.author_username ?? "?").charAt(0).toUpperCase();

  const startEdit = useCallback(() => {
    setEditText(message.decryptedContent ?? "");
    setEditing(true);
  }, [message.decryptedContent]);

  const submitEdit = useCallback(
    async (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
      if (e.key === "Enter" && !e.shiftKey) {
        e.preventDefault();
        if (editText.trim()) {
          await onEdit(message.id, editText.trim());
        }
        setEditing(false);
      }
      if (e.key === "Escape") {
        setEditing(false);
      }
    },
    [editText, message.id, onEdit],
  );

  return (
    <div
      style={styles.wrapper(hovered)}
      onMouseEnter={() => setHovered(true)}
      onMouseLeave={() => setHovered(false)}
      onDoubleClick={isOwn && !message.deleted ? startEdit : undefined}
    >
      <div style={styles.avatar}>{authorLetter}</div>

      <div style={styles.content}>
        <div style={styles.header}>
          <span style={styles.author}>
            {message.author_username ?? "Неизвестный"}
          </span>
          <span style={styles.timestamp}>{formatTime(message.created_at)}</span>
          {message.edited && !message.deleted && (
            <span style={styles.editedBadge}>(изменено)</span>
          )}
        </div>

        {message.deleted ? (
          <div style={styles.deletedText}>
            <em>Сообщение удалено</em>
          </div>
        ) : editing ? (
          <div>
            <textarea
              style={styles.editInput}
              value={editText}
              onChange={(e) => setEditText(e.target.value)}
              onKeyDown={submitEdit}
              autoFocus
              rows={2}
            />
            <div style={styles.editHint}>
              Enter — сохранить · Escape — отмена · Shift+Enter — новая строка
            </div>
          </div>
        ) : (
          <div style={styles.text}>
            {formatText(displayText, message.author_username)}
          </div>
        )}
      </div>

      {hovered && isOwn && !message.deleted && !editing && (
        <div style={styles.actions}>
          <button
            style={styles.actionBtn}
            onClick={startEdit}
            title="Редактировать"
          >
            ✏️
          </button>
          <button
            style={styles.actionBtn}
            onClick={() => onDelete(message.id)}
            title="Удалить"
          >
            🗑️
          </button>
        </div>
      )}
    </div>
  );
}
