import { useCallback, useEffect, useState } from "react";
import type React from "react";
import {
  addCustomServer,
  checkServerHealth,
  getSelectedServer,
  serverFromApiBase,
  setSelectedServer,
  type ZenthrilServer,
} from "../config/servers";
import { reloadServerPool } from "../api/index";
import { disconnectGlobalWS, connectGlobalWS } from "../store/wsGlobal";
import {
  loadTransportPolicy,
  policyForConnectivityMode,
  saveTransportPolicy,
  type ConnectivityMode,
} from "../transport/connectivityPolicy";

interface ServerSettingsProps {
  onClose: () => void;
}

type HealthState = "unknown" | "checking" | "online" | "offline";

export default function ServerSettings({ onClose }: ServerSettingsProps) {
  const [servers, setServers] = useState<ZenthrilServer[]>([]);
  const [selectedID, setSelectedID] = useState<string>("");
  const [customURL, setCustomURL] = useState("");
  const [customName, setCustomName] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [health, setHealth] = useState<Record<string, HealthState>>({});
  const [connectivityMode, setConnectivityMode] = useState<ConnectivityMode>(() => loadTransportPolicy().connectivityMode);

  const refresh = useCallback(async () => {
    const loaded = await reloadServerPool();
    setServers(loaded);
    setSelectedID(getSelectedServer(loaded).id);
  }, []);

  useEffect(() => {
    refresh().catch(() => setServers([]));
  }, [refresh]);

  const chooseServer = useCallback(async (server: ZenthrilServer) => {
    setSelectedServer(server.id);
    setSelectedID(server.id);
    disconnectGlobalWS();
    // CONNECTIVITY: switching servers reconnects realtime traffic without requiring app restart.
    await connectGlobalWS().catch(() => {});
  }, []);

  const addServer = useCallback(async () => {
    setError(null);
    try {
      const server = serverFromApiBase(customName, customURL);
      addCustomServer(server);
      await refresh();
      await chooseServer(server);
      setCustomURL("");
      setCustomName("");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Invalid server URL");
    }
  }, [chooseServer, customName, customURL, refresh]);

  const checkOne = useCallback(async (server: ZenthrilServer) => {
    setHealth(prev => ({ ...prev, [server.id]: "checking" }));
    const ok = await checkServerHealth(server);
    setHealth(prev => ({ ...prev, [server.id]: ok ? "online" : "offline" }));
  }, []);

  const changeConnectivityMode = useCallback((mode: ConnectivityMode) => {
    setConnectivityMode(mode);
    saveTransportPolicy(policyForConnectivityMode(mode));
  }, []);

  return (
    <div style={s.backdrop} onMouseDown={onClose}>
      <div style={s.modal} onMouseDown={e => e.stopPropagation()}>
        <div style={s.header}>
          <div>
            <div style={s.title}>Servers</div>
            <div style={s.subtitle}>Change server or add a custom backup endpoint</div>
          </div>
          <button style={s.close} onClick={onClose}>x</button>
        </div>

        <div style={s.list}>
          {servers.map(server => (
            <div key={server.id} style={s.row(server.id === selectedID)}>
              <div style={{ minWidth: 0 }}>
                <div style={s.name}>{server.name}</div>
                <div style={s.url}>{server.apiBase}</div>
                <div style={s.meta}>
                  {server.custom ? "Custom" : "Listed"} · {health[server.id] ?? "unknown"}
                </div>
              </div>
              <div style={s.actions}>
                <button style={s.secondaryButton} onClick={() => checkOne(server)}>
                  Check
                </button>
                <button style={s.primaryButton} onClick={() => chooseServer(server)}>
                  Use
                </button>
              </div>
            </div>
          ))}
        </div>

        <div style={s.connectivityPanel}>
          <div>
            <div style={s.name}>Connectivity Mode</div>
            <div style={s.meta}>Optional JSON padding and retry timing controls</div>
          </div>
          <div style={s.segmented}>
            {(["off", "balanced", "strict"] as ConnectivityMode[]).map(mode => (
              <button
                key={mode}
                style={s.segment(connectivityMode === mode)}
                onClick={() => changeConnectivityMode(mode)}
              >
                {mode}
              </button>
            ))}
          </div>
        </div>

        <div style={s.form}>
          <input
            style={s.input}
            value={customName}
            onChange={e => setCustomName(e.target.value)}
            placeholder="Server name"
          />
          <input
            style={s.input}
            value={customURL}
            onChange={e => setCustomURL(e.target.value)}
            placeholder="https://backup.example.com"
          />
          {error && <div style={s.error}>{error}</div>}
          <button style={s.addButton} onClick={addServer}>Add Custom Server</button>
        </div>
      </div>
    </div>
  );
}

