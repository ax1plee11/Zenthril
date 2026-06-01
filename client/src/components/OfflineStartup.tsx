import { useState } from "react";
import type React from "react";
import {
  loadAutoConnectOnStartup,
  saveAutoConnectOnStartup,
} from "../store/startupPrivacy";

interface OfflineStartupProps {
  username: string;
  onConnect: () => void;
  onLogout: () => void;
}

export default function OfflineStartup({ username, onConnect, onLogout }: OfflineStartupProps) {
  const [autoConnect, setAutoConnect] = useState(loadAutoConnectOnStartup);

  function toggleAutoConnect(next: boolean): void {
    setAutoConnect(next);
    saveAutoConnectOnStartup(next);
  }

  return (
    <div style={s.wrap}>
      <div style={s.panel}>
        <div style={s.eyebrow}>Privacy-first startup</div>
        <h1 style={s.title}>Zenthril is offline</h1>
        <p style={s.body}>
          Signed in as <strong>{username}</strong>. The app has not contacted the
          server yet. Connect when you are ready to load guilds, messages, and
          realtime events.
        </p>
        <p style={s.notice}>
          Zenthril minimizes startup network activity and encrypts message
          contents. Network providers may still observe metadata such as
          destination IP addresses, domains, timing, and traffic volume once you
          connect.
        </p>
        <label style={s.checkRow}>
          <input
            type="checkbox"
            checked={autoConnect}
            onChange={event => toggleAutoConnect(event.currentTarget.checked)}
          />
          <span>Connect automatically on startup</span>
        </label>
        <div style={s.actions}>
          <button style={s.primary} onClick={onConnect}>Connect</button>
          <button style={s.secondary} onClick={onLogout}>Log out</button>
        </div>
      </div>
    </div>
  );
}

const s = {
  wrap: {
    height: "100%",
    display: "flex",
    alignItems: "center",
    justifyContent: "center",
    padding: 24,
  } as React.CSSProperties,
  panel: {
    width: "min(520px, 100%)",
    border: "1px solid var(--border)",
    borderRadius: 12,
    background: "var(--bg-elevated)",
    boxShadow: "var(--shadow-lg)",
    padding: 24,
  } as React.CSSProperties,
  eyebrow: {
    color: "var(--accent)",
    fontSize: 12,
    fontWeight: 800,
    textTransform: "uppercase",
  } as React.CSSProperties,
  title: {
    margin: "8px 0 10px",
    fontSize: 28,
    color: "var(--text-primary)",
  } as React.CSSProperties,
  body: {
    color: "var(--text-secondary)",
    fontSize: 14,
    lineHeight: 1.55,
  } as React.CSSProperties,
  notice: {
    marginTop: 14,
    padding: 12,
    borderRadius: 8,
    border: "1px solid var(--border)",
    background: "var(--bg-input)",
    color: "var(--text-muted)",
    fontSize: 12,
    lineHeight: 1.5,
  } as React.CSSProperties,
  checkRow: {
    marginTop: 16,
    display: "flex",
    alignItems: "center",
    gap: 8,
    color: "var(--text-secondary)",
    fontSize: 13,
  } as React.CSSProperties,
  actions: {
    display: "flex",
    gap: 10,
    marginTop: 18,
  } as React.CSSProperties,
  primary: {
    border: "none",
    borderRadius: 8,
    padding: "10px 16px",
    background: "var(--accent)",
    color: "#fff",
    fontWeight: 800,
    cursor: "pointer",
  } as React.CSSProperties,
  secondary: {
    border: "1px solid var(--border)",
    borderRadius: 8,
    padding: "10px 16px",
    background: "transparent",
    color: "var(--text-secondary)",
    fontWeight: 700,
    cursor: "pointer",
  } as React.CSSProperties,
};

