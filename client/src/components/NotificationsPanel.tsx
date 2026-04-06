import { useState, useEffect, useCallback } from "react";
import { onWSEvent } from "../store/wsGlobal";

export interface Notification {
  id: string;
  type: "friend_request" | "friend_accepted" | "invite" | "guild_join";
  title: string;
  body: string;
  timestamp: number;
  read: boolean;
  meta?: Record<string, string>;
}

const STORAGE_KEY = "vibrora_notifications";

function loadNotifications(): Notification[] {
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    return raw ? JSON.parse(raw) : [];
  } catch { return []; }
}

function saveNotifications(list: Notification[]): void {
  // Храним последние 50
  localStorage.setItem(STORAGE_KEY, JSON.stringify(list.slice(0, 50)));
}

function makeId(): string {
  return Math.random().toString(36).slice(2);
}

interface Props {
  onClose: () => void;
  onAcceptFriend: (userId: string) => Promise<void>;
  onDeclineFriend: (userId: string) => Promise<void>;
  onJoinGuild: (code: string) => Promise<void>;
  anchorBottom?: number;
  anchorLeft?: number;
  anchorTop?: number;
  anchorRight?: number;
}

export default function NotificationsPanel({ onClose, onAcceptFriend, onDeclineFriend, onJoinGuild, anchorBottom, anchorLeft, anchorTop, anchorRight }: Props) {
  const [notifications, setNotifications] = useState<Notification[]>(loadNotifications);
  const [actionLoading, setActionLoading] = useState<string | null>(null);

  // Сохраняем при изменении
  useEffect(() => {
    saveNotifications(notifications);
  }, [notifications]);

  const addNotification = useCallback((n: Omit<Notification, "id" | "timestamp" | "read">) => {
    setNotifications(prev => {
      const next = [{ ...n, id: makeId(), timestamp: Date.now(), read: false }, ...prev];
      saveNotifications(next);
      return next;
    });
  }, []);

  // Подписка на WS-события
  useEffect(() => {
    const unsubFriendReq = onWSEvent("friend.request", (data) => {
      addNotification({
        type: "friend_request",
        title: "Запрос в друзья",
        body: `${data.from_username} хочет добавить вас в друзья`,
        meta: { userId: data.from_user_id as string, username: data.from_username as string },
      });
    });

    const unsubFriendAcc = onWSEvent("friend.accepted", (data) => {
      addNotification({
        type: "friend_accepted",
        title: "Запрос принят",
        body: `${data.from_username} принял ваш запрос в друзья`,
        meta: { userId: data.from_user_id as string, username: data.from_username as string },
      });
    });

    const unsubInvite = onWSEvent("invite.received", (data) => {
      addNotification({
        type: "invite",
        title: "Приглашение на сервер",
        body: `Вас пригласили на сервер`,
        meta: { code: data.invite_code as string, fromUserId: data.from_user_id as string },
      });
    });

    const unsubGuildJoin = onWSEvent("guild.member_joined", (data) => {
      addNotification({
        type: "guild_join",
        title: "Новый участник",
        body: `${data.username} вступил на сервер`,
        meta: { guildId: data.guild_id as string },
      });
    });

    return () => { unsubFriendReq(); unsubFriendAcc(); unsubInvite(); unsubGuildJoin(); };
  }, [addNotification]);

  function markAllRead() {
    setNotifications(prev => prev.map(n => ({ ...n, read: true })));
  }

  function clearAll() {
    setNotifications([]);
  }

  function markRead(id: string) {
    setNotifications(prev => prev.map(n => n.id === id ? { ...n, read: true } : n));
  }

  async function handleAccept(n: Notification) {
    if (!n.meta?.userId) return;
    setActionLoading(n.id + "_accept");
    try {
      await onAcceptFriend(n.meta.userId);
      setNotifications(prev => prev.map(x => x.id === n.id ? { ...x, read: true, meta: { ...x.meta!, done: "accepted" } } : x));
    } finally { setActionLoading(null); }
  }

  async function handleDecline(n: Notification) {
    if (!n.meta?.userId) return;
    setActionLoading(n.id + "_decline");
    try {
      await onDeclineFriend(n.meta.userId);
      setNotifications(prev => prev.map(x => x.id === n.id ? { ...x, read: true, meta: { ...x.meta!, done: "declined" } } : x));
    } finally { setActionLoading(null); }
  }

  async function handleJoinGuild(n: Notification) {
    if (!n.meta?.code) return;
    setActionLoading(n.id + "_join");
    try {
      await onJoinGuild(n.meta.code);
      setNotifications(prev => prev.map(x => x.id === n.id ? { ...x, read: true, meta: { ...x.meta!, done: "joined" } } : x));
    } finally { setActionLoading(null); }
  }

  const unread = notifications.filter(n => !n.read).length;

  return (
    <div style={{
      position: "fixed", inset: 0, zIndex: 1000,
    }} onClick={onClose}>
      <div style={{
        position: "fixed",
        ...(anchorTop    !== undefined ? { top: anchorTop }       : {}),
        ...(anchorBottom !== undefined ? { bottom: anchorBottom } : {}),
        ...(anchorRight  !== undefined ? { right: anchorRight }   : {}),
        ...(anchorLeft   !== undefined ? { left: anchorLeft }     : {}),
        width: 360, maxHeight: "70vh",
        background: "var(--bg-surface)", borderRadius: 16,
        border: "1px solid var(--border)", boxShadow: "var(--shadow-lg)",
        display: "flex", flexDirection: "column",
        animation: "fadeUp 0.15s ease", overflow: "hidden",
      }} onClick={e => e.stopPropagation()}>

        {/* Header */}
        <div style={{
          padding: "14px 16px", borderBottom: "1px solid var(--border)",
          display: "flex", alignItems: "center", justifyContent: "space-between",
          flexShrink: 0,
        }}>
          <div style={{ display: "flex", alignItems: "center", gap: 8 }}>
            <span style={{ fontSize: 15, fontWeight: 700, color: "var(--text-primary)" }}>
              Уведомления
            </span>
            {unread > 0 && (
              <span style={{
                background: "#f04f5e", color: "#fff",
                fontSize: 10, fontWeight: 700, padding: "1px 6px",
                borderRadius: 20, minWidth: 18, textAlign: "center",
              }}>{unread}</span>
            )}
          </div>
          <div style={{ display: "flex", gap: 6 }}>
            {unread > 0 && (
              <button onClick={markAllRead} style={{
                background: "none", border: "none", cursor: "pointer",
                color: "var(--text-muted)", fontSize: 11, fontWeight: 600,
                padding: "3px 8px", borderRadius: 6,
              }}>Прочитать все</button>
            )}
            {notifications.length > 0 && (
              <button onClick={clearAll} style={{
                background: "none", border: "none", cursor: "pointer",
                color: "var(--text-muted)", fontSize: 11, fontWeight: 600,
                padding: "3px 8px", borderRadius: 6,
              }}>Очистить</button>
            )}
          </div>
        </div>

        {/* List */}
        <div style={{ flex: 1, overflowY: "auto" }}>
          {notifications.length === 0 ? (
            <div style={{
              padding: "40px 20px", textAlign: "center",
              color: "var(--text-muted)", fontSize: 13,
            }}>
              <div style={{ fontSize: 32, marginBottom: 10 }}>🔔</div>
              <div style={{ fontWeight: 600, color: "var(--text-secondary)", marginBottom: 4 }}>Нет уведомлений</div>
              <div style={{ fontSize: 12 }}>Здесь будут запросы в друзья и приглашения</div>
            </div>
          ) : (
            notifications.map(n => (
              <NotificationItem
                key={n.id}
                n={n}
                actionLoading={actionLoading}
                onRead={() => markRead(n.id)}
                onAccept={() => handleAccept(n)}
                onDecline={() => handleDecline(n)}
                onJoin={() => handleJoinGuild(n)}
              />
            ))
          )}
        </div>
      </div>
    </div>
  );
}

