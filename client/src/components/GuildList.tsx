import React, { useState } from "react";
import type { GuildAPI } from "../api/index";
import { api } from "../api/index";
import { apiErrorFields } from "../util/errors";

interface GuildListProps {
  guilds: GuildAPI[];
  selectedGuildId: string | null;
  onSelect: (id: string) => void;
  onCreateGuild: (name: string) => Promise<void>;
  onJoinGuild?: (guild: GuildAPI) => void;
  hasBg?: boolean;
  panelBg?: string;
}

const COLORS = ["#7c6af7","#3ecf8e","#f5a623","#f04f5e","#06b6d4","#ec4899"];
const guildColor = (name: string) => COLORS[name.charCodeAt(0) % COLORS.length];

// ── Guild icon in sidebar ─────────────────────────────────────────────────────
function GuildIcon({ guild, selected, onClick }: { guild: GuildAPI; selected: boolean; onClick: () => void }) {
  const [hovered, setHovered] = useState(false);
  const color = guildColor(guild.name);
  return (
    <div style={{ position: "relative" }}>
      <div style={{
        width: 44, height: 44, borderRadius: selected ? 14 : "50%",
        background: selected ? color : "var(--bg-elevated)",
        border: `2px solid ${selected ? color : "var(--border)"}`,
        display: "flex", alignItems: "center", justifyContent: "center",
        cursor: "pointer", fontSize: 16, fontWeight: 700,
        color: selected ? "#fff" : "var(--text-secondary)",
        transition: "all 0.2s cubic-bezier(0.34,1.56,0.64,1)",
        boxShadow: selected ? `0 4px 16px ${color}44` : "none",
        userSelect: "none" as const, flexShrink: 0,
        transform: hovered && !selected ? "scale(1.08)" : "scale(1)",
      }} onClick={onClick} onMouseEnter={() => setHovered(true)} onMouseLeave={() => setHovered(false)}>
        {guild.name.charAt(0).toUpperCase()}
      </div>
      {hovered && (
        <div style={{
          position: "absolute", left: 54, top: "50%", transform: "translateY(-50%)",
          background: "var(--bg-elevated)", color: "var(--text-primary)",
          padding: "6px 10px", borderRadius: 6, fontSize: 12, fontWeight: 600,
          whiteSpace: "nowrap" as const, pointerEvents: "none" as const, zIndex: 100,
          border: "1px solid var(--border)", boxShadow: "var(--shadow-md)",
        }}>{guild.name}</div>
      )}
    </div>
  );
}

// ── Server creation / join modal ──────────────────────────────────────────────
type ModalTab = "pick" | "create" | "join";

interface ServerModalProps {
  onClose: () => void;
  onCreateGuild: (name: string) => Promise<void>;
  onJoinGuild?: (guild: GuildAPI) => void;
}

