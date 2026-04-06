import React, { useState, useEffect, useRef, useCallback } from "react";
import { api } from "../api/index";
import type { UserSearchResult, FriendUser } from "../api/index";

interface Props {
  onClose: () => void;
  onSendInvite?: (event: Record<string, unknown>) => void;
}

type Tab = "friends" | "search";

const COLORS = ["#7c6af7","#3ecf8e","#f5a623","#f04f5e","#06b6d4","#ec4899"];
const avatarColor = (name: string) => COLORS[name.charCodeAt(0) % COLORS.length];

function Avatar({ username, size = 38 }: { username: string; size?: number }) {
  return (
    <div style={{
      width: size, height: size, borderRadius: size * 0.28, flexShrink: 0,
      background: `linear-gradient(135deg, ${avatarColor(username)}, ${avatarColor(username)}99)`,
      display: "flex", alignItems: "center", justifyContent: "center",
      fontSize: size * 0.4, fontWeight: 700, color: "#fff",
      boxShadow: `0 2px 8px ${avatarColor(username)}44`,
    }}>
      {username.charAt(0).toUpperCase()}
    </div>
  );
}

export default function UserSearch({ onClose, onSendInvite }: Props) {
  const [tab, setTab] = useState<Tab>("friends");
  const [friends, setFriends] = useState<FriendUser[]>([]);
  const [query, setQuery] = useState("");
  const [results, setResults] = useState<UserSearchResult[]>([]);
  const [searchLoading, setSearchLoading] = useState(false);
  const [actionLoading, setActionLoading] = useState<string | null>(null);
  const [toast, setToast] = useState("");
  const [inviteModal, setInviteModal] = useState<{ userId: string; username: string } | null>(null);
  const [inviteCode, setInviteCode] = useState("");
  const debounceRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const loadFriends = useCallback(() => {
    api.friends.list().then(setFriends).catch(() => {});
  }, []);

  useEffect(() => { loadFriends(); }, [loadFriends]);

  const showToast = useCallback((msg: string) => {
    setToast(msg);
    setTimeout(() => setToast(""), 2500);
  }, []);

  function handleQueryChange(e: React.ChangeEvent<HTMLInputElement>) {
    const q = e.target.value;
    setQuery(q);
    if (debounceRef.current) clearTimeout(debounceRef.current);
    if (q.length < 2) { setResults([]); return; }
    debounceRef.current = setTimeout(() => {
      setSearchLoading(true);
      api.users.search(q)
        .then(setResults)
        .catch(() => setResults([]))
        .finally(() => setSearchLoading(false));
    }, 300);
  }

  async function handleSendRequest(userId: string, username: string) {
    setActionLoading(userId);
    try {
      await api.friends.sendRequest(userId);
      showToast(`Запрос отправлен ${username}`);
      loadFriends();
    } catch (err: any) {
      const code = err?.code;
      if (code === "already_friends") showToast("Уже в друзьях");
      else if (code === "request_pending") showToast("Запрос уже отправлен");
      else showToast("Ошибка отправки запроса");
    } finally { setActionLoading(null); }
  }

  async function handleAccept(userId: string, username: string) {
    setActionLoading(userId + "_accept");
    try {
      await api.friends.accept(userId);
      showToast(`${username} теперь в друзьях!`);
      loadFriends();
    } catch { showToast("Ошибка"); }
    finally { setActionLoading(null); }
  }

  function handleSendInviteToFriend(userId: string, username: string) {
    if (!inviteCode.trim()) {
      setInviteModal({ userId, username });
      return;
    }
    onSendInvite?.({ type: "invite.send", target_user_id: userId, invite_code: inviteCode.trim() });
    showToast(`Инвайт отправлен ${username}`);
    setInviteCode("");
    setInviteModal(null);
  }

  async function handleDecline(userId: string, username: string, isFriend: boolean) {
    setActionLoading(userId + "_decline");
    try {
      await api.friends.decline(userId);
      showToast(isFriend ? `${username} удалён из друзей` : "Запрос отклонён");
      loadFriends();
    } catch { showToast("Ошибка"); }
    finally { setActionLoading(null); }
  }

  const accepted = friends.filter(f => f.status === "accepted");
  const incoming = friends.filter(f => f.status === "pending" && f.direction === "incoming");
  const outgoing = friends.filter(f => f.status === "pending" && f.direction === "outgoing");

  // Фильтруем результаты поиска — убираем уже добавленных
  const friendIds = new Set(friends.map(f => f.id));
  const filteredResults = results.filter(r => !friendIds.has(r.id));

  return (
    <div style={{
      position: "fixed", inset: 0, background: "rgba(0,0,0,0.75)",
      display: "flex", alignItems: "center", justifyContent: "center",
      zIndex: 1000, backdropFilter: "blur(6px)",
    }} onClick={onClose}>
      <div style={{
        background: "var(--bg-surface)", borderRadius: 20,
        width: 460, maxHeight: "85vh", overflow: "hidden",
        border: "1px solid var(--border)", boxShadow: "var(--shadow-lg)",
        display: "flex", flexDirection: "column",
        animation: "fadeUp 0.2s ease",
      }} onClick={e => e.stopPropagation()}>

        {/* Header */}
        <div style={{ padding: "20px 24px 0", borderBottom: "1px solid var(--border)" }}>
          <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between", marginBottom: 16 }}>
            <div style={{ fontSize: 17, fontWeight: 700, color: "var(--text-primary)" }}>
              Друзья
              {accepted.length > 0 && (
                <span style={{
                  marginLeft: 8, fontSize: 12, fontWeight: 600,
                  background: "var(--accent-dim)", color: "var(--accent)",
                  padding: "2px 8px", borderRadius: 20,
                }}>{accepted.length}</span>
              )}
            </div>
            <button onClick={onClose} style={{
              background: "none", border: "none", cursor: "pointer",
              color: "var(--text-muted)", fontSize: 18, padding: "2px 6px", borderRadius: 6,
            }}>✕</button>
          </div>
          <div style={{ display: "flex" }}>
            {([
              { id: "friends", label: "Друзья" },
              { id: "search",  label: "Найти" },
            ] as { id: Tab; label: string }[]).map(t => (
              <button key={t.id} onClick={() => setTab(t.id)} style={{
                padding: "8px 16px", background: "none", border: "none", cursor: "pointer",
                fontSize: 13, fontWeight: 600,
                color: tab === t.id ? "var(--text-primary)" : "var(--text-muted)",
                borderBottom: `2px solid ${tab === t.id ? "var(--accent)" : "transparent"}`,
                marginBottom: -1, transition: "all 0.15s",
                position: "relative",
              }}>
                {t.label}
                {t.id === "friends" && incoming.length > 0 && (
                  <span style={{
                    position: "absolute", top: 4, right: 4,
                    width: 8, height: 8, borderRadius: "50%",
                    background: "#f04f5e",
                  }} />
                )}
              </button>
            ))}
          </div>
        </div>

        {/* Content */}
        <div style={{ flex: 1, overflowY: "auto", padding: "16px 24px" }}>

          {/* ── FRIENDS TAB ── */}
          {tab === "friends" && (
            <div style={{ display: "flex", flexDirection: "column", gap: 20 }}>

              {/* Incoming requests */}
              {incoming.length > 0 && (
                <div>
                  <div style={sectionLabel}>
                    Входящие запросы
                    <span style={{
                      marginLeft: 6, background: "#f04f5e22", color: "#f04f5e",
                      padding: "1px 7px", borderRadius: 20, fontSize: 10,
                    }}>{incoming.length}</span>
                  </div>
                  <div style={{ display: "flex", flexDirection: "column", gap: 6 }}>
                    {incoming.map(f => (
                      <div key={f.id} style={rowStyle}>
                        <Avatar username={f.username} />
                        <div style={{ flex: 1, minWidth: 0 }}>
                          <div style={{ fontSize: 14, fontWeight: 600, color: "var(--text-primary)" }}>{f.username}</div>
                          <div style={{ fontSize: 11, color: "var(--text-muted)" }}>хочет добавить вас в друзья</div>
                        </div>
                        <div style={{ display: "flex", gap: 6, flexShrink: 0 }}>
                          <button
                            disabled={!!actionLoading}
                            onClick={() => handleAccept(f.id, f.username)}
                            style={{
                              ...actionBtn,
                              background: "rgba(62,207,142,0.15)", color: "#3ecf8e",
                              opacity: actionLoading === f.id + "_accept" ? 0.5 : 1,
                            }}>✓</button>
                          <button
                            disabled={!!actionLoading}
                            onClick={() => handleDecline(f.id, f.username, false)}
                            style={{
                              ...actionBtn,
                              background: "rgba(240,79,94,0.12)", color: "#f04f5e",
                              opacity: actionLoading === f.id + "_decline" ? 0.5 : 1,
                            }}>✕</button>
                        </div>
                      </div>
                    ))}
                  </div>
                </div>
              )}

              {/* Accepted friends */}
              {accepted.length > 0 && (
                <div>
                  <div style={sectionLabel}>Друзья — {accepted.length}</div>
                  <div style={{ display: "flex", flexDirection: "column", gap: 4 }}>
                    {accepted.map(f => (
                      <div key={f.id} style={{ ...rowStyle, padding: "8px 10px", borderRadius: 10, background: "var(--bg-elevated)" }}
                        onMouseEnter={e => (e.currentTarget.style.background = "rgba(255,255,255,0.05)")}
                        onMouseLeave={e => (e.currentTarget.style.background = "var(--bg-elevated)")}>
                        <Avatar username={f.username} />
                        <div style={{ flex: 1, minWidth: 0 }}>
                          <div style={{ fontSize: 14, fontWeight: 600, color: "var(--text-primary)", overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap" }}>
                            {f.username}
                          </div>
                        </div>
                        <div style={{ display: "flex", gap: 6, alignItems: "center", flexShrink: 0 }}>
                          {onSendInvite && (
                            <button
                              onClick={() => setInviteModal({ userId: f.id, username: f.username })}
                              title="Пригласить на сервер"
                              style={{
                                padding: "5px 10px", borderRadius: 7, border: "none",
                                background: "rgba(62,207,142,0.12)", color: "#3ecf8e",
                                cursor: "pointer", fontSize: 11, fontWeight: 700,
                              }}>
                              + Инвайт
                            </button>
                          )}
                          <button
                            disabled={!!actionLoading}
                            onClick={() => handleDecline(f.id, f.username, true)}
                            title="Удалить из друзей"
                            style={{
                              background: "none", border: "none", cursor: "pointer",
                              color: "var(--text-muted)", padding: "4px 6px", borderRadius: 6,
                              fontSize: 13, opacity: actionLoading === f.id + "_decline" ? 0.5 : 1,
                              transition: "color 0.15s",
                            }}
                            onMouseEnter={e => (e.currentTarget.style.color = "#f04f5e")}
                            onMouseLeave={e => (e.currentTarget.style.color = "var(--text-muted)")}>
                            ✕
                          </button>
                        </div>
                      </div>
                    ))}
                  </div>
                </div>
              )}

              {/* Outgoing requests */}
              {outgoing.length > 0 && (
                <div>
                  <div style={sectionLabel}>Исходящие запросы</div>
                  <div style={{ display: "flex", flexDirection: "column", gap: 4 }}>
                    {outgoing.map(f => (
                      <div key={f.id} style={rowStyle}>
                        <Avatar username={f.username} />
                        <div style={{ flex: 1, minWidth: 0 }}>
                          <div style={{ fontSize: 14, fontWeight: 600, color: "var(--text-primary)" }}>{f.username}</div>
                          <div style={{ fontSize: 11, color: "var(--text-muted)" }}>ожидает ответа</div>
                        </div>
                        <button
                          disabled={!!actionLoading}
                          onClick={() => handleDecline(f.id, f.username, false)}
                          title="Отменить запрос"
                          style={{
                            ...actionBtn,
                            background: "rgba(255,255,255,0.06)", color: "var(--text-muted)",
                            opacity: actionLoading === f.id + "_decline" ? 0.5 : 1,
                          }}>Отмена</button>
                      </div>
                    ))}
                  </div>
                </div>
              )}

              {friends.length === 0 && (
                <div style={{
                  textAlign: "center", padding: "40px 20px",
                  color: "var(--text-muted)", fontSize: 13,
                }}>
                  <div style={{ fontSize: 36, marginBottom: 12 }}>👥</div>
                  <div style={{ fontWeight: 600, marginBottom: 6, color: "var(--text-secondary)" }}>Пока нет друзей</div>
                  <div>Найди пользователей через вкладку "Найти"</div>
                </div>
              )}
            </div>
          )}

          {/* ── SEARCH TAB ── */}
          {tab === "search" && (
            <div style={{ display: "flex", flexDirection: "column", gap: 12 }}>
              {/* Search input */}
              <div style={{
                display: "flex", alignItems: "center", gap: 8,
                background: "var(--bg-input)", borderRadius: 10,
                padding: "9px 12px", border: "1px solid var(--border)",
              }}>
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="var(--text-muted)" strokeWidth="2">
                  <circle cx="11" cy="11" r="8"/><line x1="21" y1="21" x2="16.65" y2="16.65"/>
                </svg>
                <input autoFocus value={query} onChange={handleQueryChange}
                  placeholder="Поиск по имени (мин. 2 символа)..."
                  style={{
                    flex: 1, background: "none", border: "none", outline: "none",
                    color: "var(--text-primary)", fontSize: 14, fontFamily: "inherit",
                  }}
                />
                {query && (
                  <button onClick={() => { setQuery(""); setResults([]); }} style={{
                    background: "none", border: "none", cursor: "pointer",
                    color: "var(--text-muted)", fontSize: 14, padding: 0,
                  }}>✕</button>
                )}
              </div>

              {query.length >= 2 && (
                <>
                  {searchLoading && (
                    <div style={{ textAlign: "center", color: "var(--text-muted)", padding: 24, fontSize: 13 }}>
                      Поиск...
                    </div>
                  )}
                  {!searchLoading && filteredResults.length === 0 && (
                    <div style={{ textAlign: "center", color: "var(--text-muted)", padding: 24, fontSize: 13 }}>
                      Никого не найдено
                    </div>
                  )}
                  {!searchLoading && filteredResults.length > 0 && (
                    <div style={{ display: "flex", flexDirection: "column", gap: 4 }}>
                      {filteredResults.map(u => {
                        const isLoading = actionLoading === u.id;
                        return (
                          <div key={u.id} style={{ ...rowStyle, padding: "10px 12px", borderRadius: 10, background: "var(--bg-elevated)" }}>
                            <Avatar username={u.username} />
                            <div style={{ flex: 1, minWidth: 0 }}>
                              <div style={{ fontSize: 14, fontWeight: 600, color: "var(--text-primary)", overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap" }}>
                                {u.username}
                              </div>
                            </div>
                            <button
                              disabled={isLoading}
                              onClick={() => handleSendRequest(u.id, u.username)}
                              style={{
                                padding: "7px 14px", borderRadius: 8, border: "none",
                                background: "linear-gradient(135deg, var(--accent), var(--accent-hover))",
                                color: "#fff", cursor: isLoading ? "default" : "pointer",
                                fontSize: 12, fontWeight: 700, flexShrink: 0,
                                opacity: isLoading ? 0.6 : 1,
                                boxShadow: "0 2px 8px rgba(124,106,247,0.3)",
                                transition: "opacity 0.15s",
                              }}>
                              {isLoading ? "..." : "+ Добавить"}
                            </button>
                          </div>
                        );
                      })}
                    </div>
                  )}
                </>
              )}

              {query.length < 2 && (
                <div style={{
                  textAlign: "center", padding: "32px 20px",
                  color: "var(--text-muted)", fontSize: 13,
                }}>
                  <div style={{ fontSize: 32, marginBottom: 10 }}>🔍</div>
                  Введи имя пользователя для поиска
                </div>
              )}
            </div>
          )}
        </div>
      </div>

      {/* Invite code modal */}
      {inviteModal && (
        <div style={{
          position: "fixed", inset: 0, background: "rgba(0,0,0,0.6)",
          display: "flex", alignItems: "center", justifyContent: "center", zIndex: 2000,
        }} onClick={() => setInviteModal(null)}>
          <div style={{
            background: "var(--bg-surface)", borderRadius: 16, padding: "24px",
            width: 360, border: "1px solid var(--border)", boxShadow: "var(--shadow-lg)",
            animation: "fadeUp 0.15s ease",
          }} onClick={e => e.stopPropagation()}>
            <div style={{ fontSize: 15, fontWeight: 700, color: "var(--text-primary)", marginBottom: 6 }}>
              Пригласить {inviteModal.username}
            </div>
            <div style={{ fontSize: 12, color: "var(--text-muted)", marginBottom: 16 }}>
              Вставь инвайт-код сервера — он придёт другу как уведомление
            </div>
            <input
              autoFocus
              value={inviteCode}
              onChange={e => setInviteCode(e.target.value)}
              onKeyDown={e => { if (e.key === "Enter") handleSendInviteToFriend(inviteModal.userId, inviteModal.username); }}
              placeholder="Код инвайта (напр. abc123)"
              style={{
                width: "100%", padding: "10px 12px", marginBottom: 14,
                background: "var(--bg-input)", border: "1px solid var(--border)",
                borderRadius: 8, color: "var(--text-primary)", fontSize: 13,
                outline: "none", fontFamily: "inherit", boxSizing: "border-box" as const,
              }}
            />
            <div style={{ display: "flex", gap: 8 }}>
              <button onClick={() => setInviteModal(null)} style={{
                flex: 1, padding: "9px", background: "var(--bg-elevated)",
                border: "1px solid var(--border)", borderRadius: 8,
                color: "var(--text-secondary)", cursor: "pointer", fontSize: 13, fontWeight: 600,
              }}>Отмена</button>
              <button
                disabled={!inviteCode.trim()}
                onClick={() => handleSendInviteToFriend(inviteModal.userId, inviteModal.username)}
                style={{
                  flex: 2, padding: "9px",
                  background: inviteCode.trim() ? "linear-gradient(135deg, #3ecf8e, #06b6d4)" : "var(--bg-elevated)",
                  border: "none", borderRadius: 8,
                  color: inviteCode.trim() ? "#fff" : "var(--text-muted)",
                  cursor: inviteCode.trim() ? "pointer" : "default",
                  fontSize: 13, fontWeight: 700,
                }}>Отправить</button>
            </div>
          </div>
        </div>
      )}

      {/* Toast */}
      {toast && (
        <div style={{
          position: "fixed", bottom: 24, left: "50%", transform: "translateX(-50%)",
          background: "var(--bg-elevated)", border: "1px solid var(--border)",
          borderRadius: 10, padding: "10px 20px", fontSize: 13, fontWeight: 600,
          color: "var(--text-primary)", boxShadow: "var(--shadow-md)",
          animation: "fadeUp 0.2s ease", zIndex: 9999, whiteSpace: "nowrap",
        }}>{toast}</div>
      )}
    </div>
  );
}

const sectionLabel: React.CSSProperties = {
  fontSize: 11, fontWeight: 700, color: "var(--text-muted)",
  textTransform: "uppercase", letterSpacing: 0.8, marginBottom: 10,
  display: "flex", alignItems: "center", gap: 6,
};

const rowStyle: React.CSSProperties = {
  display: "flex", alignItems: "center", gap: 12,
  padding: "6px 0", transition: "background 0.15s",
};

const actionBtn: React.CSSProperties = {
  padding: "6px 12px", borderRadius: 8, border: "none",
  cursor: "pointer", fontSize: 13, fontWeight: 700, flexShrink: 0,
  transition: "opacity 0.15s",
};
