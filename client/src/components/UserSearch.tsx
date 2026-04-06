/**
 * UserSearch — поиск пользователей и список друзей
 */

import React, { useState, useEffect, useRef, useCallback } from "react";
import { api } from "../api/index";
import type { UserSearchResult, FriendAPI } from "../api/index";

interface Props {
  onClose: () => void;
}

const overlay: React.CSSProperties = {
  position: "fixed",
  inset: 0,
  background: "rgba(0,0,0,0.7)",
  display: "flex",
  alignItems: "center",
  justifyContent: "center",
  zIndex: 1000,
};

const modal: React.CSSProperties = {
  background: "var(--bg-secondary, #2f3136)",
  borderRadius: 8,
  padding: 24,
  width: 420,
  maxHeight: "80vh",
  color: "var(--text-primary, #dcddde)",
  display: "flex",
  flexDirection: "column",
  gap: 16,
  overflow: "hidden",
};

const searchInput: React.CSSProperties = {
  width: "100%",
  background: "var(--bg-tertiary, #202225)",
  border: "none",
  borderRadius: 4,
  padding: "10px 12px",
  color: "var(--text-primary, #dcddde)",
  fontSize: 14,
  outline: "none",
};

const sectionLabel: React.CSSProperties = {
  fontSize: 11,
  fontWeight: 700,
  textTransform: "uppercase",
  color: "var(--text-muted, #72767d)",
  marginBottom: 6,
};

const userRow: React.CSSProperties = {
  display: "flex",
  alignItems: "center",
  gap: 10,
  padding: "6px 0",
};

const avatarCircle: React.CSSProperties = {
  width: 36,
  height: 36,
  borderRadius: "50%",
  background: "var(--accent, #7289da)",
  display: "flex",
  alignItems: "center",
  justifyContent: "center",
  fontWeight: 700,
  fontSize: 14,
  color: "#fff",
  flexShrink: 0,
};

const STATUS_COLORS: Record<string, string> = {
  online: "#43b581",
  dnd: "#f04747",
  offline: "#747f8d",
};

function Avatar({ username }: { username: string }) {
  return <div style={avatarCircle}>{username.charAt(0).toUpperCase()}</div>;
}

