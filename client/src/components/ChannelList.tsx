import { useState, useEffect, useCallback } from "react";
import type { GuildAPI, ChannelAPI } from "../api/index";
import { api } from "../api/index";
import { onWSEvent } from "../store/wsGlobal";

interface ChannelListProps {
  guild: GuildAPI | null;
  channels: ChannelAPI[];
  selectedChannelId: string | null;
  onSelect: (id: string) => void;
  currentUserId: string;
}

function ChannelItem({ ch, selected, onClick, unread = 0 }: { ch: ChannelAPI; selected: boolean; onClick: () => void; unread?: number }) {
  const [hovered, setHovered] = useState(false);
  const isVoice = ch.type === "voice";

  return (
    <div
      style={{
        display: "flex", alignItems: "center", gap: 8,
        padding: "7px 10px", margin: "1px 6px", borderRadius: "var(--radius-sm)",
        cursor: "pointer",
        background: selected ? "rgba(124,106,247,0.15)" : hovered ? "rgba(255,255,255,0.04)" : "transparent",
        color: selected ? "var(--text-primary)" : unread > 0 ? "var(--text-primary)" : hovered ? "var(--text-secondary)" : "var(--text-muted)",
        fontSize: 14, fontWeight: selected || unread > 0 ? 600 : 400,
        transition: "all 0.15s", userSelect: "none" as const,
        borderLeft: selected ? "2px solid var(--accent)" : "2px solid transparent",
      }}
      onClick={onClick}
      onMouseEnter={() => setHovered(true)}
      onMouseLeave={() => setHovered(false)}
    >
      <span style={{ fontSize: 13, opacity: selected ? 1 : 0.7, flexShrink: 0 }}>
        {isVoice ? (
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
            <path d="M12 1a3 3 0 0 0-3 3v8a3 3 0 0 0 6 0V4a3 3 0 0 0-3-3z"/>
            <path d="M19 10v2a7 7 0 0 1-14 0v-2"/><line x1="12" y1="19" x2="12" y2="23"/>
            <line x1="8" y1="23" x2="16" y2="23"/>
          </svg>
        ) : (
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5">
            <path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"/>
          </svg>
        )}
      </span>
      <span style={{ overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap" as const, flex: 1 }}>
        {ch.name}
      </span>
      {unread > 0 && (
        <span style={{
          minWidth: 18, height: 18, borderRadius: 9,
          background: "var(--accent)", color: "#fff",
          fontSize: 10, fontWeight: 800,
          display: "flex", alignItems: "center", justifyContent: "center",
          padding: "0 4px", flexShrink: 0,
        }}>
          {unread > 99 ? "99+" : unread}
        </span>
      )}
    </div>
  );
}

export default function ChannelList({ guild, channels, selectedChannelId, onSelect, currentUserId }: ChannelListProps) {
  const textChannels  = channels.filter(c => c.type === "text");
  const voiceChannels = channels.filter(c => c.type === "voice");
  const [showManage, setShowManage] = useState(false);
  const isOwner = guild?.owner_id === currentUserId;
  // Счётчики непрочитанных: channelId → count
  const [unread, setUnread] = useState<Record<string, number>>({});

  // Слушаем новые сообщения — если канал не выбран, увеличиваем счётчик
  useEffect(() => {
    const unsub = onWSEvent("message.new", (data) => {
      const msg = data.message as { channel_id: string };
      if (!msg?.channel_id) return;
      if (msg.channel_id === selectedChannelId) return; // уже читаем
      setUnread(prev => ({ ...prev, [msg.channel_id]: (prev[msg.channel_id] ?? 0) + 1 }));
    });
    return unsub;
  }, [selectedChannelId]);

  // Сбрасываем счётчик при выборе канала
  const handleSelect = (id: string) => {
    setUnread(prev => { const n = { ...prev }; delete n[id]; return n; });
    onSelect(id);
  };

  return (
    <div style={{
      width: 232, background: "transparent", display: "flex",
      flexDirection: "column" as const, flexShrink: 0,
      borderRight: "1px solid var(--border)",
    }}>
      {/* Header */}
      <div style={{
        padding: "0 12px 0 16px", height: 52, display: "flex", alignItems: "center",
        borderBottom: "1px solid var(--border)", flexShrink: 0, gap: 8,
      }}>
        {guild ? (
          <>
            <div style={{ display: "flex", alignItems: "center", gap: 8, minWidth: 0, flex: 1 }}>
              <div style={{
                width: 28, height: 28, borderRadius: 8, flexShrink: 0,
                background: "linear-gradient(135deg, var(--accent), #a78bfa)",
                display: "flex", alignItems: "center", justifyContent: "center",
                fontSize: 12, fontWeight: 700, color: "#fff",
              }}>
                {guild.name.charAt(0).toUpperCase()}
              </div>
              <span style={{
                fontWeight: 700, fontSize: 14, color: "var(--text-primary)",
                overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap" as const,
              }}>
                {guild.name}
              </span>
            </div>
            {/* Settings button — visible to owner */}
            {isOwner && (
              <button onClick={() => setShowManage(true)} title="Управление сервером" style={{
                background: "none", border: "none", cursor: "pointer",
                color: "var(--text-muted)", padding: 4, borderRadius: 6, flexShrink: 0,
                display: "flex", alignItems: "center", justifyContent: "center",
                transition: "color 0.15s",
              }}
              onMouseEnter={e => (e.currentTarget.style.color = "var(--text-primary)")}
              onMouseLeave={e => (e.currentTarget.style.color = "var(--text-muted)")}>
                <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                  <circle cx="12" cy="12" r="3"/>
                  <path d="M19.07 4.93a10 10 0 0 1 0 14.14M4.93 4.93a10 10 0 0 0 0 14.14"/>
                </svg>
              </button>
            )}
          </>
        ) : (
          <span style={{ color: "var(--text-muted)", fontSize: 13 }}>Select a server</span>
        )}
      </div>

      {/* Channels */}
      <div style={{ flex: 1, overflowY: "auto" as const, paddingTop: 8 }}>
        {!guild && <NoServerSelected />}

        {textChannels.length > 0 && (
          <>
            <div style={{
              padding: "12px 16px 4px", fontSize: 10, fontWeight: 700,
              color: "var(--text-muted)", textTransform: "uppercase" as const, letterSpacing: 1,
              display: "flex", alignItems: "center", gap: 6,
            }}>
              <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5">
                <path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"/>
              </svg>
              Text Channels
            </div>
            {textChannels.map(ch => (
              <ChannelItem key={ch.id} ch={ch} selected={ch.id === selectedChannelId} onClick={() => handleSelect(ch.id)} unread={unread[ch.id] ?? 0} />
            ))}
          </>
        )}

        {voiceChannels.length > 0 && (
          <>
            <div style={{
              padding: "12px 16px 4px", fontSize: 10, fontWeight: 700,
              color: "var(--text-muted)", textTransform: "uppercase" as const, letterSpacing: 1,
              display: "flex", alignItems: "center", gap: 6, marginTop: 8,
            }}>
              <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5">
                <path d="M12 1a3 3 0 0 0-3 3v8a3 3 0 0 0 6 0V4a3 3 0 0 0-3-3z"/>
                <path d="M19 10v2a7 7 0 0 1-14 0v-2"/>
              </svg>
              Voice Channels
            </div>
            {voiceChannels.map(ch => (
              <ChannelItem key={ch.id} ch={ch} selected={ch.id === selectedChannelId} onClick={() => handleSelect(ch.id)} unread={0} />
            ))}
          </>
        )}

        {guild && channels.length === 0 && <NoChannelsYet />}
      </div>

      {/* Member management modal */}
      {showManage && guild && (
        <MemberManageModal
          guild={guild}
          currentUserId={currentUserId}
          onClose={() => setShowManage(false)}
        />
      )}
    </div>
  );
}

// ── Member management modal ───────────────────────────────────────────────────

interface Member { id: string; username: string; }

function MemberManageModal({ guild, currentUserId, onClose }: {
  guild: GuildAPI; currentUserId: string; onClose: () => void;
}) {
  const [members, setMembers] = useState<Member[]>([]);
  const [loading, setLoading] = useState(true);
  const [loadError, setLoadError] = useState("");
  const [search, setSearch] = useState("");
  const [actionLoading, setActionLoading] = useState<string | null>(null);
  const [toast, setToast] = useState("");
  const [tab, setTab] = useState<"members" | "invite">("members");
  const [inviteCode, setInviteCode] = useState<string | null>(null);
  const [inviteLoading, setInviteLoading] = useState(false);

  // Загружаем участников конкретного сервера
  useEffect(() => {
    setLoading(true);
    setLoadError("");
    api.guilds.members(guild.id)
      .then(res => { setMembers(res); setLoadError(""); })
      .catch((err) => {
        console.error("[Members] load failed:", err);
        setLoadError(err?.message ?? "Не удалось загрузить участников");
      })
      .finally(() => setLoading(false));

    // Слушаем WS — новый участник вступил
    const unsub = onWSEvent("guild.member_joined", (data) => {
      if (data.guild_id !== guild.id) return;
      const newMember = { id: data.user_id as string, username: data.username as string };
      setMembers(prev => prev.some(m => m.id === newMember.id) ? prev : [...prev, newMember]);
    });
    return unsub;
  }, [guild.id]);

  const showToast = useCallback((msg: string) => {
    setToast(msg);
    setTimeout(() => setToast(""), 2500);
  }, []);

  async function handleKick(userId: string, username: string) {
    if (!confirm(`Кикнуть ${username} с сервера?`)) return;
    setActionLoading(userId + "_kick");
    try {
      await api.guilds.kickMember(guild.id, userId);
      setMembers(prev => prev.filter(m => m.id !== userId));
      showToast(`${username} кикнут`);
    } catch { showToast("Ошибка"); }
    finally { setActionLoading(null); }
  }

  async function handleBan(userId: string, username: string) {
    if (!confirm(`Забанить ${username} на этом сервере?`)) return;
    setActionLoading(userId + "_ban");
    try {
      await api.guilds.banMember(guild.id, userId);
      setMembers(prev => prev.filter(m => m.id !== userId));
      showToast(`${username} забанен на сервере`);
    } catch { showToast("Ошибка"); }
    finally { setActionLoading(null); }
  }

  async function handleGlobalBan(userId: string, username: string) {
    const reason = prompt(`Причина глобального бана ${username}:`);
    if (reason === null) return;
    setActionLoading(userId + "_gban");
    try {
      await api.admin.globalBan(userId, reason);
      setMembers(prev => prev.filter(m => m.id !== userId));
      showToast(`${username} глобально забанен`);
    } catch { showToast("Ошибка"); }
    finally { setActionLoading(null); }
  }

  async function handleCreateInvite() {
    setInviteLoading(true);
    try {
      const res = await api.guilds.createInvite(guild.id);
      setInviteCode(res.code);
    } catch { showToast("Не удалось создать инвайт"); }
    finally { setInviteLoading(false); }
  }

  const filtered = members.filter(m =>
    m.id !== currentUserId &&
    m.username.toLowerCase().includes(search.toLowerCase())
  );

  return (
    <div style={{
      position: "fixed", inset: 0, background: "rgba(0,0,0,0.75)",
      display: "flex", alignItems: "center", justifyContent: "center",
      zIndex: 600, backdropFilter: "blur(6px)",
    }} onClick={onClose}>
      <div style={{
        background: "var(--bg-surface)", borderRadius: 20, width: 460,
        maxHeight: "80vh", overflow: "hidden", display: "flex", flexDirection: "column",
        border: "1px solid var(--border)", boxShadow: "var(--shadow-lg)",
        animation: "fadeUp 0.2s ease",
      }} onClick={e => e.stopPropagation()}>

        {/* Header */}
        <div style={{ padding: "20px 24px 0", borderBottom: "1px solid var(--border)" }}>
          <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between", marginBottom: 16 }}>
            <div>
              <div style={{ fontSize: 16, fontWeight: 700, color: "var(--text-primary)" }}>
                Управление сервером
              </div>
              <div style={{ fontSize: 12, color: "var(--text-muted)", marginTop: 2 }}>
                {guild.name}
                {!loading && (
                  <span style={{ marginLeft: 8, color: "var(--accent)", fontWeight: 600 }}>
                    {members.length} {members.length === 1 ? "участник" : members.length < 5 ? "участника" : "участников"}
                  </span>
                )}
              </div>
            </div>
            <button onClick={onClose} style={{
              background: "none", border: "none", cursor: "pointer",
              color: "var(--text-muted)", fontSize: 18, padding: "2px 6px", borderRadius: 6,
            }}>✕</button>
          </div>
          {/* Tabs */}
          <div style={{ display: "flex" }}>
            {([
              { id: "members", label: `Участники${!loading ? ` (${members.length})` : ""}` },
              { id: "invite",  label: "Инвайт" },
            ] as { id: "members" | "invite"; label: string }[]).map(t => (
              <button key={t.id} onClick={() => setTab(t.id)} style={{
                padding: "8px 16px", background: "none", border: "none", cursor: "pointer",
                fontSize: 13, fontWeight: 600,
                color: tab === t.id ? "var(--text-primary)" : "var(--text-muted)",
                borderBottom: `2px solid ${tab === t.id ? "var(--accent)" : "transparent"}`,
                marginBottom: -1, transition: "all 0.15s",
              }}>{t.label}</button>
            ))}
          </div>
        </div>

        {/* Content */}
        <div style={{ flex: 1, overflowY: "auto", padding: "16px 24px" }}>

          {tab === "members" && (
            <>
              {/* Search */}
              <div style={{
                display: "flex", alignItems: "center", gap: 8,
                background: "var(--bg-input)", borderRadius: 8,
                padding: "7px 10px", border: "1px solid var(--border)", marginBottom: 12,
              }}>
                <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="var(--text-muted)" strokeWidth="2">
                  <circle cx="11" cy="11" r="8"/><line x1="21" y1="21" x2="16.65" y2="16.65"/>
                </svg>
                <input value={search} onChange={e => setSearch(e.target.value)}
                  placeholder="Поиск участников..."
                  style={{
                    flex: 1, background: "none", border: "none", outline: "none",
                    color: "var(--text-primary)", fontSize: 13, fontFamily: "inherit",
                  }}
                />
              </div>

              {loading ? (
                <div style={{ textAlign: "center", color: "var(--text-muted)", padding: 32, fontSize: 13 }}>
                  <div style={{
                    width: 20, height: 20, border: "2px solid var(--border)",
                    borderTopColor: "var(--accent)", borderRadius: "50%",
                    animation: "spin 0.7s linear infinite", margin: "0 auto 10px",
                  }} />
                  Загрузка участников...
                </div>
              ) : loadError ? (
                <div style={{ textAlign: "center", padding: 32 }}>
                  <div style={{ fontSize: 13, color: "#f04f5e", marginBottom: 12 }}>{loadError}</div>
                  <button onClick={() => {
                    setLoading(true); setLoadError("");
                    api.guilds.members(guild.id)
                      .then(res => setMembers(res))
                      .catch(e => setLoadError(e?.message ?? "Ошибка"))
                      .finally(() => setLoading(false));
                  }} style={{
                    padding: "7px 16px", borderRadius: 8, border: "none",
                    background: "var(--accent-dim)", color: "var(--accent)",
                    cursor: "pointer", fontSize: 12, fontWeight: 600,
                  }}>Повторить</button>
                </div>
              ) : filtered.length === 0 ? (
                <div style={{ textAlign: "center", color: "var(--text-muted)", padding: 32, fontSize: 13 }}>
                  Участники не найдены
                </div>
              ) : (
                <div style={{ display: "flex", flexDirection: "column", gap: 4 }}>
                  {filtered.map(m => (
                    <div key={m.id} style={{
                      display: "flex", alignItems: "center", gap: 10,
                      padding: "10px 12px", borderRadius: 10,
                      background: "var(--bg-elevated)", border: "1px solid var(--border)",
                    }}>
                      {/* Avatar */}
                      <div style={{
                        width: 34, height: 34, borderRadius: 10, flexShrink: 0,
                        background: `linear-gradient(135deg, #7c6af7, #a78bfa)`,
                        display: "flex", alignItems: "center", justifyContent: "center",
                        fontSize: 13, fontWeight: 700, color: "#fff",
                      }}>
                        {m.username.charAt(0).toUpperCase()}
                      </div>
                      <div style={{ flex: 1, minWidth: 0 }}>
                        <div style={{ fontSize: 13, fontWeight: 600, color: "var(--text-primary)", overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap" }}>
                          {m.username}
                        </div>
                      </div>
                      {/* Actions */}
                      <div style={{ display: "flex", gap: 4, flexShrink: 0 }}>
                        {/* Kick */}
                        <button
                          disabled={!!actionLoading}
                          onClick={() => handleKick(m.id, m.username)}
                          title="Кикнуть с сервера"
                          style={{
                            padding: "5px 10px", borderRadius: 6, border: "none",
                            background: "rgba(245,166,35,0.15)", color: "#f5a623",
                            cursor: "pointer", fontSize: 11, fontWeight: 700,
                            opacity: actionLoading === m.id + "_kick" ? 0.5 : 1,
                          }}>
                          Кик
                        </button>
                        {/* Server ban */}
                        <button
                          disabled={!!actionLoading}
                          onClick={() => handleBan(m.id, m.username)}
                          title="Забанить на сервере"
                          style={{
                            padding: "5px 10px", borderRadius: 6, border: "none",
                            background: "rgba(240,79,94,0.15)", color: "#f04f5e",
                            cursor: "pointer", fontSize: 11, fontWeight: 700,
                            opacity: actionLoading === m.id + "_ban" ? 0.5 : 1,
                          }}>
                          Бан
                        </button>
                        {/* Global ban */}
                        <button
                          disabled={!!actionLoading}
                          onClick={() => handleGlobalBan(m.id, m.username)}
                          title="Глобальный бан (из всего приложения)"
                          style={{
                            padding: "5px 10px", borderRadius: 6, border: "none",
                            background: "rgba(127,0,0,0.25)", color: "#ff4444",
                            cursor: "pointer", fontSize: 11, fontWeight: 700,
                            opacity: actionLoading === m.id + "_gban" ? 0.5 : 1,
                          }}>
                          🔨 Глоб.
                        </button>
                      </div>
                    </div>
                  ))}
                </div>
              )}

              {/* Legend */}
              <div style={{
                marginTop: 16, padding: "10px 12px", borderRadius: 10,
                background: "rgba(255,255,255,0.02)", border: "1px solid var(--border)",
                fontSize: 11, color: "var(--text-muted)", lineHeight: 1.7,
              }}>
                <div><span style={{ color: "#f5a623", fontWeight: 700 }}>Кик</span> — удаляет с сервера, может вернуться по инвайту</div>
                <div><span style={{ color: "#f04f5e", fontWeight: 700 }}>Бан</span> — блокирует доступ к этому серверу навсегда</div>
                <div><span style={{ color: "#ff4444", fontWeight: 700 }}>Глоб. бан</span> — блокирует доступ ко всему приложению</div>
              </div>
            </>
          )}

          {tab === "invite" && (
            <div style={{ display: "flex", flexDirection: "column", gap: 16 }}>
              <div style={{ fontSize: 13, color: "var(--text-muted)", lineHeight: 1.6 }}>
                Создай инвайт-ссылку и поделись ей с теми, кого хочешь пригласить на сервер.
              </div>

              {inviteCode ? (
                <div>
                  <div style={{ fontSize: 11, fontWeight: 700, color: "var(--text-muted)", textTransform: "uppercase", letterSpacing: 0.8, marginBottom: 8 }}>
                    Ссылка-приглашение
                  </div>
                  <div style={{
                    display: "flex", gap: 8, alignItems: "center",
                    padding: "10px 14px", borderRadius: 10,
                    background: "var(--bg-input)", border: "1px solid var(--border-accent)",
                  }}>
                    <span style={{ flex: 1, fontSize: 13, color: "var(--accent)", fontFamily: "monospace", wordBreak: "break-all" }}>
                      {inviteCode}
                    </span>
                    <button onClick={() => { navigator.clipboard.writeText(inviteCode); showToast("Скопировано!"); }} style={{
                      padding: "6px 12px", background: "var(--accent-dim)",
                      border: "1px solid var(--border-accent)", borderRadius: 8,
                      color: "var(--accent)", cursor: "pointer", fontSize: 12, fontWeight: 600, flexShrink: 0,
                    }}>Копировать</button>
                  </div>
                  <div style={{ fontSize: 11, color: "var(--text-muted)", marginTop: 8 }}>
                    Инвайт действует 24 часа
                  </div>
                </div>
              ) : (
                <button onClick={handleCreateInvite} disabled={inviteLoading} style={{
                  padding: "12px", borderRadius: 10,
                  background: "linear-gradient(135deg, #7c6af7, #a78bfa)",
                  border: "none", color: "#fff", cursor: "pointer",
                  fontSize: 13, fontWeight: 700,
                  boxShadow: "0 4px 12px rgba(124,106,247,0.4)",
                  opacity: inviteLoading ? 0.7 : 1,
                }}>
                  {inviteLoading ? "Создаём..." : "Создать инвайт"}
                </button>
              )}
            </div>
          )}
        </div>
      </div>

      {/* Toast */}
      {toast && (
        <div style={{
          position: "fixed", bottom: 24, left: "50%", transform: "translateX(-50%)",
          background: "var(--bg-elevated)", border: "1px solid var(--border)",
          borderRadius: 10, padding: "10px 20px", fontSize: 13, fontWeight: 600,
          color: "var(--text-primary)", boxShadow: "var(--shadow-md)",
          animation: "fadeUp 0.2s ease", zIndex: 9999,
        }}>{toast}</div>
      )}
    </div>
  );
}

// ── Пустые состояния ──────────────────────────────────────────────────────────

function NoServerSelected() {
  return (
    <div style={{
      display: "flex", flexDirection: "column" as const,
      alignItems: "center", padding: "32px 16px", gap: 16,
      animation: "fadeUp 0.3s ease",
    }}>
      <div style={{
        width: 56, height: 56, borderRadius: 16, marginTop: 8,
        background: "linear-gradient(135deg, rgba(124,106,247,0.2), rgba(167,139,250,0.1))",
        border: "1px solid rgba(124,106,247,0.25)",
        display: "flex", alignItems: "center", justifyContent: "center",
        boxShadow: "0 0 24px rgba(124,106,247,0.15)",
      }}>
        <svg width="26" height="26" viewBox="0 0 28 28" fill="none">
          <path d="M14 2L26 8V20L14 26L2 20V8L14 2Z" fill="url(#cl_lg)" opacity="0.9"/>
          <path d="M14 8L20 11V17L14 20L8 17V11L14 8Z" fill="rgba(255,255,255,0.2)"/>
          <defs>
            <linearGradient id="cl_lg" x1="2" y1="2" x2="26" y2="26" gradientUnits="userSpaceOnUse">
              <stop stopColor="#7c6af7"/><stop offset="1" stopColor="#a78bfa"/>
            </linearGradient>
          </defs>
        </svg>
      </div>
      <div style={{ textAlign: "center" as const }}>
        <div style={{ fontSize: 13, fontWeight: 700, color: "var(--text-secondary)", marginBottom: 4 }}>
          Welcome to Zenthril
        </div>
        <div style={{ fontSize: 11, color: "var(--text-muted)", lineHeight: 1.5 }}>
          Select a server from the left<br/>to start chatting
        </div>
      </div>
    </div>
  );
}

function NoChannelsYet() {
  return (
    <div style={{
      display: "flex", flexDirection: "column" as const,
      alignItems: "center", padding: "32px 16px", gap: 12,
      animation: "fadeUp 0.3s ease",
    }}>
      <div style={{
        width: 44, height: 44, borderRadius: 12,
        background: "rgba(255,255,255,0.04)", border: "1px solid var(--border)",
        display: "flex", alignItems: "center", justifyContent: "center",
      }}>
        <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="var(--text-muted)" strokeWidth="1.5">
          <path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"/>
          <line x1="12" y1="8" x2="12" y2="12"/><line x1="12" y1="16" x2="12.01" y2="16"/>
        </svg>
      </div>
      <div style={{ textAlign: "center" as const }}>
        <div style={{ fontSize: 12, fontWeight: 600, color: "var(--text-secondary)", marginBottom: 4 }}>
          No channels yet
        </div>
        <div style={{ fontSize: 11, color: "var(--text-muted)", lineHeight: 1.5 }}>
          Ask an admin to create<br/>the first channel
        </div>
      </div>
    </div>
  );
}
