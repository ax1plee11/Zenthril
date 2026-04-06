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
import MessageItem from "./MessageItem";
import MessageInput from "./MessageInput";
import { showNotification } from "../notifications/index";

const WS_URL = "ws://localhost:8080/ws";
const TOKEN_KEY = "veltrix_token";

interface ChatViewProps {
  channelId: string | null;
  channelName: string;
  currentUserId: string;
  currentUsername: string;
}

const styles = {
  root: {
    flex: 1,
    display: "flex",
    flexDirection: "column" as const,
    background: "#36393f",
    minWidth: 0,
  } as React.CSSProperties,

  header: {
    padding: "0 16px",
    height: 48,
    borderBottom: "1px solid #202225",
    display: "flex",
    alignItems: "center",
    gap: 8,
    flexShrink: 0,
  } as React.CSSProperties,

  headerIcon: {
    color: "#72767d",
    fontSize: 18,
    fontWeight: 700,
  } as React.CSSProperties,

  headerName: {
    fontWeight: 700,
    fontSize: 15,
    color: "#fff",
  } as React.CSSProperties,

  messagesArea: {
    flex: 1,
    overflowY: "auto" as const,
    display: "flex",
    flexDirection: "column" as const,
    paddingTop: 16,
  } as React.CSSProperties,

  loadMoreBtn: {
    margin: "8px auto",
    padding: "6px 16px",
    background: "#4f545c",
    border: "none",
    borderRadius: 4,
    color: "#dcddde",
    cursor: "pointer",
    fontSize: 13,
    display: "block",
  } as React.CSSProperties,

  empty: {
    flex: 1,
    display: "flex",
    alignItems: "center",
    justifyContent: "center",
    color: "#72767d",
    fontSize: 15,
  } as React.CSSProperties,

  noChannel: {
    flex: 1,
    display: "flex",
    flexDirection: "column" as const,
    alignItems: "center",
    justifyContent: "center",
    color: "#72767d",
    gap: 8,
  } as React.CSSProperties,

  noChannelIcon: {
    fontSize: 48,
    opacity: 0.3,
  } as React.CSSProperties,

  dateDivider: {
    display: "flex",
    alignItems: "center",
    gap: 8,
    padding: "16px 16px 8px",
  } as React.CSSProperties,

  dateLine: {
    flex: 1,
    height: 1,
    background: "#3f4147",
  } as React.CSSProperties,

  dateText: {
    fontSize: 12,
    color: "#72767d",
    fontWeight: 600,
    whiteSpace: "nowrap" as const,
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
  const messagesEndRef = useRef<HTMLDivElement>(null);
  const messagesAreaRef = useRef<HTMLDivElement>(null);
  const wsRef = useRef<WebSocket | null>(null);
  const channelIdRef = useRef<string | null>(null);

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

  // Подключение WebSocket
  const connectWS = useCallback(
    (cId: string) => {
      const token = localStorage.getItem(TOKEN_KEY);
      if (!token) return;

      // Закрываем предыдущее соединение
      if (wsRef.current) {
        wsRef.current.close();
        wsRef.current = null;
      }

      const ws = new WebSocket(`${WS_URL}?token=${token}`);
      wsRef.current = ws;

      ws.onopen = () => {
        // Подписываемся на канал (Требование 3.2)
        ws.send(JSON.stringify({ type: "subscribe", channel_id: cId }));
      };

      ws.onmessage = (event: MessageEvent) => {
        try {
          const data = JSON.parse(event.data as string);

          if (data.type === "message.new" && data.message) {
            const msg = data.message as MessageAPI;
            if (msg.channel_id === channelIdRef.current) {
              const enriched = enrichMessage(msg, currentUserId, currentUsername);
              setMessages((prev) => {
                // Избегаем дублей
                if (prev.some((m) => m.id === enriched.id)) return prev;
                return [...prev, enriched];
              });
              setTimeout(scrollToBottom, 50);

              // Уведомление если окно не в фокусе (Требование 10.8)
              if (!document.hasFocus()) {
                const author = enriched.author_username ?? `user_${enriched.author_id.slice(0, 6)}`;
                showNotification(author, enriched.decryptedContent ?? "Новое сообщение").catch(() => {});
              }
            }
          }

          if (data.type === "message.edited") {
            setMessages((prev) =>
              prev.map((m) =>
                m.id === data.message_id
                  ? { ...m, payload: data.payload, edited: true }
                  : m,
              ),
            );
          }

          if (data.type === "message.deleted") {
            setMessages((prev) =>
              prev.map((m) =>
                m.id === data.message_id ? { ...m, deleted: true } : m,
              ),
            );
          }
        } catch {
          // Игнорируем невалидные сообщения
        }
      };

      ws.onerror = () => {
        console.warn("[ChatView] WebSocket error");
      };

      ws.onclose = () => {
        // Переподключение через 3 сек если канал ещё активен
        setTimeout(() => {
          if (channelIdRef.current === cId) {
            connectWS(cId);
          }
        }, 3000);
      };
    },
    [currentUserId, currentUsername, scrollToBottom],
  );

  // При смене канала
  useEffect(() => {
    channelIdRef.current = channelId;

    if (!channelId) {
      setMessages([]);
      return;
    }

    setMessages([]);
    setHasMore(false);
    loadHistory(channelId);
    connectWS(channelId);

    return () => {
      if (wsRef.current) {
        wsRef.current.close();
        wsRef.current = null;
      }
    };
  }, [channelId]); // eslint-disable-line react-hooks/exhaustive-deps

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

  // Дешифрование входящих сообщений (для MVP — показываем ciphertext если нет ключа)
  useEffect(() => {
    if (!channelId) return;

    const decryptMessages = async () => {
      try {
        const sessionKey = await getOrCreateSessionKey(channelId);
        setMessages((prev) =>
          prev.map((m) => {
            if (m.decryptedContent !== undefined) return m;
            // Пробуем дешифровать
            const payload = fromApiPayload(m.payload);
            // Для MVP: если нет tag — показываем ciphertext
            if (!payload.tag) return m;
            return m; // Дешифрование асинхронное, делаем ниже
          }),
        );

        // Асинхронное дешифрование
        const { decrypt } = await import("../crypto/index");
        setMessages((prev) =>
          prev.map((m) => {
            if (m.decryptedContent !== undefined) return m;
            return m; // Будет дешифровано в следующем эффекте
          }),
        );

        // Дешифруем все сообщения без decryptedContent
        const toDecrypt = messages.filter(
          (m) => m.decryptedContent === undefined && !m.deleted && m.payload.tag,
        );

        if (toDecrypt.length === 0) return;

        const decrypted = await Promise.allSettled(
          toDecrypt.map(async (m) => {
            const payload = fromApiPayload(m.payload);
            const text = await decrypt(payload, sessionKey);
            return { id: m.id, text };
          }),
        );

        setMessages((prev) =>
          prev.map((m) => {
            const result = decrypted.find(
              (r) => r.status === "fulfilled" && r.value.id === m.id,
            );
            if (result && result.status === "fulfilled") {
              return { ...m, decryptedContent: result.value.text };
            }
            return m;
          }),
        );
      } catch {
        // Ключ недоступен — показываем ciphertext
      }
    };

    decryptMessages();
  }, [channelId, messages.length]); // eslint-disable-line react-hooks/exhaustive-deps

  if (!channelId) {
    return (
      <div style={styles.root}>
        <div style={styles.noChannel}>
          <div style={styles.noChannelIcon}>💬</div>
          <div>Выберите канал для начала общения</div>
        </div>
      </div>
    );
  }

  return (
    <div style={styles.root}>
      {/* Заголовок */}
      <div style={styles.header}>
        <span style={styles.headerIcon}>#</span>
        <span style={styles.headerName}>{channelName}</span>
      </div>

      {/* Область сообщений */}
      <div style={styles.messagesArea} ref={messagesAreaRef}>
        {hasMore && (
          <button
            style={styles.loadMoreBtn}
            onClick={loadMore}
            disabled={loading}
          >
            {loading ? "Загрузка..." : "Загрузить ещё"}
          </button>
        )}

        {messages.length === 0 && !loading && (
          <div style={styles.empty}>
            Нет сообщений. Напишите первым!
          </div>
        )}

        {messages.map((msg) => (
          <MessageItem
            key={msg.id}
            message={msg}
            currentUserId={currentUserId}
            onEdit={handleEdit}
            onDelete={handleDelete}
          />
        ))}

        <div ref={messagesEndRef} />
      </div>

      {/* Поле ввода */}
      <MessageInput
        channelName={channelName}
        onSend={handleSend}
        disabled={!channelId}
      />
    </div>
  );
}