export default function UserSearch({ onClose }: Props) {
  const [query, setQuery] = useState("");
  const [results, setResults] = useState<UserSearchResult[]>([]);
  const [friends, setFriends] = useState<FriendAPI[]>([]);
  const [sentRequests, setSentRequests] = useState<Set<string>>(new Set());
  const [loading, setLoading] = useState(false);
  const debounceRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  // Загружаем список друзей при открытии
  useEffect(() => {
    api.friends.list().then(setFriends).catch(() => {});
  }, []);

  const doSearch = useCallback((q: string) => {
    if (q.length < 2) {
      setResults([]);
      return;
    }
    setLoading(true);
    api.users
      .search(q)
      .then(setResults)
      .catch(() => setResults([]))
      .finally(() => setLoading(false));
  }, []);

  function handleQueryChange(e: React.ChangeEvent<HTMLInputElement>) {
    const q = e.target.value;
    setQuery(q);
    if (debounceRef.current) clearTimeout(debounceRef.current);
    debounceRef.current = setTimeout(() => doSearch(q), 300);
  }

  async function handleAddFriend(userId: string) {
    try {
      await api.friends.sendRequest(userId);
      setSentRequests((prev) => new Set(prev).add(userId));
    } catch {
      // ignore
    }
  }

  const onlineFriends = friends.filter((f) => f.status === "online" || f.status === "dnd");
  const offlineFriends = friends.filter((f) => f.status === "offline" || !f.status);

  return (
    <div style={overlay} onClick={onClose}>
      <div style={modal} onClick={(e) => e.stopPropagation()}>
        <div style={{ fontSize: 18, fontWeight: 700 }}>Поиск пользователей</div>

        <input
          autoFocus
          type="text"
          placeholder="Введите имя (минимум 2 символа)..."
          value={query}
          onChange={handleQueryChange}
          style={searchInput}
        />

        <div style={{ overflowY: "auto", flex: 1 }}>
          {/* Результаты поиска */}
          {query.length >= 2 && (
            <div style={{ marginBottom: 16 }}>
              <div style={sectionLabel}>Результаты поиска</div>
              {loading && <div style={{ color: "var(--text-muted, #72767d)", fontSize: 13 }}>Поиск...</div>}
              {!loading && results.length === 0 && (
                <div style={{ color: "var(--text-muted, #72767d)", fontSize: 13 }}>Ничего не найдено</div>
              )}
              {results.map((u) => (
                <div key={u.id} style={userRow}>
                  <Avatar username={u.username} />
                  <div style={{ flex: 1, fontSize: 14, fontWeight: 600 }}>{u.username}</div>
                  <button
                    disabled={sentRequests.has(u.id)}
                    onClick={() => handleAddFriend(u.id)}
                    style={{
                      padding: "5px 12px",
                      borderRadius: 4,
                      border: "none",
                      cursor: sentRequests.has(u.id) ? "default" : "pointer",
                      background: sentRequests.has(u.id) ? "var(--bg-tertiary, #202225)" : "var(--accent, #7289da)",
                      color: sentRequests.has(u.id) ? "var(--text-muted, #72767d)" : "#fff",
                      fontWeight: 600,
                      fontSize: 12,
                    }}
                  >
                    {sentRequests.has(u.id) ? "Отправлено" : "Добавить в друзья"}
                  </button>
                </div>
              ))}
            </div>
          )}

          {/* Список друзей */}
          {onlineFriends.length > 0 && (
            <div style={{ marginBottom: 12 }}>
              <div style={sectionLabel}>Онлайн — {onlineFriends.length}</div>
              {onlineFriends.map((f) => (
                <div key={f.id} style={userRow}>
                  <div style={{ position: "relative" }}>
                    <Avatar username={f.username} />
                    <span
                      style={{
                        position: "absolute",
                        bottom: 0,
                        right: 0,
                        width: 10,
                        height: 10,
                        borderRadius: "50%",
                        background: STATUS_COLORS[f.status ?? "offline"],
                        border: "2px solid var(--bg-secondary, #2f3136)",
                      }}
                    />
                  </div>
                  <div style={{ fontSize: 14, fontWeight: 600 }}>{f.username}</div>
                </div>
              ))}
            </div>
          )}

          {offlineFriends.length > 0 && (
            <div>
              <div style={sectionLabel}>Офлайн — {offlineFriends.length}</div>
              {offlineFriends.map((f) => (
                <div key={f.id} style={{ ...userRow, opacity: 0.6 }}>
                  <div style={{ position: "relative" }}>
                    <Avatar username={f.username} />
                    <span
                      style={{
                        position: "absolute",
                        bottom: 0,
                        right: 0,
                        width: 10,
                        height: 10,
                        borderRadius: "50%",
                        background: STATUS_COLORS.offline,
                        border: "2px solid var(--bg-secondary, #2f3136)",
                      }}
                    />
                  </div>
                  <div style={{ fontSize: 14, fontWeight: 600 }}>{f.username}</div>
                </div>
              ))}
            </div>
          )}

          {friends.length === 0 && query.length < 2 && (
            <div style={{ color: "var(--text-muted, #72767d)", fontSize: 13 }}>
              У вас пока нет друзей. Найдите пользователей через поиск.
            </div>
          )}
        </div>

        <div style={{ display: "flex", justifyContent: "flex-end" }}>
          <button
            onClick={onClose}
            style={{
              padding: "8px 16px",
              borderRadius: 4,
              border: "none",
              cursor: "pointer",
              background: "var(--bg-tertiary, #202225)",
              color: "var(--text-muted, #72767d)",
              fontWeight: 600,
              fontSize: 13,
            }}
          >
            Закрыть
          </button>
        </div>
      </div>
    </div>
  );
}
