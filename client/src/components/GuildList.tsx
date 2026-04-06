/**
 * GuildList — список серверов слева
 * Требования: 2.2
 */

import React, { useState } from "react";
import type { GuildAPI } from "../api/index";

interface GuildListProps {
  guilds: GuildAPI[];
  selectedGuildId: string | null;
  onSelect: (guildId: string) => void;
  onCreateGuild: (name: string) => Promise<void>;
}

const styles = {
  sidebar: {
    width: 72,
    background: "#202225",
    display: "flex",
    flexDirection: "column" as const,
    alignItems: "center",
    padding: "12px 0",
    gap: 8,
    overflowY: "auto" as const,
    flexShrink: 0,
  } as React.CSSProperties,

  icon: (selected: boolean): React.CSSProperties => ({
    width: 48,
    height: 48,
    borderRadius: selected ? 16 : "50%",
    background: selected ? "#7289da" : "#36393f",
    display: "flex",
    alignItems: "center",
    justifyContent: "center",
    cursor: "pointer",
    fontSize: 18,
    fontWeight: 700,
    color: selected ? "#fff" : "#dcddde",
    transition: "border-radius 0.15s, background 0.15s",
    userSelect: "none" as const,
    flexShrink: 0,
    position: "relative" as const,
  }),

  divider: {
    width: 32,
    height: 2,
    background: "#36393f",
    borderRadius: 1,
    margin: "4px 0",
  } as React.CSSProperties,

  addBtn: {
    width: 48,
    height: 48,
    borderRadius: "50%",
    background: "#36393f",
    border: "none",
    display: "flex",
    alignItems: "center",
    justifyContent: "center",
    cursor: "pointer",
    fontSize: 24,
    color: "#43b581",
    transition: "border-radius 0.15s, background 0.15s",
    flexShrink: 0,
  } as React.CSSProperties,

  tooltip: {
    position: "absolute" as const,
    left: 60,
    background: "#18191c",
    color: "#fff",
    padding: "6px 10px",
    borderRadius: 4,
    fontSize: 13,
    fontWeight: 600,
    whiteSpace: "nowrap" as const,
    pointerEvents: "none" as const,
    zIndex: 100,
    boxShadow: "0 4px 12px rgba(0,0,0,0.4)",
  } as React.CSSProperties,

  modal: {
    position: "fixed" as const,
    inset: 0,
    background: "rgba(0,0,0,0.7)",
    display: "flex",
    alignItems: "center",
    justifyContent: "center",
    zIndex: 200,
  } as React.CSSProperties,

  modalCard: {
    background: "#36393f",
    borderRadius: 8,
    padding: "32px",
    width: 360,
    boxShadow: "0 8px 32px rgba(0,0,0,0.5)",
  } as React.CSSProperties,

  modalTitle: {
    fontSize: 20,
    fontWeight: 700,
    color: "#fff",
    marginBottom: 8,
  } as React.CSSProperties,

  modalSubtitle: {
    fontSize: 14,
    color: "#b9bbbe",
    marginBottom: 20,
  } as React.CSSProperties,

  modalLabel: {
    display: "block",
    fontSize: 12,
    fontWeight: 700,
    color: "#b9bbbe",
    textTransform: "uppercase" as const,
    letterSpacing: 0.5,
    marginBottom: 8,
  } as React.CSSProperties,

  modalInput: {
    width: "100%",
    padding: "10px 12px",
    background: "#202225",
    border: "1px solid #040405",
    borderRadius: 4,
    color: "#dcddde",
    fontSize: 16,
    outline: "none",
    boxSizing: "border-box" as const,
  } as React.CSSProperties,

  modalActions: {
    display: "flex",
    justifyContent: "flex-end",
    gap: 8,
    marginTop: 20,
  } as React.CSSProperties,

  btnCancel: {
    padding: "10px 16px",
    background: "none",
    border: "none",
    color: "#b9bbbe",
    cursor: "pointer",
    fontSize: 14,
    fontWeight: 600,
    borderRadius: 4,
  } as React.CSSProperties,

  btnCreate: {
    padding: "10px 20px",
    background: "#7289da",
    border: "none",
    color: "#fff",
    cursor: "pointer",
    fontSize: 14,
    fontWeight: 600,
    borderRadius: 4,
  } as React.CSSProperties,
};

function GuildIcon({
  guild,
  selected,
  onClick,
}: {
  guild: GuildAPI;
  selected: boolean;
  onClick: () => void;
}) {
  const [hovered, setHovered] = useState(false);
  const letter = guild.name.charAt(0).toUpperCase();

  return (
    <div style={{ position: "relative" }}>
      <div
        style={styles.icon(selected)}
        onClick={onClick}
        onMouseEnter={() => setHovered(true)}
        onMouseLeave={() => setHovered(false)}
        title={guild.name}
      >
        {letter}
      </div>
      {hovered && <div style={styles.tooltip}>{guild.name}</div>}
    </div>
  );
}

export default function GuildList({
  guilds,
  selectedGuildId,
  onSelect,
  onCreateGuild,
}: GuildListProps) {
  const [showModal, setShowModal] = useState(false);
  const [newName, setNewName] = useState("");
  const [creating, setCreating] = useState(false);

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!newName.trim()) return;
    setCreating(true);
    try {
      await onCreateGuild(newName.trim());
      setNewName("");
      setShowModal(false);
    } finally {
      setCreating(false);
    }
  };

  return (
    <>
      <div style={styles.sidebar}>
        {guilds.map((g) => (
          <GuildIcon
            key={g.id}
            guild={g}
            selected={g.id === selectedGuildId}
            onClick={() => onSelect(g.id)}
          />
        ))}

        {guilds.length > 0 && <div style={styles.divider} />}

        <button
          style={styles.addBtn}
          onClick={() => setShowModal(true)}
          title="Создать сервер"
        >
          +
        </button>
      </div>

      {showModal && (
        <div style={styles.modal} onClick={() => setShowModal(false)}>
          <div style={styles.modalCard} onClick={(e) => e.stopPropagation()}>
            <div style={styles.modalTitle}>Создать сервер</div>
            <div style={styles.modalSubtitle}>
              Дайте вашему серверу имя. Его можно изменить позже.
            </div>
            <form onSubmit={handleCreate}>
              <label style={styles.modalLabel} htmlFor="guild-name">
                Название сервера
              </label>
              <input
                id="guild-name"
                style={styles.modalInput}
                type="text"
                value={newName}
                onChange={(e) => setNewName(e.target.value)}
                placeholder="Мой сервер"
                maxLength={100}
                autoFocus
                required
              />
              <div style={styles.modalActions}>
                <button
                  type="button"
                  style={styles.btnCancel}
                  onClick={() => setShowModal(false)}
                >
                  Отмена
                </button>
                <button
                  type="submit"
                  style={styles.btnCreate}
                  disabled={creating}
                >
                  {creating ? "Создание..." : "Создать"}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </>
  );
}
