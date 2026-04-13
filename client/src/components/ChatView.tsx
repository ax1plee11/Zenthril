/**
 * ChatView — область чата с историей сообщений
 * Требования: 3.1, 3.2, 3.3, 3.5, 3.6
 */

import React, {
  useState,
  useEffect,
  useRef,
  useCallback,
} from "react";
import { api } from "../api/index";
import type { MessageAPI, EncryptedPayloadAPI } from "../api/index";
import {
  encrypt,
  getOrCreateSessionKey,
} from "../crypto/index";
import type { EncryptedPayload } from "../types/index";
import MessageItem, { canGroup, formatDateDivider } from "./MessageItem";
import MessageInput from "./MessageInput";
import { showNotification } from "../notifications/index";
import { useTheme } from "../store/theme";
import { trackMessage } from "../store/mood";
import { onWSEvent, sendWSEvent, connectGlobalWS } from "../store/wsGlobal";

interface ChatViewProps {
  channelId: string | null;
  channelName: string;
  currentUserId: string;
  currentUsername: string;
}

const styles = {
  root: {
    flex: 1, display: "flex", flexDirection: "column" as const,
    background: "transparent", minWidth: 0,
    position: "relative" as const,
  } as React.CSSProperties,
  header: {
    padding: "0 20px", height: 52,
    borderBottom: "1px solid var(--border)",
    display: "flex", alignItems: "center", gap: 10, flexShrink: 0,
    background: "var(--panel-topbar)",
    backdropFilter: "blur(16px)",
    WebkitBackdropFilter: "blur(16px)",
    boxShadow: "0 1px 0 rgba(255,255,255,0.04), 0 2px 8px rgba(0,0,0,0.2)",
  } as React.CSSProperties,
  headerIcon: {
    color: "var(--accent)", display: "flex", alignItems: "center",
  } as React.CSSProperties,
  headerName: {
    fontWeight: 700, fontSize: 15, color: "var(--text-primary)",
  } as React.CSSProperties,
  messagesArea: {
    flex: 1, overflowY: "auto" as const,
    display: "flex", flexDirection: "column" as const, paddingTop: 12,
  } as React.CSSProperties,
  loadMoreBtn: {
    margin: "8px auto", padding: "6px 20px",
    background: "var(--bg-elevated)", border: "1px solid var(--border)",
    borderRadius: 20, color: "var(--text-secondary)",
    cursor: "pointer", fontSize: 12, fontWeight: 600, display: "block",
    transition: "all 0.15s",
  } as React.CSSProperties,
  empty: {
    flex: 1, display: "flex", flexDirection: "column" as const,
    alignItems: "center", justifyContent: "center",
    color: "var(--text-muted)", fontSize: 14, gap: 8,
  } as React.CSSProperties,
  noChannel: {
    flex: 1, display: "flex", flexDirection: "column" as const,
    alignItems: "center", justifyContent: "center",
    color: "var(--text-muted)", gap: 12,
  } as React.CSSProperties,
  noChannelIcon: { fontSize: 48, opacity: 0.2 } as React.CSSProperties,
  dateDivider: {
    display: "flex", alignItems: "center", gap: 12, padding: "16px 20px 8px",
  } as React.CSSProperties,
  dateLine: { flex: 1, height: 1, background: "var(--border)" } as React.CSSProperties,
  dateText: {
    fontSize: 11, color: "var(--text-muted)", fontWeight: 600,
    whiteSpace: "nowrap" as const, letterSpacing: 0.3,
  } as React.CSSProperties,
};

/** Обогащаем сообщение username'ом автора (для MVP используем author_id) */
function enrichMessage(msg: MessageAPI, currentUserId: string, currentUsername: string): MessageAPI {
  if (!msg.author_username) {
    return {
      ...msg,
      author_username: msg.author_id === currentUserId ? currentUsername : `user_${msg.author_id.slice(0, 6)}`,
    };
  }
  return msg;
}

