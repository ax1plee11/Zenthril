import { useState, useEffect, useCallback, useRef } from "react";
import { api } from "../api/index";
import type { GuildAPI, ChannelAPI } from "../api/index";
import { useAuth } from "../store/auth";
import { useTheme } from "../store/theme";
import { DEFAULT_TOPBAR_ITEMS } from "../store/theme";
import { connectGlobalWS, onWSEvent, sendWSEvent } from "../store/wsGlobal";
import GuildList from "./GuildList";
import ChannelList from "./ChannelList";
import ChatView from "./ChatView";
import ThemeSettings from "./ThemeSettings";
import ProfileModal from "./ProfileModal";
import UserSearch from "./UserSearch";
import NotificationsPanel, { useUnreadCount } from "./NotificationsPanel";
import { loadProfile } from "./ProfileModal";

export default function MainLayout() {
  const { user, logout } = useAuth();
  const [guilds, setGuilds]                       = useState<GuildAPI[]>([]);
  const [selectedGuildId, setSelectedGuildId]     = useState<string | null>(null);
  const [channels, setChannels]                   = useState<ChannelAPI[]>([]);
  const [selectedChannelId, setSelectedChannelId] = useState<string | null>(null);
  const [showTheme, setShowTheme]         = useState(false);
  const [showProfile, setShowProfile]     = useState(false);
  const [showSearch, setShowSearch]       = useState(false);
  const [showNotifications, setShowNotifications] = useState(false);
  const [showProfileMenu, setShowProfileMenu]     = useState(false);
  const [inviteToast, setInviteToast]     = useState<{ code: string } | null>(null);
  const [friendToast, setFriendToast]     = useState<{ type: "request" | "accepted"; username: string; userId: string } | null>(null);
  const unreadCount   = useUnreadCount();
  const profileMenuRef = useRef<HTMLDivElement>(null);
  const profileBtnRef  = useRef<HTMLButtonElement>(null);
  const notifBtnRef    = useRef<HTMLButtonElement>(null);
  const profile = loadProfile();

  // WS
  useEffect(() => {
    connectGlobalWS();
    const u1 = onWSEvent("invite.received", (d) => setInviteToast({ code: d.invite_code as string }));
    const u2 = onWSEvent("friend.request",  (d) => setFriendToast({ type: "request",  username: d.from_username as string, userId: d.from_user_id as string }));
    const u3 = onWSEvent("friend.accepted", (d) => setFriendToast({ type: "accepted", username: d.from_username as string, userId: d.from_user_id as string }));
    return () => { u1(); u2(); u3(); };
  }, []);

  // Закрытие меню профиля по клику вне
  useEffect(() => {
    if (!showProfileMenu) return;
    function h(e: MouseEvent) {
      if (profileMenuRef.current?.contains(e.target as Node)) return;
      if (profileBtnRef.current?.contains(e.target as Node)) return;
      setShowProfileMenu(false);
    }
    document.addEventListener("mousedown", h);
    return () => document.removeEventListener("mousedown", h);
  }, [showProfileMenu]);

  useEffect(() => { api.guilds.list().then(setGuilds).catch(console.error); }, []);

  useEffect(() => {
    if (!selectedGuildId) { setChannels([]); setSelectedChannelId(null); return; }
    api.guilds.channels(selectedGuildId).then(chs => {
      setChannels(chs);
      const first = chs.find(c => c.type === "text");
      if (first) setSelectedChannelId(first.id);
    }).catch(console.error);
  }, [selectedGuildId]);

  const handleSelectGuild   = useCallback((id: string) => { setSelectedGuildId(id); setSelectedChannelId(null); }, []);
  const handleSelectChannel = useCallback((id: string) => setSelectedChannelId(id), []);
  const handleCreateGuild   = useCallback(async (name: string) => {
    const g = await api.guilds.create(name);
    setGuilds(prev => [...prev, g]);
    setSelectedGuildId(g.id);
  }, []);
  const handleJoinGuild = useCallback((g: GuildAPI) => {
    setGuilds(prev => prev.find(x => x.id === g.id) ? prev : [...prev, g]);
    setSelectedGuildId(g.id);
  }, []);
  const handleLogout = useCallback(() => { api.auth.logout().catch(() => {}); logout(); }, [logout]);

  const selectedGuild   = guilds.find(g => g.id === selectedGuildId) ?? null;
  const selectedChannel = channels.find(c => c.id === selectedChannelId) ?? null;
  const username        = user?.username ?? "?";
  const { theme }       = useTheme();
  const hasBg    = !!(theme.chatBackground && !theme.chatBackground.startsWith("__pattern__"));
  const hasAppBg = !!theme.animatedPresetId;
  const glassy   = hasBg || hasAppBg;

  // Единая прозрачность для всех панелей из настроек
  const panelBg   = glassy ? "var(--panel-bg)"         : "var(--bg-surface)";
  const sidebarBg = panelBg;
  const topbarBg  = glassy ? "var(--panel-topbar)"     : (theme.scheme === "light" ? "#f2f3f5" : "var(--bg-base)");

  return (
    <div style={{ display: "flex", flexDirection: "column", height: "100vh", overflow: "hidden" }}>

      {/* ── STEAM-STYLE TOPBAR ── */}
      <div style={{
        height: 48, flexShrink: 0,
        background: topbarBg,
        backdropFilter: glassy ? "blur(20px)" : "none",
        WebkitBackdropFilter: glassy ? "blur(20px)" : "none",
        borderBottom: "1px solid var(--border)",
        display: "flex", alignItems: "center",
        padding: "0 12px 0 16px",
        gap: 6, zIndex: 100,
        boxShadow: "0 1px 0 rgba(255,255,255,0.04), 0 2px 12px rgba(0,0,0,0.3)",
      }}>
        {/* Logo */}
        <div style={{
          width: 28, height: 28, borderRadius: 8, flexShrink: 0,
          background: "linear-gradient(135deg, #7c6af7, #a78bfa)",
          display: "flex", alignItems: "center", justifyContent: "center",
          boxShadow: "0 2px 8px rgba(124,106,247,0.4)",
          marginRight: 4,
        }}>
          <svg width="14" height="14" viewBox="0 0 28 28" fill="none">
            <path d="M14 2L26 8V20L14 26L2 20V8L14 2Z" fill="rgba(255,255,255,0.9)"/>
            <path d="M14 8L20 11V17L14 20L8 17V11L14 8Z" fill="rgba(255,255,255,0.3)"/>
          </svg>
        </div>

        <span style={{ fontSize: 14, fontWeight: 700, color: "var(--text-primary)", marginRight: 8 }}>
          Zenthril
        </span>

        {/* Spacer */}
        <div style={{ flex: 1 }} />

        {/* Dynamic topbar items from theme */}
        {(theme.topbarItems ?? DEFAULT_TOPBAR_ITEMS).filter(it => it.visible).map(item => {
          if (item.id === "divider") {
            return <div key="divider" style={{ width: 1, height: 20, background: "var(--border)", margin: "0 4px" }} />;
          }
          if (item.id === "search") return (
            <TopBtn key="search" title="Поиск пользователей" onClick={() => setShowSearch(true)}>
              <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                <circle cx="11" cy="11" r="8"/><line x1="21" y1="21" x2="16.65" y2="16.65"/>
              </svg>
            </TopBtn>
          );
          if (item.id === "notifications") return (
            <div key="notifications" style={{ position: "relative" }}>
              <TopBtn ref={notifBtnRef} title="Уведомления" onClick={() => setShowNotifications(v => !v)} active={showNotifications}>
                <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                  <path d="M18 8A6 6 0 0 0 6 8c0 7-3 9-3 9h18s-3-2-3-9"/>
                  <path d="M13.73 21a2 2 0 0 1-3.46 0"/>
                </svg>
                {unreadCount > 0 && (
                  <span style={{
                    position: "absolute", top: 4, right: 4,
                    minWidth: 15, height: 15, borderRadius: "50%",
                    background: "#f04f5e", border: "1.5px solid var(--bg-base)",
                    fontSize: 8, fontWeight: 800, color: "#fff",
                    display: "flex", alignItems: "center", justifyContent: "center",
                  }}>
                    {unreadCount > 9 ? "9+" : unreadCount}
                  </span>
                )}
              </TopBtn>
            </div>
          );
          if (item.id === "friends") return (
            <TopBtn key="friends" title="Друзья" onClick={() => setShowSearch(true)}>
              <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                <path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/>
                <circle cx="9" cy="7" r="4"/>
                <path d="M23 21v-2a4 4 0 0 0-3-3.87"/>
                <path d="M16 3.13a4 4 0 0 1 0 7.75"/>
              </svg>
            </TopBtn>
          );
          if (item.id === "settings") return (
            <TopBtn key="settings" title="Настройки" onClick={() => setShowTheme(true)}>
              <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                <circle cx="12" cy="12" r="3"/>
                <path d="M19.07 4.93a10 10 0 0 1 0 14.14M4.93 4.93a10 10 0 0 0 0 14.14"/>
              </svg>
            </TopBtn>
          );
          return null;
        })}

        {/* Always-visible profile button */}
        <div style={{ width: 1, height: 20, background: "var(--border)", margin: "0 4px" }} />
        <button
          ref={profileBtnRef}
          onClick={() => setShowProfileMenu(v => !v)}
          style={{
            display: "flex", alignItems: "center", gap: 8,
            padding: "4px 10px 4px 4px", borderRadius: 8, cursor: "pointer",
            background: showProfileMenu ? "rgba(255,255,255,0.1)" : "transparent",
            border: "none",
            transition: "background 0.15s",
          }}
          onMouseEnter={e => { if (!showProfileMenu) e.currentTarget.style.background = "rgba(255,255,255,0.06)"; }}
          onMouseLeave={e => { if (!showProfileMenu) e.currentTarget.style.background = "transparent"; }}
        >
          {/* Avatar */}
          <div style={{
            width: 28, height: 28, borderRadius: 8, flexShrink: 0,
            background: `linear-gradient(135deg, ${profile.theme.bannerColor}, ${profile.theme.bannerColor}99)`,
            display: "flex", alignItems: "center", justifyContent: "center",
            fontSize: 12, fontWeight: 700, color: "#fff", overflow: "hidden",
            border: `2px solid ${profile.theme.bannerColor}55`,
          }}>
            {profile.avatarBase64
              ? <img src={profile.avatarBase64} alt="av" style={{ width: "100%", height: "100%", objectFit: "cover" }} />
              : username.charAt(0).toUpperCase()}
          </div>
          <span style={{ fontSize: 13, fontWeight: 600, color: "var(--text-primary)", maxWidth: 100, overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap" }}>
            {username}
          </span>
          <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="var(--text-muted)" strokeWidth="2.5"
            style={{ transform: showProfileMenu ? "rotate(180deg)" : "none", transition: "transform 0.2s", flexShrink: 0 }}>
            <polyline points="6 9 12 15 18 9"/>
          </svg>
        </button>

        {/* Profile dropdown */}
        {showProfileMenu && (
          <div ref={profileMenuRef} style={{
            position: "fixed", top: 52, right: 8,
            width: 220,
            background: glassy ? "var(--panel-bg-elevated)" : "var(--bg-elevated)",
            backdropFilter: glassy ? "blur(20px)" : "none",
            WebkitBackdropFilter: glassy ? "blur(20px)" : "none",
            border: "1px solid var(--border)",
            borderRadius: 12,
            boxShadow: "0 8px 32px rgba(0,0,0,0.5)",
            overflow: "hidden",
            animation: "fadeUp 0.15s ease",
            zIndex: 500,
          }}>
            {/* Header */}
            <div style={{
              padding: "14px 14px",
              background: "rgba(255,255,255,0.04)",
              borderBottom: "1px solid var(--border)",
              display: "flex", alignItems: "center", gap: 10,
            }}>
              <div style={{
                width: 38, height: 38, borderRadius: 10, flexShrink: 0,
                background: `linear-gradient(135deg, ${profile.theme.bannerColor}, ${profile.theme.bannerColor}99)`,
                display: "flex", alignItems: "center", justifyContent: "center",
                fontSize: 15, fontWeight: 700, color: "#fff", overflow: "hidden",
                border: `2px solid ${profile.theme.bannerColor}55`,
                boxShadow: `0 0 12px ${profile.theme.bannerColor}44`,
              }}>
                {profile.avatarBase64
                  ? <img src={profile.avatarBase64} alt="av" style={{ width: "100%", height: "100%", objectFit: "cover" }} />
                  : username.charAt(0).toUpperCase()}
              </div>
              <div style={{ minWidth: 0 }}>
                <div style={{ fontSize: 13, fontWeight: 700, color: "var(--text-primary)", overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap" }}>
                  {username}
                </div>
                <div style={{ fontSize: 11, color: "var(--text-muted)", marginTop: 2 }}>
                  {profile.status === "online" ? "🟢 Онлайн" : profile.status === "dnd" ? "🔴 Не беспокоить" : "⚫ Офлайн"}
                </div>
              </div>
            </div>

            <div style={{ padding: "6px 0" }}>
              <DropItem icon="👤" label="Мой профиль"         onClick={() => { setShowProfileMenu(false); setShowProfile(true); }} />
              <DropItem icon="🔍" label="Найти пользователей" onClick={() => { setShowProfileMenu(false); setShowSearch(true); }} />
              <DropItem icon="🎨" label="Оформление"          onClick={() => { setShowProfileMenu(false); setShowTheme(true); }} />
            </div>
            <div style={{ height: 1, background: "var(--border)" }} />
            <div style={{ padding: "6px 0" }}>
              <DropItem icon="🚪" label="Выйти из аккаунта" onClick={() => { setShowProfileMenu(false); handleLogout(); }} danger />
            </div>
          </div>
        )}
      </div>

      {/* ── MAIN CONTENT ── */}
      <div style={{ display: "flex", flex: 1, overflow: "hidden" }}>
        {/* Guild list */}
        <GuildList
          guilds={guilds} selectedGuildId={selectedGuildId}
          onSelect={handleSelectGuild} onCreateGuild={handleCreateGuild}
          onJoinGuild={handleJoinGuild} hasBg={glassy} panelBg={glassy ? panelBg : undefined}
        />

        {/* Channel sidebar */}
        <div style={{
          display: "flex", flexDirection: "column", width: 232, flexShrink: 0,
          background: sidebarBg,
          backdropFilter: glassy ? "blur(16px)" : "none",
          WebkitBackdropFilter: glassy ? "blur(16px)" : "none",
          borderRight: "1px solid var(--border)",
          boxShadow: glassy ? "2px 0 12px rgba(0,0,0,0.2)" : "none",
        }}>
          <ChannelList
            guild={selectedGuild} channels={channels}
            selectedChannelId={selectedChannelId}
            onSelect={handleSelectChannel} currentUserId={user?.id ?? ""}
          />
        </div>

        {/* Chat */}
        <ChatView
          channelId={selectedChannelId}
          channelName={selectedChannel?.name ?? ""}
          currentUserId={user?.id ?? ""}
          currentUsername={user?.username ?? ""}
        />
      </div>

      {/* ── MODALS ── */}
      {showTheme   && <ThemeSettings onClose={() => setShowTheme(false)} />}
      {showProfile && <ProfileModal  onClose={() => setShowProfile(false)} />}
      {showSearch  && <UserSearch    onClose={() => setShowSearch(false)} onSendInvite={sendWSEvent} />}

      {showNotifications && (
        <NotificationsPanel
          onClose={() => setShowNotifications(false)}
          anchorBottom={undefined}
          anchorTop={52}
          anchorRight={8}
          onAcceptFriend={async (userId) => { await api.friends.accept(userId); }}
          onDeclineFriend={async (userId) => { await api.friends.decline(userId); }}
          onJoinGuild={async (code) => {
            const g = await api.guilds.joinByInvite(code);
            handleJoinGuild(g);
          }}
        />
      )}

      {/* ── TOASTS ── */}
      {friendToast && (
        <div style={{
          position: "fixed", bottom: inviteToast ? 160 : 24, right: 24, zIndex: 9999,
          background: "var(--bg-elevated)",
          border: `1px solid ${friendToast.type === "request" ? "rgba(124,106,247,0.4)" : "rgba(62,207,142,0.4)"}`,
          borderRadius: 16, padding: "14px 18px", boxShadow: "var(--shadow-lg)",
          animation: "fadeUp 0.2s ease", minWidth: 260, maxWidth: 320,
        }}>
          <div style={{ display: "flex", alignItems: "center", gap: 10, marginBottom: friendToast.type === "request" ? 10 : 0 }}>
            <div style={{
              width: 36, height: 36, borderRadius: 10, flexShrink: 0,
              background: friendToast.type === "request" ? "linear-gradient(135deg,#7c6af7,#a78bfa)" : "linear-gradient(135deg,#3ecf8e,#06b6d4)",
              display: "flex", alignItems: "center", justifyContent: "center", fontSize: 16,
            }}>
              {friendToast.type === "request" ? "👋" : "🎉"}
            </div>
            <div style={{ flex: 1 }}>
              <div style={{ fontSize: 13, fontWeight: 700, color: "var(--text-primary)" }}>
                {friendToast.type === "request" ? "Запрос в друзья" : "Запрос принят!"}
              </div>
              <div style={{ fontSize: 12, color: "var(--text-muted)", marginTop: 1 }}>
                <span style={{ color: friendToast.type === "request" ? "var(--accent)" : "#3ecf8e", fontWeight: 600 }}>{friendToast.username}</span>
                {friendToast.type === "request" ? " хочет добавить вас" : " принял ваш запрос"}
              </div>
            </div>
            <button onClick={() => setFriendToast(null)} style={{ background: "none", border: "none", cursor: "pointer", color: "var(--text-muted)", fontSize: 14 }}>✕</button>
          </div>
          {friendToast.type === "request" && (
            <div style={{ display: "flex", gap: 8 }}>
              <button onClick={async () => { try { await api.friends.accept(friendToast.userId); } finally { setFriendToast(null); } }} style={{
                flex: 1, padding: "7px", borderRadius: 8, border: "none",
                background: "linear-gradient(135deg,#7c6af7,#a78bfa)", color: "#fff", cursor: "pointer", fontSize: 12, fontWeight: 700,
              }}>Принять</button>
              <button onClick={async () => { try { await api.friends.decline(friendToast.userId); } finally { setFriendToast(null); } }} style={{
                flex: 1, padding: "7px", borderRadius: 8,
                background: "var(--bg-input)", border: "1px solid var(--border)",
                color: "var(--text-muted)", cursor: "pointer", fontSize: 12, fontWeight: 600,
              }}>Отклонить</button>
            </div>
          )}
        </div>
      )}

      {inviteToast && (
        <div style={{
          position: "fixed", bottom: 24, right: 24, zIndex: 9999,
          background: "var(--bg-elevated)", border: "1px solid var(--border-accent)",
          borderRadius: 16, padding: "16px 20px", boxShadow: "var(--shadow-lg)",
          animation: "fadeUp 0.2s ease", minWidth: 280, maxWidth: 340,
        }}>
          <div style={{ fontSize: 13, fontWeight: 700, color: "var(--text-primary)", marginBottom: 6 }}>📨 Приглашение на сервер</div>
          <div style={{ fontSize: 12, color: "var(--text-muted)", marginBottom: 14 }}>
            Код: <span style={{ color: "var(--accent)", fontFamily: "monospace", fontWeight: 600 }}>{inviteToast.code}</span>
          </div>
          <div style={{ display: "flex", gap: 8 }}>
            <button onClick={async () => { try { const g = await api.guilds.joinByInvite(inviteToast.code); handleJoinGuild(g); } finally { setInviteToast(null); } }} style={{
              flex: 1, padding: "8px", borderRadius: 8, border: "none",
              background: "linear-gradient(135deg,var(--accent),var(--accent-hover))", color: "#fff", cursor: "pointer", fontSize: 12, fontWeight: 700,
            }}>Принять</button>
            <button onClick={() => setInviteToast(null)} style={{
              flex: 1, padding: "8px", borderRadius: 8,
              background: "var(--bg-input)", border: "1px solid var(--border)",
              color: "var(--text-muted)", cursor: "pointer", fontSize: 12, fontWeight: 600,
            }}>Отклонить</button>
          </div>
        </div>
      )}
    </div>
  );
}

// ── Вспомогательные компоненты ────────────────────────────────────────────────

import { forwardRef } from "react";

const TopBtn = forwardRef<HTMLButtonElement, {
  children: React.ReactNode; title: string; onClick: () => void; active?: boolean;
}>(({ children, title, onClick, active }, ref) => {
  const [hovered, setHovered] = useState(false);
  return (
    <button ref={ref} title={title} onClick={onClick}
      onMouseEnter={() => setHovered(true)} onMouseLeave={() => setHovered(false)}
      style={{
        position: "relative", background: active || hovered ? "rgba(255,255,255,0.08)" : "none",
        border: "none", cursor: "pointer",
        color: active || hovered ? "var(--text-primary)" : "var(--text-muted)",
        padding: "7px 8px", borderRadius: 7,
        display: "flex", alignItems: "center", justifyContent: "center",
        transition: "all 0.15s",
      }}>
      {children}
    </button>
  );
});

function DropItem({ icon, label, onClick, danger }: { icon: string; label: string; onClick: () => void; danger?: boolean }) {
  const [hovered, setHovered] = useState(false);
  return (
    <div onClick={onClick}
      onMouseEnter={() => setHovered(true)} onMouseLeave={() => setHovered(false)}
      style={{
        padding: "9px 14px", cursor: "pointer",
        background: hovered ? "rgba(255,255,255,0.06)" : "transparent",
        display: "flex", alignItems: "center", gap: 10, transition: "background 0.1s",
      }}>
      <span style={{ fontSize: 14, width: 18, textAlign: "center", flexShrink: 0 }}>{icon}</span>
      <span style={{ fontSize: 13, fontWeight: 500, color: danger ? "#f04f5e" : hovered ? "var(--text-primary)" : "var(--text-secondary)" }}>
        {label}
      </span>
    </div>
  );
}
