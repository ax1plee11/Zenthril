/**
 * MessageInput — поле ввода сообщения
 * Требования: 3.1 (шифрование перед отправкой)
 */

import React, { useState, useRef, useCallback } from "react";

interface MessageInputProps {
  channelName: string;
  onSend: (text: string) => Promise<void>;
  disabled?: boolean;
}

const styles = {
  wrapper: {
    padding: "0 16px 24px 16px",
    flexShrink: 0,
  } as React.CSSProperties,

  inputBox: {
    display: "flex",
    alignItems: "flex-end",
    background: "#40444b",
    borderRadius: 8,
    padding: "0 16px",
    gap: 8,
  } as React.CSSProperties,

  textarea: {
    flex: 1,
    background: "none",
    border: "none",
    outline: "none",
    color: "#dcddde",
    fontSize: 15,
    lineHeight: 1.5,
    padding: "11px 0",
    resize: "none" as const,
    fontFamily: "inherit",
    maxHeight: 200,
    overflowY: "auto" as const,
  } as React.CSSProperties,

  sendBtn: (hasText: boolean): React.CSSProperties => ({
    background: "none",
    border: "none",
    cursor: hasText ? "pointer" : "default",
    color: hasText ? "#7289da" : "#4f545c",
    fontSize: 20,
    padding: "8px 0",
    flexShrink: 0,
    transition: "color 0.1s",
  }),
};

export default function MessageInput({
  channelName,
  onSend,
  disabled = false,
}: MessageInputProps) {
  const [text, setText] = useState("");
  const [sending, setSending] = useState(false);
  const textareaRef = useRef<HTMLTextAreaElement>(null);

  const handleSend = useCallback(async () => {
    const trimmed = text.trim();
    if (!trimmed || sending || disabled) return;

    setSending(true);
    try {
      await onSend(trimmed);
      setText("");
      // Сбрасываем высоту textarea
      if (textareaRef.current) {
        textareaRef.current.style.height = "auto";
      }
    } finally {
      setSending(false);
    }
  }, [text, sending, disabled, onSend]);

  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
      if (e.key === "Enter" && !e.shiftKey) {
        e.preventDefault();
        handleSend();
      }
    },
    [handleSend],
  );

  const handleChange = useCallback(
    (e: React.ChangeEvent<HTMLTextAreaElement>) => {
      setText(e.target.value);
      // Авторесайз
      const el = e.target;
      el.style.height = "auto";
      el.style.height = `${Math.min(el.scrollHeight, 200)}px`;
    },
    [],
  );

  return (
    <div style={styles.wrapper}>
      <div style={styles.inputBox}>
        <textarea
          ref={textareaRef}
          style={styles.textarea}
          value={text}
          onChange={handleChange}
          onKeyDown={handleKeyDown}
          placeholder={disabled ? "Выберите канал..." : `Написать в #${channelName}`}
          disabled={disabled || sending}
          rows={1}
          maxLength={4000}
        />
        <button
          style={styles.sendBtn(text.trim().length > 0)}
          onClick={handleSend}
          disabled={!text.trim() || sending || disabled}
          title="Отправить"
        >
          ➤
        </button>
      </div>
    </div>
  );
}
