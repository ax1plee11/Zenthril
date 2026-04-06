/**
 * MainLayout — главный layout: серверы + каналы + чат
 * Требования: 2.2, 3.3
 */

import React, { useState, useEffect, useCallback } from "react";
import { api } from "../api/index";
import type { GuildAPI, ChannelAPI } from "../api/index";
import { useAuth } from "../store/auth";
import GuildList from "./GuildList";
import ChannelList from "./ChannelList";
import ChatView from "./ChatView";

const styles = {
  root: {
    display: "flex",
    height: "100vh",
    overflow: "hidden",
    fontFamily: "'Segoe UI', system-ui, sans-serif",
    background: "#36393f",
    color: "#dcddde",
  } as React.CSSProperties,

  userArea: {
    padding: "8px",
    background: "#292b2f",
    display: "flex",
    alignItems: "center",
    gap: 8,
    borderTop: "1px solid #202225",
  } as React.CSSProperties,

  avatar: {
    width: 32,
    height: 32,
    borderRadius: "50%",
    background: "#7289da",
    display: "flex",
    alignItems: "center",
    justifyContent: "center",
    fontSize: 14,
    fontWeight: 700,
    color: "#fff",
    flexShrink: 0,
  } as React.CSSProperties,

  usernameText: {
    fontSize: 14,
    fontWeight: 600,
    color: "#fff",
    overflow: "hidden",
    textOverflow: "ellipsis",
    whiteSpace: "nowrap" as const,
    flex: 1,
  } as React.CSSProperties,

  logoutBtn: {
    background: "none",
    border: "none",
    cursor: "pointer",
    color: "#72767d",
    fontSize: 16,
    padding: "4px",
    borderRadius: 4,
    transition: "color 0.1s",
    flexShrink: 0,
  } as React.CSSProperties,

  channelSidebar: {
    width: 240,
    background: "#2f3136",
    display: "flex",
    flexDirection: "column" as const,
    flexShrink: 0,
  } as React.CSSProperties,
};

export default function MainLayout() {
  const { user, logout } = useAuth();
  const [guilds, setGuilds] = useState<GuildAPI[]>([]);
  const [selectedGuildId, setSelectedGuildId] = useState<string | null>(null);
  const [channels, setChannels] = useState<ChannelAPI[]>([]);
  const [selectedChannelId, setSelectedChannelId] = useState<string | null>(null);

  // Загружаем список серверов
  useEffect(() => {
    api.guilds.list().then(setGuilds).catch(console.error);
  }, []);

  // При выборе сервера — загружаем каналы
  useEffect(() => {
    if (!selectedGuildId) {
      setChannels([]);
      setSelectedChannelId(null);
      return;
    }

    api.guilds
      .channels(selectedGuildId)
      .then((chs) => {
        setChannels(chs);
        // Автовыбор первого текстового канала
        const first = chs.find((c) => c.type === "text");
        if (first) setSelectedChannelId(first.id);
      })
      .catch(console.error);
  }, [selectedGuildId]);

  const handleSelectGuild = useCallback((guildId: string) => {
    setSelectedGuildId(guildId);
    setSelectedChannelId(null);
  }, []);

  const handleSelectChannel = useCallback((channelId: string) => {
    setSelectedChannelId(channelId);
  }, []);

  const handleCreateGuild = useCallback(async (name: string) => {
    const guild = await api.guilds.create(name);
    setGuilds((prev) => [...prev, guild]);
    setSelectedGuildId(guild.id);
  }, []);

  const handleLogout = useCallback(() => {
    api.auth.logout().catch(() => {});
    logout();
  }, [logout]);

  const selectedGuild = guilds.find((g) => g.id === selectedGuildId) ?? null;
  const selectedChannel = channels.find((c) => c.id === selectedChannelId) ?? null;

  return (
    <div style={styles.root}>
      {/* Список серверов */}
      <GuildList
        guilds={guilds}
        selectedGuildId={selectedGuildId}
        onSelect={handleSelectGuild}
        onCreateGuild={handleCreateGuild}
      />

      {/* Список каналов + user area */}
      <div style={styles.channelSidebar}>
        <ChannelList
          guild={selectedGuild}
          channels={channels}
          selectedChannelId={selectedChannelId}
          onSelect={handleSelectChannel}
          currentUserId={user?.id ?? ""}
        />

        {/* Зона пользователя */}
        <div style={styles.userArea}>
          <div style={styles.avatar}>
            {(user?.username ?? "?").charAt(0).toUpperCase()}
          </div>
          <div style={styles.usernameText}>{user?.username ?? "Гость"}</div>
          <button
            style={styles.logoutBtn}
            onClick={handleLogout}
            title="Выйти"
          >
            ⏻
          </button>
        </div>
      </div>

      {/* Область чата */}
      <ChatView
        channelId={selectedChannelId}
        channelName={selectedChannel?.name ?? ""}
        currentUserId={user?.id ?? ""}
        currentUsername={user?.username ?? ""}
      />
    </div>
  );
}