const s = {
  backdrop: {
    position: "fixed",
    inset: 0,
    zIndex: 10000,
    background: "rgba(0,0,0,0.55)",
    display: "flex",
    alignItems: "center",
    justifyContent: "center",
    padding: 20,
  } as React.CSSProperties,
  modal: {
    width: "min(560px, 100%)",
    maxHeight: "86vh",
    overflow: "auto",
    background: "var(--bg-elevated)",
    border: "1px solid var(--border)",
    borderRadius: 12,
    boxShadow: "var(--shadow-lg)",
    padding: 18,
  } as React.CSSProperties,
  header: {
    display: "flex",
    justifyContent: "space-between",
    alignItems: "flex-start",
    marginBottom: 14,
  } as React.CSSProperties,
  title: { fontSize: 18, fontWeight: 800, color: "var(--text-primary)" } as React.CSSProperties,
  subtitle: { fontSize: 12, color: "var(--text-muted)", marginTop: 3 } as React.CSSProperties,
  close: {
    border: "1px solid var(--border)",
    background: "var(--bg-input)",
    color: "var(--text-secondary)",
    borderRadius: 8,
    width: 30,
    height: 30,
    cursor: "pointer",
  } as React.CSSProperties,
  list: { display: "flex", flexDirection: "column", gap: 8 } as React.CSSProperties,
  connectivityPanel: {
    marginTop: 14,
    padding: 12,
    borderRadius: 10,
    border: "1px solid var(--border)",
    background: "var(--bg-input)",
    display: "flex",
    justifyContent: "space-between",
    gap: 12,
    alignItems: "center",
  } as React.CSSProperties,
  segmented: {
    display: "grid",
    gridTemplateColumns: "repeat(3, minmax(0, 1fr))",
    gap: 4,
    minWidth: 210,
  } as React.CSSProperties,
  segment: (active: boolean): React.CSSProperties => ({
    border: "1px solid var(--border)",
    borderRadius: 7,
    padding: "7px 8px",
    background: active ? "var(--accent)" : "transparent",
    color: active ? "#fff" : "var(--text-secondary)",
    cursor: "pointer",
    textTransform: "capitalize",
    fontSize: 12,
    fontWeight: 700,
  }),
  row: (active: boolean): React.CSSProperties => ({
    display: "flex",
    justifyContent: "space-between",
    gap: 12,
    padding: 12,
    borderRadius: 10,
    border: active ? "1px solid var(--accent)" : "1px solid var(--border)",
    background: active ? "rgba(124,106,247,0.12)" : "var(--bg-input)",
  }),
  name: { fontSize: 13, fontWeight: 700, color: "var(--text-primary)" } as React.CSSProperties,
  url: {
    fontSize: 12,
    color: "var(--text-secondary)",
    overflow: "hidden",
    textOverflow: "ellipsis",
    whiteSpace: "nowrap",
    maxWidth: 320,
  } as React.CSSProperties,
  meta: { fontSize: 11, color: "var(--text-muted)", marginTop: 4 } as React.CSSProperties,
  actions: { display: "flex", alignItems: "center", gap: 6 } as React.CSSProperties,
  primaryButton: {
    border: "none",
    borderRadius: 8,
    padding: "7px 10px",
    background: "var(--accent)",
    color: "#fff",
    cursor: "pointer",
    fontWeight: 700,
  } as React.CSSProperties,
  secondaryButton: {
    border: "1px solid var(--border)",
    borderRadius: 8,
    padding: "7px 10px",
    background: "transparent",
    color: "var(--text-secondary)",
    cursor: "pointer",
  } as React.CSSProperties,
  form: {
    marginTop: 16,
    display: "flex",
    flexDirection: "column",
    gap: 8,
    borderTop: "1px solid var(--border)",
    paddingTop: 14,
  } as React.CSSProperties,
  input: {
    background: "var(--bg-input)",
    border: "1px solid var(--border)",
    borderRadius: 8,
    color: "var(--text-primary)",
    padding: "10px 12px",
    outline: "none",
  } as React.CSSProperties,
  addButton: {
    border: "none",
    borderRadius: 8,
    padding: "10px 12px",
    background: "linear-gradient(135deg, #7c6af7, #a78bfa)",
    color: "#fff",
    cursor: "pointer",
    fontWeight: 800,
  } as React.CSSProperties,
  error: { color: "#f04f5e", fontSize: 12 } as React.CSSProperties,
};