/** Конвертируем EncryptedPayload (клиент) → EncryptedPayloadAPI (бэкенд) */
function toApiPayload(p: EncryptedPayload): EncryptedPayloadAPI {
  return {
    ciphertext: p.ciphertext,
    iv: p.iv,
    key_id: p.keyId,
    tag: p.tag,
  };
}

/** Конвертируем EncryptedPayloadAPI (бэкенд) → EncryptedPayload (клиент) */
function fromApiPayload(p: EncryptedPayloadAPI): EncryptedPayload {
  return {
    ciphertext: p.ciphertext,
    iv: p.iv,
    keyId: p.key_id,
    tag: p.tag ?? "",
  };
}

export default function ChatView({
  channelId,
  channelName,
  currentUserId,
  currentUsername,
}: ChatViewProps) {
  const [messages, setMessages] = useState<MessageAPI[]>([]);
  const [loading, setLoading] = useState(false);
  const [hasMore, setHasMore] = useState(false);
  const [showScrollBtn, setShowScrollBtn] = useState(false);
  const [typingUsers, setTypingUsers] = useState<Set<string>>(new Set());
  const typingTimers = useRef<Map<string, ReturnType<typeof setTimeout>>>(new Map());
  const messagesEndRef = useRef<HTMLDivElement>(null);
  const messagesAreaRef = useRef<HTMLDivElement>(null);
  const channelIdRef = useRef<string | null>(null);
  const { theme } = useTheme();

  // Показываем кнопку scroll-to-bottom когда прокрутили вверх
  useEffect(() => {
    const el = messagesAreaRef.current;
    if (!el) return;
    const handler = () => {
      const distFromBottom = el.scrollHeight - el.scrollTop - el.clientHeight;
      setShowScrollBtn(distFromBottom > 200);
    };
    el.addEventListener("scroll", handler, { passive: true });
    return () => el.removeEventListener("scroll", handler);
  }, []);

  // Прокрутка вниз при новых сообщениях
  const scrollToBottom = useCallback(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: "smooth" });
  }, []);

  // Загрузка истории
  const loadHistory = useCallback(
    async (cId: string, before?: string) => {
      setLoading(true);
      try {
        const msgs = await api.messages.history(cId, before);
        const enriched = msgs.map((m) =>
          enrichMessage(m, currentUserId, currentUsername),
        );

        if (before) {
          setMessages((prev) => [...enriched, ...prev]);
        } else {
          setMessages(enriched);
          setHasMore(msgs.length === 50);
          setTimeout(scrollToBottom, 50);
        }

        setHasMore(msgs.length === 50);
      } catch (err) {
        console.error("Failed to load history:", err);
      } finally {
        setLoading(false);
      }
    },
    [currentUserId, currentUsername, scrollToBottom],
  );

  // Подписка на WS-события через глобальный сокет
  useEffect(() => {
    channelIdRef.current = channelId;
    const typingTimersMap = typingTimers.current;

    if (!channelId) {
      setMessages([]);
      return;
    }

    setMessages([]);
    setHasMore(false);
    loadHistory(channelId);

    // Убеждаемся что глобальный WS подключён, затем подписываемся на канал
    const subscribeToChannel = () => {
      sendWSEvent({ type: "subscribe", channel_id: channelId });
    };

    connectGlobalWS().then(subscribeToChannel).catch(() => {});

    // Если WS уже подключён — подписываемся сразу
    subscribeToChannel();

    // Обработчики сообщений
    const unsubNew = onWSEvent("message.new", (data) => {
      const msg = data.message as MessageAPI;
      if (!msg || msg.channel_id !== channelIdRef.current) return;
      const enriched = enrichMessage(msg, currentUserId, currentUsername);

      void (async () => {
        try {
          if (enriched.payload?.tag) {
            const { decrypt: dec, getOrCreateSessionKey: getKey } = await import("../crypto/index");
            const key = await getKey(enriched.channel_id);
            const payload = fromApiPayload(enriched.payload);
            const text = await dec(payload, key);
            setMessages((prev) => {
              if (prev.some((m) => m.id === enriched.id)) return prev;
              return [...prev, { ...enriched, decryptedContent: text }];
            });
          } else {
            setMessages((prev) => {
              if (prev.some((m) => m.id === enriched.id)) return prev;
              return [...prev, enriched];
            });
          }
        } catch {
          setMessages((prev) => {
            if (prev.some((m) => m.id === enriched.id)) return prev;
            return [...prev, enriched];
          });
        }
      })();

      setTimeout(scrollToBottom, 50);
      if (!document.hasFocus()) {
        const author = enriched.author_username ?? `user_${enriched.author_id.slice(0, 6)}`;
        showNotification(author, "New message").catch(() => {});
      }
    });

    const unsubEdited = onWSEvent("message.edited", (data) => {
      setMessages((prev) =>
        prev.map((m) => m.id === data.message_id ? { ...m, payload: data.payload as MessageAPI["payload"], edited: true } : m)
      );
    });

    const unsubDeleted = onWSEvent("message.deleted", (data) => {
      setMessages((prev) =>
        prev.map((m) => m.id === data.message_id ? { ...m, deleted: true } : m)
      );
    });

    // При переподключении WS — переподписываемся на канал
    const unsubReconnect = onWSEvent("ws.connected", () => {
      if (channelIdRef.current) {
        sendWSEvent({ type: "subscribe", channel_id: channelIdRef.current });
      }
    });

    // Typing indicator
    const unsubTyping = onWSEvent("typing", (data) => {
      if (data.channel_id !== channelIdRef.current) return;
      const userId = data.user_id as string;
      setTypingUsers(prev => new Set(prev).add(userId));
      // Убираем через 3 секунды
      const existing = typingTimersMap.get(userId);
      if (existing) clearTimeout(existing);
      const t = setTimeout(() => {
        setTypingUsers(prev => { const s = new Set(prev); s.delete(userId); return s; });
        typingTimersMap.delete(userId);
      }, 3000);
      typingTimersMap.set(userId, t);
    });

    return () => {
      if (channelIdRef.current) {
        sendWSEvent({ type: "unsubscribe", channel_id: channelId });
      }
      unsubNew();
      unsubEdited();
      unsubDeleted();
      unsubReconnect();
      unsubTyping();
      typingTimersMap.forEach((t) => clearTimeout(t));
      typingTimersMap.clear();
      setTypingUsers(new Set());
    };
  }, [channelId, currentUserId, currentUsername, scrollToBottom, loadHistory]);

  // Загрузка ещё (прокрутка вверх)
  const loadMore = useCallback(() => {
    if (!channelId || !hasMore || loading) return;
    const oldest = messages[0];
    if (oldest) {
      loadHistory(channelId, oldest.id);
    }
  }, [channelId, hasMore, loading, messages, loadHistory]);

  // Отправка сообщения (Требование 3.1 — шифрование перед отправкой)
  const handleSend = useCallback(
    async (text: string) => {
      if (!channelId) return;

      const sessionKey = await getOrCreateSessionKey(channelId);
      const encrypted = await encrypt(text, sessionKey);
      const apiPayload = toApiPayload(encrypted);

      const msg = await api.messages.send(channelId, apiPayload);
      trackMessage();
      const enriched = enrichMessage(
        { ...msg, decryptedContent: text },
        currentUserId,
        currentUsername,
      );

      setMessages((prev) => {
        if (prev.some((m) => m.id === enriched.id)) return prev;
        return [...prev, enriched];
      });
      setTimeout(scrollToBottom, 50);
    },
    [channelId, currentUserId, currentUsername, scrollToBottom],
  );

  // Редактирование сообщения (Требование 3.5)
  const handleEdit = useCallback(
    async (messageId: string, newText: string) => {
      if (!channelId) return;

      const sessionKey = await getOrCreateSessionKey(channelId);
      const encrypted = await encrypt(newText, sessionKey);
      const apiPayload = toApiPayload(encrypted);

      await api.messages.edit(messageId, apiPayload);

      setMessages((prev) =>
        prev.map((m) =>
          m.id === messageId
            ? { ...m, decryptedContent: newText, edited: true, payload: apiPayload }
            : m,
        ),
      );
    },
    [channelId],
  );

  // Удаление сообщения (Требование 3.6)
  const handleDelete = useCallback(async (messageId: string) => {
    await api.messages.delete(messageId);
    setMessages((prev) =>
      prev.map((m) => (m.id === messageId ? { ...m, deleted: true } : m)),
    );
  }, []);

  // Дешифрование входящих сообщений
  useEffect(() => {
    if (!channelId) return;

    let cancelled = false;

    const decryptMessages = async () => {
      try {
        const sessionKey = await getOrCreateSessionKey(channelId);
        const { decrypt } = await import("../crypto/index");

        // Берём актуальный список через функциональный setState
        setMessages((prev) => {
          const toDecrypt = prev.filter(
            (m) => m.decryptedContent === undefined && !m.deleted && m.payload?.tag,
          );
          if (toDecrypt.length === 0) return prev;

          // Запускаем асинхронное дешифрование и обновляем по мере готовности
          Promise.allSettled(
            toDecrypt.map(async (m) => {
              const payload = fromApiPayload(m.payload);
              const text = await decrypt(payload, sessionKey);
              return { id: m.id, text };
            }),
          ).then((results) => {
            if (cancelled) return;
            setMessages((current) =>
              current.map((m) => {
                const r = results.find(
                  (x) => x.status === "fulfilled" && x.value.id === m.id,
                );
                if (r && r.status === "fulfilled") {
                  return { ...m, decryptedContent: r.value.text };
                }
                return m;
              }),
            );
          });

          return prev; // Не меняем state синхронно
        });
      } catch {
        // Ключ недоступен
      }
    };

    decryptMessages();
    return () => { cancelled = true; };
  }, [channelId, messages.length]);

  if (!channelId) {
    return (
      <div style={styles.root}>
        <div style={styles.noChannel}>
          <div style={{ opacity: 0.5 }}>
            <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="var(--accent)" strokeWidth="1.5">
              <path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"/>
            </svg>
          </div>
          <div style={{
            fontSize: 17, fontWeight: 700, color: "#fff",
            textShadow: "0 1px 8px rgba(0,0,0,0.8)",
          }}>No channel selected</div>
          <div style={{
            fontSize: 13, color: "rgba(255,255,255,0.6)", textAlign: "center",
            textShadow: "0 1px 6px rgba(0,0,0,0.8)",
          }}>
            Pick a channel from the sidebar<br/>to start chatting
          </div>
        </div>
      </div>
    );
  }

  return (
    <div style={styles.root}>
      {/* Заголовок */}
      <div style={styles.header}>
        <span style={styles.headerIcon}>
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5">
            <path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"/>
          </svg>
        </span>
        <span style={styles.headerName}>{channelName}</span>
        <div style={{ flex: 1 }} />
        <div style={{ fontSize: 11, color: "var(--text-muted)", display: "flex", alignItems: "center", gap: 4 }}>
          <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
            <rect x="3" y="11" width="18" height="11" rx="2"/><path d="M7 11V7a5 5 0 0 1 10 0v4"/>
          </svg>
          End-to-end encrypted
        </div>
      </div>

      {/* Область сообщений */}
      <div style={{
        ...styles.messagesArea,
        position: "relative" as const,
      }} ref={messagesAreaRef}>
        {/* Chat background layer */}
        {theme.chatBackground && (
          <div style={{
            position: "absolute" as const, inset: 0, zIndex: 0, pointerEvents: "none" as const,
            opacity: (theme.chatBgOpacity ?? 100) / 100,
            backgroundImage: "var(--chat-bg-pattern, none), var(--chat-bg-image, none)",
            backgroundSize: "var(--chat-bg-size, auto), cover",
            backgroundPosition: "center",
            backgroundRepeat: "repeat, no-repeat",
          }} />
        )}
        <div style={{ position: "relative" as const, zIndex: 1, display: "flex", flexDirection: "column" as const, flex: 1 }}>
        {hasMore && (
          <button
            style={styles.loadMoreBtn}
            onClick={loadMore}
            disabled={loading}
          >
            {loading ? "Loading..." : "Load more"}
          </button>
        )}

        {messages.length === 0 && !loading && (
          <div style={styles.empty}>
            <svg width="40" height="40" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" style={{ opacity: 0.2 }}>
              <path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"/>
            </svg>
            <div>No messages yet. Say hello!</div>
          </div>
        )}

        {messages.map((msg, idx) => {
          const prev = messages[idx - 1];
          const grouped = !!prev && canGroup(prev, msg);
          const showDate = !prev ||
            new Date(msg.created_at).toDateString() !== new Date(prev.created_at).toDateString();
          return (
            <React.Fragment key={msg.id}>
              {showDate && (
                <div style={{
                  display: "flex", alignItems: "center", gap: 12,
                  padding: "16px 20px 8px", userSelect: "none" as const,
                }}>
                  <div style={{ flex: 1, height: 1, background: "var(--border)" }} />
                  <span style={{
                    fontSize: 11, color: "var(--text-muted)", fontWeight: 600,
                    padding: "2px 10px", borderRadius: 20,
                    background: "var(--bg-elevated)", border: "1px solid var(--border)",
                    whiteSpace: "nowrap" as const,
                  }}>
                    {formatDateDivider(msg.created_at)}
                  </span>
                  <div style={{ flex: 1, height: 1, background: "var(--border)" }} />
                </div>
              )}
              <MessageItem
                message={msg}
                currentUserId={currentUserId}
                onEdit={handleEdit}
                onDelete={handleDelete}
                grouped={grouped}
              />
            </React.Fragment>
          );
        })}

        <div ref={messagesEndRef} />
        </div>
      </div>

      {/* Scroll-to-bottom button */}
      {showScrollBtn && (
        <button
          onClick={scrollToBottom}
          style={{
            position: "absolute" as const, bottom: 80, right: 24, zIndex: 10,
            width: 40, height: 40, borderRadius: "50%", border: "none",
            background: "var(--accent)", color: "#fff", cursor: "pointer",
            display: "flex", alignItems: "center", justifyContent: "center",
            boxShadow: "0 4px 16px rgba(124,106,247,0.5)",
            animation: "fadeUp 0.2s ease",
            transition: "transform 0.15s",
          }}
          onMouseEnter={e => (e.currentTarget.style.transform = "scale(1.1)")}
          onMouseLeave={e => (e.currentTarget.style.transform = "scale(1)")}
          title="Scroll to bottom"
        >
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5">
            <polyline points="6 9 12 15 18 9"/>
          </svg>
        </button>
      )}

      {/* Typing indicator */}
      {typingUsers.size > 0 && (
        <div style={{
          padding: "4px 20px 6px",
          fontSize: 12, color: "var(--text-muted)",
          display: "flex", alignItems: "center", gap: 6,
          minHeight: 24,
        }}>
          <div style={{ display: "flex", gap: 3, alignItems: "center" }}>
            {[0,1,2].map(i => (
              <div key={i} style={{
                width: 5, height: 5, borderRadius: "50%",
                background: "var(--accent)",
                animation: `typingDot 1.2s ease-in-out ${i * 0.2}s infinite`,
              }} />
            ))}
          </div>
          <span>
            {typingUsers.size === 1
              ? "кто-то печатает..."
              : `${typingUsers.size} человека печатают...`}
          </span>
        </div>
      )}

      {/* Поле ввода */}
      <MessageInput
        channelName={channelName}
        onSend={handleSend}
        onTyping={() => {
          if (channelIdRef.current) {
            sendWSEvent({ type: "typing", channel_id: channelIdRef.current });
          }
        }}
        disabled={!channelId}
      />
    </div>
  );
}
