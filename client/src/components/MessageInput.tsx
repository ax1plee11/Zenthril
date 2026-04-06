import React, { useState, useRef, useCallback } from "react";
import GifPicker from "./GifPicker";
import { trackTyping } from "../store/mood";

interface MessageInputProps {
  channelName: string;
  onSend: (text: string) => Promise<void>;
  disabled?: boolean;
}

export default function MessageInput({ channelName, onSend, disabled = false }: MessageInputProps) {
  const [text, setText]         = useState("");
  const [sending, setSending]   = useState(false);
  const [focused, setFocused]   = useState(false);
  const [showGif, setShowGif]   = useState(false);
  const ref = useRef<HTMLTextAreaElement>(null);
  const wrapRef = useRef<HTMLDivElement>(null);

  const handleSend = useCallback(async (override?: string) => {
    const t = (override ?? text).trim();
    if (!t || sending || disabled) return;
    setSending(true);
    try {
      await onSend(t);
      setText("");
      if (ref.current) ref.current.style.height = "auto";
    } finally { setSending(false); }
  }, [text, sending, disabled, onSend]);

  const handleKeyDown = useCallback((e: React.KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.key === "Enter" && !e.shiftKey) { e.preventDefault(); handleSend(); }
  }, [handleSend]);

  const handleChange = useCallback((e: React.ChangeEvent<HTMLTextAreaElement>) => {
    setText(e.target.value);
    trackTyping();
    const el = e.target;
    el.style.height = "auto";
    el.style.height = `${Math.min(el.scrollHeight, 180)}px`;
  }, []);

  const handleGifSelect = useCallback((url: string) => {
    handleSend(url);
    setShowGif(false);
  }, [handleSend]);

  const hasText = text.trim().length > 0;

  return (
    <div style={{ padding: "0 16px 20px", flexShrink: 0, position: "relative" as const }} ref={wrapRef}>
      {/* GIF Picker */}
      {showGif && (
        <GifPicker onSelect={handleGifSelect} onClose={() => setShowGif(false)} />
      )}

      <div style={{
        display: "flex", alignItems: "flex-end", gap: 6,
        background: "var(--bg-elevated)",
        border: `1px solid ${focused ? "var(--border-accent)" : "var(--border)"}`,
        borderRadius: "var(--radius-lg)", padding: "4px 6px 4px 14px",
        transition: "border-color 0.2s, box-shadow 0.2s",
        boxShadow: focused ? "0 0 0 3px rgba(124,106,247,0.1)" : "none",
      }}>
        <textarea
          ref={ref}
          style={{
            flex: 1, background: "none", border: "none", outline: "none",
            color: "var(--text-primary)", fontSize: 14, lineHeight: 1.6,
            padding: "10px 0", resize: "none" as const, fontFamily: "inherit",
            maxHeight: 180, overflowY: "auto" as const,
          }}
          value={text}
          onChange={handleChange}
          onKeyDown={handleKeyDown}
          onFocus={() => setFocused(true)}
          onBlur={() => setFocused(false)}
          placeholder={disabled ? "Select a channel..." : `Message #${channelName}`}
          disabled={disabled || sending}
          rows={1}
          maxLength={4000}
        />

        {/* GIF button */}
        <button
          onClick={() => setShowGif(v => !v)}
          disabled={disabled}
          title="Send a GIF"
          style={{
            width: 34, height: 34, borderRadius: 9, border: "none",
            background: showGif ? "var(--accent-dim)" : "none",
            color: showGif ? "var(--accent)" : "var(--text-muted)",
            cursor: disabled ? "default" : "pointer",
            display: "flex", alignItems: "center", justifyContent: "center",
            flexShrink: 0, transition: "all 0.15s", fontSize: 11, fontWeight: 800,
            letterSpacing: -0.5,
          }}
          onMouseEnter={e => { if (!disabled) (e.currentTarget as HTMLButtonElement).style.color = "var(--accent)"; }}
          onMouseLeave={e => { if (!showGif) (e.currentTarget as HTMLButtonElement).style.color = "var(--text-muted)"; }}
        >
          GIF
        </button>

        {/* Send button */}
        <button
          onClick={() => handleSend()}
          disabled={!hasText || sending || disabled}
          style={{
            width: 34, height: 34, borderRadius: 9, border: "none",
            background: hasText ? "linear-gradient(135deg, #7c6af7, #a78bfa)" : "var(--bg-input)",
            color: hasText ? "#fff" : "var(--text-muted)",
            cursor: hasText ? "pointer" : "default",
            display: "flex", alignItems: "center", justifyContent: "center",
            flexShrink: 0, transition: "all 0.2s",
            boxShadow: hasText ? "0 2px 8px rgba(124,106,247,0.4)" : "none",
          }}
        >
          <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5">
            <line x1="22" y1="2" x2="11" y2="13"/><polygon points="22 2 15 22 11 13 2 9 22 2"/>
          </svg>
        </button>
      </div>
    </div>
  );
}
