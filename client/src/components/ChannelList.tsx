/**
 * ChannelList — список каналов выбранного сервера
 * Требования: 2.2
 */

import React from "react";
import type { GuildAPI, ChannelAPI } from "../api/index";

interface ChannelListProps {
  guild: GuildAPI | null;
  channels: ChannelAPI[];
  selectedChannelId: string | null;
  onSelect: (channelId: string) => void;
  currentUserId: string;
}

const styles = {
  sidebar: {
    width: 240,
    background: "#2f3136",
    display: "flex",
    flexDirection: "column" as const,
    flexShrink: 0,
  } as React.CSSProperties,

  header: {
    padding: "16px",
    borderBottom: "1px solid #202225",
    fontWeight: 700,
    fontSize: 15,
    color: "#fff",
    whiteSpace: "nowrap" as const,
    overflow: "hidden",
    textOverflow: "ellipsis",
  } as React.CSSProperties,

  section: {
    padding: "16px 8px 4px 8px",
    fontSize: 11,
    fontWeight: 700,
    color: "#72767d",
    textTransform: "uppercase" as const,
    letterSpacing: 0.8,
  } as React.CSSProperties,

  channelItem: (selected: boolean): React.CSSProperties => ({
    display: "flex",
    alignItems: "center",
    gap: 6,
    padding: "6px 8px",
    margin: "1px 8px",
    borderRadius: 4,
    cursor: "pointer",
    background: selected ? "#42464d" : "transparent",
    color: selected ? "#dcddde" : "#8e9297",
    fontSize: 15,
    fontWeight: selected ? 600 : 400,
    transition: "background 0.1s, color 0.1s",
    userSelect: "none" as const,
  }),

  channelIcon: {
    fontSize: 16,
    flexShrink: 0,
    width: 20,
    textAlign: "center" as const,
  } as React.CSSProperties,

  channelName: {
    overflow: "hidden",
    textOverflow: "ellipsis",
    whiteSpace: "nowrap" as const,
  } as React.CSSProperties,

  empty: {
    padding: "16px",
    color: "#72767d",
    fontSize: 13,
  } as React.CSSProperties,

  userArea: {
    marginTop: "auto",
    padding: "8px",
    background: "#292b2f",
    display: "flex",
    alignItems: "center",
    gap: 8,
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

  username: {
    fontSize: 14,
    fontWeight: 600,
    color: "#fff",
    overflow: "hidden",
    textOverflow: "ellipsis",
    whiteSpace: "nowrap" as const,
  } as React.CSSProperties,
};

export default function ChannelList({
  guild,
  channels,
  selectedChannelId,
  onSelect,
  currentUserId: _currentUserId,
}: ChannelListProps) {
  const textChannels = channels.filter((c) => c.type === "text");
  const voiceChannels = channels.filter((c) => c.type === "voice");

  if (!guild) {
    return (
      <div style={styles.sidebar}>
        <div style={styles.empty}>Выберите сервер</div>
      </div>
    );
  }

  return (
    <div style={styles.sidebar}>
      <div style={styles.header}>{guild.name}</div>

      {textChannels.length > 0 && (
        <>
          <div style={styles.section}>Текстовые каналы</div>
          {textChannels.map((ch) => (
            <div
              key={ch.id}
              style={styles.channelItem(ch.id === selectedChannelId)}
              onClick={() => onSelect(ch.id)}
            >
              <span style={styles.channelIcon}>#</span>
              <span style={styles.channelName}>{ch.name}</span>
            </div>
          ))}
        </>
      )}

      {voiceChannels.length > 0 && (
        <>
          <div style={styles.section}>Голосовые каналы</div>
          {voiceChannels.map((ch) => (
            <div
              key={ch.id}
              style={styles.channelItem(ch.id === selectedChannelId)}
              onClick={() => onSelect(ch.id)}
            >
              <span style={styles.channelIcon}>🔊</span>
              <span style={styles.channelName}>{ch.name}</span>
            </div>
          ))}
        </>
      )}

      {channels.length === 0 && (
        <div style={styles.empty}>Нет каналов</div>
      )}
    </div>
  );
}