// ── Notification item ─────────────────────────────────────────────────────────

function NotificationItem({ n, actionLoading, onRead, onAccept, onDecline, onJoin }: {
  n: Notification;
  actionLoading: string | null;
  onRead: () => void;
  onAccept: () => void;
  onDecline: () => void;
  onJoin: () => void;
}) {
  const isDone = !!n.meta?.done;
  const icons: Record<Notification["type"], string> = {
    friend_request: "👋",
    friend_accepted: "🎉",
    invite: "📨",
    guild_join: "👥",
  };
  const colors: Record<Notification["type"], string> = {
    friend_request: "#7c6af7",
    friend_accepted: "#3ecf8e",
    invite: "#f5a623",
    guild_join: "#06b6d4",
  };

  function formatTime(ts: number): string {
    const diff = Date.now() - ts;
    if (diff < 60000) return "только что";
    if (diff < 3600000) return `${Math.floor(diff / 60000)} мин назад`;
    if (diff < 86400000) return `${Math.floor(diff / 3600000)} ч назад`;
    return new Date(ts).toLocaleDateString("ru", { day: "numeric", month: "short" });
  }

  return (
    <div
      onClick={!n.read ? onRead : undefined}
      style={{
        padding: "12px 16px",
        background: n.read ? "transparent" : "rgba(124,106,247,0.05)",
        borderBottom: "1px solid var(--border)",
        cursor: n.read ? "default" : "pointer",
        transition: "background 0.15s",
      }}
      onMouseEnter={e => { if (!n.read) (e.currentTarget as HTMLDivElement).style.background = "rgba(255,255,255,0.03)"; }}
      onMouseLeave={e => { (e.currentTarget as HTMLDivElement).style.background = n.read ? "transparent" : "rgba(124,106,247,0.05)"; }}
    >
      <div style={{ display: "flex", gap: 10, alignItems: "flex-start" }}>
        {/* Icon */}
        <div style={{
          width: 36, height: 36, borderRadius: 10, flexShrink: 0,
          background: `${colors[n.type]}22`,
          border: `1px solid ${colors[n.type]}44`,
          display: "flex", alignItems: "center", justifyContent: "center",
          fontSize: 16,
        }}>
          {icons[n.type]}
        </div>

        {/* Content */}
        <div style={{ flex: 1, minWidth: 0 }}>
          <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between", marginBottom: 2 }}>
            <span style={{ fontSize: 13, fontWeight: 700, color: "var(--text-primary)" }}>{n.title}</span>
            <span style={{ fontSize: 10, color: "var(--text-muted)", flexShrink: 0, marginLeft: 8 }}>
              {formatTime(n.timestamp)}
            </span>
          </div>
          <div style={{ fontSize: 12, color: "var(--text-secondary)", lineHeight: 1.4, marginBottom: 8 }}>
            {n.body}
          </div>

          {/* Actions */}
          {!isDone && n.type === "friend_request" && (
            <div style={{ display: "flex", gap: 6 }}>
              <button
                disabled={!!actionLoading}
                onClick={e => { e.stopPropagation(); onAccept(); }}
                style={{
                  padding: "5px 12px", borderRadius: 7, border: "none",
                  background: "rgba(62,207,142,0.15)", color: "#3ecf8e",
                  cursor: "pointer", fontSize: 11, fontWeight: 700,
                  opacity: actionLoading ? 0.6 : 1,
                }}>✓ Принять</button>
              <button
                disabled={!!actionLoading}
                onClick={e => { e.stopPropagation(); onDecline(); }}
                style={{
                  padding: "5px 12px", borderRadius: 7, border: "none",
                  background: "rgba(240,79,94,0.12)", color: "#f04f5e",
                  cursor: "pointer", fontSize: 11, fontWeight: 700,
                  opacity: actionLoading ? 0.6 : 1,
                }}>✕ Отклонить</button>
            </div>
          )}

          {!isDone && n.type === "invite" && n.meta?.code && (
            <button
              disabled={!!actionLoading}
              onClick={e => { e.stopPropagation(); onJoin(); }}
              style={{
                padding: "5px 14px", borderRadius: 7, border: "none",
                background: "linear-gradient(135deg, #7c6af7, #a78bfa)",
                color: "#fff", cursor: "pointer", fontSize: 11, fontWeight: 700,
                opacity: actionLoading ? 0.6 : 1,
              }}>Войти на сервер</button>
          )}

          {isDone && (
            <span style={{ fontSize: 11, color: "var(--text-muted)", fontStyle: "italic" }}>
              {n.meta?.done === "accepted" && "✓ Принято"}
              {n.meta?.done === "declined" && "✕ Отклонено"}
              {n.meta?.done === "joined" && "✓ Вступил"}
            </span>
          )}

          {!n.read && (
            <div style={{
              position: "absolute" as const, right: 12, top: "50%", transform: "translateY(-50%)",
              width: 7, height: 7, borderRadius: "50%", background: "#7c6af7",
            }} />
          )}
        </div>
      </div>
    </div>
  );
}

// ── Хук для счётчика непрочитанных ───────────────────────────────────────────

export function useUnreadCount(): number {
  const [count, setCount] = useState(() =>
    loadNotifications().filter(n => !n.read).length
  );

  useEffect(() => {
    const update = () => {
      setCount(loadNotifications().filter(n => !n.read).length);
    };

    const unsubs = [
      onWSEvent("friend.request", update),
      onWSEvent("friend.accepted", update),
      onWSEvent("invite.received", update),
      onWSEvent("guild.member_joined", update),
    ];

    return () => unsubs.forEach(u => u());
  }, []);

  return count;
}