function ServerModal({ onClose, onCreateGuild, onJoinGuild }: ServerModalProps) {
  const [tab, setTab] = useState<ModalTab>("pick");
  const [name, setName] = useState("");
  const [inviteCode, setInviteCode] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  async function handleCreate(e: React.FormEvent) {
    e.preventDefault();
    if (!name.trim()) return;
    setLoading(true); setError("");
    try {
      await onCreateGuild(name.trim());
      onClose();
    } catch (err: unknown) {
      setError(apiErrorFields(err).message ?? "Failed to create server");
    } finally { setLoading(false); }
  }

  async function handleJoin(e: React.FormEvent) {
    e.preventDefault();
    const code = inviteCode.trim().replace(/.*\//, ""); // поддержка полных ссылок
    if (!code) return;
    setLoading(true); setError("");
    try {
      const guild = await api.guilds.joinByInvite(code);
      onJoinGuild?.(guild);
      onClose();
    } catch (err: unknown) {
      setError(apiErrorFields(err).message ?? "Invalid or expired invite");
    } finally { setLoading(false); }
  }

  return (
    <div style={{
      position: "fixed", inset: 0, background: "rgba(0,0,0,0.75)",
      display: "flex", alignItems: "center", justifyContent: "center",
      zIndex: 500, backdropFilter: "blur(6px)",
    }} onClick={onClose}>
      <div style={{
        background: "var(--bg-surface)", borderRadius: 20,
        width: 420, boxShadow: "var(--shadow-lg)",
        border: "1px solid var(--border)", animation: "fadeUp 0.2s ease",
        overflow: "hidden",
      }} onClick={e => e.stopPropagation()}>

        {/* ── Pick screen ── */}
        {tab === "pick" && (
          <div style={{ padding: "32px 28px" }}>
            <div style={{ textAlign: "center", marginBottom: 24 }}>
              <div style={{ fontSize: 22, fontWeight: 800, color: "var(--text-primary)", marginBottom: 6 }}>
                Добавить сервер
              </div>
              <div style={{ fontSize: 13, color: "var(--text-muted)" }}>
                Создай свой или войди по инвайту
              </div>
            </div>

            <div style={{ display: "flex", flexDirection: "column", gap: 10 }}>
              {/* Create */}
              <div onClick={() => setTab("create")} style={{
                padding: "18px 20px", borderRadius: 14, cursor: "pointer",
                border: "1px solid var(--border)", background: "var(--bg-elevated)",
                display: "flex", alignItems: "center", gap: 16,
                transition: "all 0.15s",
              }}
              onMouseEnter={e => (e.currentTarget.style.borderColor = "var(--accent)")}
              onMouseLeave={e => (e.currentTarget.style.borderColor = "var(--border)")}>
                <div style={{
                  width: 44, height: 44, borderRadius: 12, flexShrink: 0,
                  background: "linear-gradient(135deg, #7c6af7, #a78bfa)",
                  display: "flex", alignItems: "center", justifyContent: "center",
                  boxShadow: "0 4px 12px rgba(124,106,247,0.4)",
                }}>
                  <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="#fff" strokeWidth="2.5">
                    <line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/>
                  </svg>
                </div>
                <div>
                  <div style={{ fontWeight: 700, fontSize: 14, color: "var(--text-primary)" }}>Создать сервер</div>
                  <div style={{ fontSize: 12, color: "var(--text-muted)", marginTop: 2 }}>Свой сервер с нуля</div>
                </div>
                <svg style={{ marginLeft: "auto", flexShrink: 0 }} width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="var(--text-muted)" strokeWidth="2">
                  <polyline points="9 18 15 12 9 6"/>
                </svg>
              </div>

              {/* Join */}
              <div onClick={() => setTab("join")} style={{
                padding: "18px 20px", borderRadius: 14, cursor: "pointer",
                border: "1px solid var(--border)", background: "var(--bg-elevated)",
                display: "flex", alignItems: "center", gap: 16,
                transition: "all 0.15s",
              }}
              onMouseEnter={e => (e.currentTarget.style.borderColor = "#3ecf8e")}
              onMouseLeave={e => (e.currentTarget.style.borderColor = "var(--border)")}>
                <div style={{
                  width: 44, height: 44, borderRadius: 12, flexShrink: 0,
                  background: "linear-gradient(135deg, #3ecf8e, #06b6d4)",
                  display: "flex", alignItems: "center", justifyContent: "center",
                  boxShadow: "0 4px 12px rgba(62,207,142,0.3)",
                }}>
                  <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="#fff" strokeWidth="2.5">
                    <path d="M10 13a5 5 0 0 0 7.54.54l3-3a5 5 0 0 0-7.07-7.07l-1.72 1.71"/>
                    <path d="M14 11a5 5 0 0 0-7.54-.54l-3 3a5 5 0 0 0 7.07 7.07l1.71-1.71"/>
                  </svg>
                </div>
                <div>
                  <div style={{ fontWeight: 700, fontSize: 14, color: "var(--text-primary)" }}>Войти по инвайту</div>
                  <div style={{ fontSize: 12, color: "var(--text-muted)", marginTop: 2 }}>Есть ссылка-приглашение?</div>
                </div>
                <svg style={{ marginLeft: "auto", flexShrink: 0 }} width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="var(--text-muted)" strokeWidth="2">
                  <polyline points="9 18 15 12 9 6"/>
                </svg>
              </div>
            </div>

            <button onClick={onClose} style={{
              width: "100%", marginTop: 16, padding: "10px",
              background: "none", border: "none", cursor: "pointer",
              color: "var(--text-muted)", fontSize: 13, fontWeight: 600,
            }}>Отмена</button>
          </div>
        )}

        {/* ── Create screen ── */}
        {tab === "create" && (
          <div>
            {/* Header with gradient */}
            <div style={{
              height: 100, background: "linear-gradient(135deg, #7c6af7, #a78bfa)",
              display: "flex", alignItems: "center", justifyContent: "center",
              position: "relative",
            }}>
              <button onClick={() => setTab("pick")} style={{
                position: "absolute", left: 16, top: 16,
                background: "rgba(0,0,0,0.3)", border: "none", borderRadius: 8,
                color: "#fff", cursor: "pointer", padding: "6px 10px", fontSize: 12,
              }}>← Назад</button>
              <div style={{ textAlign: "center" }}>
                <div style={{ fontSize: 20, fontWeight: 800, color: "#fff" }}>Создать сервер</div>
                <div style={{ fontSize: 12, color: "rgba(255,255,255,0.7)", marginTop: 4 }}>Твой сервер, твои правила</div>
              </div>
            </div>

            <form onSubmit={handleCreate} style={{ padding: "24px 28px" }}>
              <div style={{ marginBottom: 20 }}>
                <label style={labelS}>Название сервера</label>
                <input autoFocus value={name} onChange={e => setName(e.target.value)}
                  placeholder="Мой крутой сервер" maxLength={100} required
                  style={{
                    width: "100%", padding: "11px 14px",
                    background: "var(--bg-input)", border: "1px solid var(--border)",
                    borderRadius: 10, color: "var(--text-primary)",
                    fontSize: 14, outline: "none", fontFamily: "inherit",
                    boxSizing: "border-box" as const,
                  }}
                  onFocus={e => (e.target.style.borderColor = "var(--accent)")}
                  onBlur={e => (e.target.style.borderColor = "var(--border)")}
                />
                <div style={{ fontSize: 11, color: "var(--text-muted)", marginTop: 6 }}>
                  Название можно изменить позже
                </div>
              </div>

              {error && <div style={{ fontSize: 12, color: "#f04f5e", marginBottom: 12 }}>{error}</div>}

              <div style={{ display: "flex", gap: 8 }}>
                <button type="button" onClick={() => { setTab("pick"); setError(""); }} style={{
                  flex: 1, padding: "11px", background: "var(--bg-elevated)",
                  border: "1px solid var(--border)", borderRadius: 10,
                  color: "var(--text-secondary)", cursor: "pointer", fontSize: 13, fontWeight: 600,
                }}>Отмена</button>
                <button type="submit" disabled={loading || !name.trim()} style={{
                  flex: 2, padding: "11px",
                  background: name.trim() ? "linear-gradient(135deg, #7c6af7, #a78bfa)" : "var(--bg-elevated)",
                  border: "none", borderRadius: 10, color: name.trim() ? "#fff" : "var(--text-muted)",
                  cursor: name.trim() ? "pointer" : "default",
                  fontSize: 13, fontWeight: 700,
                  boxShadow: name.trim() ? "0 4px 12px rgba(124,106,247,0.4)" : "none",
                  transition: "all 0.2s",
                }}>
                  {loading ? "Создаём..." : "Создать сервер"}
                </button>
              </div>
            </form>
          </div>
        )}

        {/* ── Join screen ── */}
        {tab === "join" && (
          <div>
            <div style={{
              height: 100, background: "linear-gradient(135deg, #3ecf8e, #06b6d4)",
              display: "flex", alignItems: "center", justifyContent: "center",
              position: "relative",
            }}>
              <button onClick={() => setTab("pick")} style={{
                position: "absolute", left: 16, top: 16,
                background: "rgba(0,0,0,0.3)", border: "none", borderRadius: 8,
                color: "#fff", cursor: "pointer", padding: "6px 10px", fontSize: 12,
              }}>← Назад</button>
              <div style={{ textAlign: "center" }}>
                <div style={{ fontSize: 20, fontWeight: 800, color: "#fff" }}>Войти по инвайту</div>
                <div style={{ fontSize: 12, color: "rgba(255,255,255,0.7)", marginTop: 4 }}>Вставь ссылку или код</div>
              </div>
            </div>

            <form onSubmit={handleJoin} style={{ padding: "24px 28px" }}>
              <div style={{ marginBottom: 20 }}>
                <label style={labelS}>Инвайт-ссылка или код</label>
                <input autoFocus value={inviteCode} onChange={e => setInviteCode(e.target.value)}
                  placeholder="https://zenthril.app/invite/abc123 или abc123"
                  style={{
                    width: "100%", padding: "11px 14px",
                    background: "var(--bg-input)", border: "1px solid var(--border)",
                    borderRadius: 10, color: "var(--text-primary)",
                    fontSize: 14, outline: "none", fontFamily: "inherit",
                    boxSizing: "border-box" as const,
                  }}
                  onFocus={e => (e.target.style.borderColor = "#3ecf8e")}
                  onBlur={e => (e.target.style.borderColor = "var(--border)")}
                />
                <div style={{ fontSize: 11, color: "var(--text-muted)", marginTop: 6 }}>
                  Инвайты выглядят так: <span style={{ color: "var(--accent)" }}>abc123</span>
                </div>
              </div>

              {error && (
                <div style={{
                  fontSize: 12, color: "#f04f5e", marginBottom: 12,
                  padding: "8px 12px", background: "rgba(240,79,94,0.1)",
                  borderRadius: 8, border: "1px solid rgba(240,79,94,0.2)",
                }}>{error}</div>
              )}

              <div style={{ display: "flex", gap: 8 }}>
                <button type="button" onClick={() => { setTab("pick"); setError(""); }} style={{
                  flex: 1, padding: "11px", background: "var(--bg-elevated)",
                  border: "1px solid var(--border)", borderRadius: 10,
                  color: "var(--text-secondary)", cursor: "pointer", fontSize: 13, fontWeight: 600,
                }}>Отмена</button>
                <button type="submit" disabled={loading || !inviteCode.trim()} style={{
                  flex: 2, padding: "11px",
                  background: inviteCode.trim() ? "linear-gradient(135deg, #3ecf8e, #06b6d4)" : "var(--bg-elevated)",
                  border: "none", borderRadius: 10,
                  color: inviteCode.trim() ? "#fff" : "var(--text-muted)",
                  cursor: inviteCode.trim() ? "pointer" : "default",
                  fontSize: 13, fontWeight: 700,
                  boxShadow: inviteCode.trim() ? "0 4px 12px rgba(62,207,142,0.3)" : "none",
                  transition: "all 0.2s",
                }}>
                  {loading ? "Входим..." : "Войти на сервер"}
                </button>
              </div>
            </form>
          </div>
        )}
      </div>
    </div>
  );
}

// ── Main GuildList ────────────────────────────────────────────────────────────
export default function GuildList({ guilds, selectedGuildId, onSelect, onCreateGuild, onJoinGuild, hasBg, panelBg }: GuildListProps) {
  const [showModal, setShowModal] = useState(false);
  const [addHovered, setAddHovered] = useState(false);

  const bg = panelBg ?? (hasBg ? "var(--panel-bg)" : "rgba(0,0,0,0.25)");

  return (
    <>
      <div style={{
        width: 68,
        background: bg,
        backdropFilter: hasBg ? "blur(16px)" : "none",
        display: "flex", flexDirection: "column" as const, alignItems: "center",
        padding: "12px 0", gap: 8, overflowY: "auto" as const, flexShrink: 0,
        borderRight: "1px solid var(--border)",
      }}>
        {/* Logo */}
        <div style={{
          width: 44, height: 44, borderRadius: 14,
          background: "linear-gradient(135deg, #7c6af7, #a78bfa)",
          display: "flex", alignItems: "center", justifyContent: "center",
          marginBottom: 4, boxShadow: "0 4px 16px rgba(124,106,247,0.4)", flexShrink: 0,
        }}>
          <svg width="20" height="20" viewBox="0 0 28 28" fill="none">
            <path d="M14 2L26 8V20L14 26L2 20V8L14 2Z" fill="rgba(255,255,255,0.9)"/>
            <path d="M14 8L20 11V17L14 20L8 17V11L14 8Z" fill="rgba(255,255,255,0.3)"/>
          </svg>
        </div>

        <div style={{ width: 28, height: 1, background: "var(--border)", margin: "2px 0", flexShrink: 0 }} />

        {guilds.map(g => (
          <GuildIcon key={g.id} guild={g} selected={g.id === selectedGuildId} onClick={() => onSelect(g.id)} />
        ))}

        {guilds.length > 0 && <div style={{ width: 28, height: 1, background: "var(--border)", margin: "2px 0", flexShrink: 0 }} />}

        {/* Add server button */}
        <button style={{
          width: 44, height: 44, borderRadius: addHovered ? 14 : "50%",
          background: addHovered ? "rgba(62,207,142,0.15)" : "var(--bg-elevated)",
          border: `2px solid ${addHovered ? "rgba(62,207,142,0.5)" : "var(--border)"}`,
          display: "flex", alignItems: "center", justifyContent: "center",
          cursor: "pointer", fontSize: 22, color: "#3ecf8e",
          transition: "all 0.2s cubic-bezier(0.34,1.56,0.64,1)", flexShrink: 0,
        }}
        onClick={() => setShowModal(true)}
        onMouseEnter={() => setAddHovered(true)}
        onMouseLeave={() => setAddHovered(false)}
        title="Добавить сервер">
          +
        </button>
      </div>

      {showModal && (
        <ServerModal
          onClose={() => setShowModal(false)}
          onCreateGuild={onCreateGuild}
          onJoinGuild={onJoinGuild}
        />
      )}
    </>
  );
}

const labelS: React.CSSProperties = {
  display: "block", fontSize: 11, fontWeight: 700,
  color: "var(--text-muted)", letterSpacing: 0.8,
  textTransform: "uppercase", marginBottom: 8,
};
