import { useCallback, useEffect, useMemo, useState } from "react";
import type React from "react";
import {
  CheckCircle2,
  Copy,
  KeyRound,
  Laptop,
  RefreshCw,
  ShieldAlert,
  Trash2,
  X,
} from "lucide-react";
import { api } from "../api";
import { useAuth } from "../store/auth";
import {
  ensureLocalDeviceRegistered,
  getDeviceKeyStorageStatus,
  loadDeviceKeyBundle,
  storeDeviceKeyBundle,
} from "../features/e2ee";
import type { DeviceAPI, StoredDeviceKeyBundle } from "../features/e2ee";

interface Props {
  onClose: () => void;
}

type LoadState = "idle" | "loading" | "ready" | "error";

export default function DeviceManagement({ onClose }: Props) {
  const { user } = useAuth();
  const [devices, setDevices] = useState<DeviceAPI[]>([]);
  const [localBundle, setLocalBundle] = useState<StoredDeviceKeyBundle | null>(null);
  const [state, setState] = useState<LoadState>("idle");
  const [error, setError] = useState<string | null>(null);
  const [busyDeviceId, setBusyDeviceId] = useState<string | null>(null);
  const [syncing, setSyncing] = useState(false);
  const [copied, setCopied] = useState<string | null>(null);
  const storageStatus = useMemo(() => getDeviceKeyStorageStatus(), []);

  const currentDevice = useMemo(() => {
    if (!localBundle) return null;
    return devices.find(device => device.device_id === localBundle.deviceId) ?? null;
  }, [devices, localBundle]);

  const reload = useCallback(async () => {
    if (!user) return;
    setState("loading");
    setError(null);
    try {
      const [bundle, response] = await Promise.all([
        loadDeviceKeyBundle(user.id),
        api.devices.listOwn(),
      ]);
      setLocalBundle(bundle);
      setDevices(response.devices);
      setState("ready");
    } catch (err) {
      setState("error");
      setError(err instanceof Error ? err.message : "Не удалось загрузить устройства");
    }
  }, [user]);

  useEffect(() => {
    void reload();
  }, [reload]);

  async function handleRegisterCurrentDevice() {
    if (!user) return;
    setSyncing(true);
    setError(null);
    try {
      const bundle = await ensureLocalDeviceRegistered(user.id, undefined, { force: true });
      setLocalBundle(bundle);
      await reload();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Не удалось зарегистрировать устройство");
    } finally {
      setSyncing(false);
    }
  }

  async function handleRevoke(device: DeviceAPI) {
    if (!user || device.device_id === localBundle?.deviceId) return;
    const ok = window.confirm(`Отозвать устройство "${device.name || "Zenthril device"}"?`);
    if (!ok) return;
    setBusyDeviceId(device.device_id);
    setError(null);
    try {
      await api.devices.revoke(device.device_id);
      setDevices(prev => prev.filter(item => item.device_id !== device.device_id));
    } catch (err) {
      setError(err instanceof Error ? err.message : "Не удалось отозвать устройство");
    } finally {
      setBusyDeviceId(null);
    }
  }

  async function handleRenameLocalDevice() {
    if (!user || !localBundle) return;
    const nextName = window.prompt("Название текущего устройства", localBundle.deviceName);
    if (!nextName || nextName.trim() === localBundle.deviceName) return;
    setSyncing(true);
    setError(null);
    try {
      const updated = { ...localBundle, deviceName: nextName.trim().slice(0, 100) };
      await storeDeviceKeyBundle(updated);
      await ensureLocalDeviceRegistered(user.id, undefined, { force: true });
      await reload();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Не удалось переименовать устройство");
    } finally {
      setSyncing(false);
    }
  }

  async function copyText(label: string, value: string) {
    try {
      await navigator.clipboard.writeText(value);
      setCopied(label);
      window.setTimeout(() => setCopied(null), 1400);
    } catch {
      setCopied(null);
    }
  }

  const missingLocalRegistration = !!localBundle && !currentDevice && state === "ready";

  return (
    <div style={s.overlay} onClick={onClose}>
      <div style={s.modal} onClick={event => event.stopPropagation()}>
        <header style={s.header}>
          <div style={s.titleWrap}>
            <div style={s.titleIcon}><KeyRound size={18} /></div>
            <div>
              <div style={s.title}>Устройства и ключи</div>
              <div style={s.subtitle}>{devices.length} активных устройств</div>
            </div>
          </div>
          <div style={s.headerActions}>
            <IconButton title="Обновить" onClick={() => void reload()} disabled={state === "loading"}>
              <RefreshCw size={16} />
            </IconButton>
            <IconButton title="Закрыть" onClick={onClose}>
              <X size={17} />
            </IconButton>
          </div>
        </header>

        {error && (
          <div style={s.error}>
            <ShieldAlert size={15} />
            <span>{error}</span>
          </div>
        )}

        {storageStatus.warning && (
          <div style={s.warning}>
            <ShieldAlert size={16} />
            <div style={{ flex: 1 }}>
              <div style={s.warningTitle}>Key storage status: {storageStatus.backend}</div>
              <div style={s.warningText}>{storageStatus.warning}</div>
            </div>
          </div>
        )}

        {missingLocalRegistration && (
          <div style={s.warning}>
            <ShieldAlert size={16} />
            <div style={{ flex: 1 }}>
              <div style={s.warningTitle}>Текущее устройство не найдено на сервере</div>
              <div style={s.warningText}>Ключи есть локально, но backend не показывает активную запись.</div>
            </div>
            <button
              type="button"
              onClick={() => void handleRegisterCurrentDevice()}
              disabled={syncing}
              style={s.primaryButton}
            >
              {syncing ? "Синхронизация..." : "Синхронизировать"}
            </button>
          </div>
        )}

        {!localBundle && state === "ready" && (
          <div style={s.warning}>
            <ShieldAlert size={16} />
            <div style={{ flex: 1 }}>
              <div style={s.warningTitle}>Локальный device key bundle отсутствует</div>
              <div style={s.warningText}>Можно создать и зарегистрировать ключи для этого клиента.</div>
            </div>
            <button
              type="button"
              onClick={() => void handleRegisterCurrentDevice()}
              disabled={syncing}
              style={s.primaryButton}
            >
              {syncing ? "Создание..." : "Создать"}
            </button>
          </div>
        )}

        <section style={s.list}>
          {state === "loading" && <div style={s.empty}>Загрузка устройств...</div>}
          {state !== "loading" && devices.length === 0 && (
            <div style={s.empty}>Активных устройств пока нет.</div>
          )}
          {devices.map(device => {
            const isCurrent = device.device_id === localBundle?.deviceId;
            return (
              <article key={device.device_id} style={s.deviceRow}>
                <div style={s.deviceIcon}>
                  <Laptop size={20} />
                </div>
                <div style={s.deviceMain}>
                  <div style={s.deviceTop}>
                    <div style={s.deviceName}>{device.name || "Zenthril device"}</div>
                    {isCurrent && (
                      <span style={s.currentBadge}>
                        <CheckCircle2 size={12} />
                        Текущее
                      </span>
                    )}
                  </div>
                  <div style={s.metaGrid}>
                    <Meta label="Trust" value={trustLabel(device.trust_state)} />
                    <Meta label="Prekeys" value={String(device.one_time_prekey_count)} />
                    <Meta label="Created" value={formatDate(device.created_at)} />
                    <Meta label="Last seen" value={formatDate(device.last_seen_at)} />
                  </div>
                  <KeyLine
                    label="Fingerprint"
                    value={formatFingerprint(device.fingerprint)}
                    onCopy={() => void copyText(`fingerprint:${device.device_id}`, device.fingerprint)}
                    copied={copied === `fingerprint:${device.device_id}`}
                  />
                  <KeyLine
                    label="Identity"
                    value={shortKey(device.identity_public_key)}
                    onCopy={() => void copyText(`identity:${device.device_id}`, device.identity_public_key)}
                    copied={copied === `identity:${device.device_id}`}
                  />
                </div>
                <div style={s.deviceActions}>
                  {isCurrent ? (
                    <button
                      type="button"
                      onClick={() => void handleRenameLocalDevice()}
                      disabled={syncing}
                      style={s.secondaryButton}
                    >
                      {syncing ? "..." : "Переименовать"}
                    </button>
                  ) : (
                    <IconButton
                      title="Отозвать устройство"
                      onClick={() => void handleRevoke(device)}
                      disabled={busyDeviceId === device.device_id}
                      danger
                    >
                      <Trash2 size={16} />
                    </IconButton>
                  )}
                </div>
              </article>
            );
          })}
        </section>
      </div>
    </div>
  );
}

function Meta({ label, value }: { label: string; value: string }) {
  return (
    <div style={s.metaItem}>
      <span style={s.metaLabel}>{label}</span>
      <span style={s.metaValue}>{value}</span>
    </div>
  );
}

function KeyLine({
  label,
  value,
  onCopy,
  copied,
}: {
  label: string;
  value: string;
  onCopy: () => void;
  copied: boolean;
}) {
  return (
    <div style={s.keyLine}>
      <span style={s.keyLabel}>{label}</span>
      <code style={s.keyValue}>{value}</code>
      <button type="button" title="Скопировать" onClick={onCopy} style={s.copyButton}>
        {copied ? <CheckCircle2 size={14} /> : <Copy size={14} />}
      </button>
    </div>
  );
}

function IconButton({
  children,
  title,
  onClick,
  disabled,
  danger,
}: {
  children: React.ReactNode;
  title: string;
  onClick: () => void;
  disabled?: boolean;
  danger?: boolean;
}) {
  return (
    <button
      type="button"
      title={title}
      onClick={onClick}
      disabled={disabled}
      style={{
        ...s.iconButton,
        color: danger ? "#f04f5e" : "var(--text-secondary)",
        opacity: disabled ? 0.5 : 1,
      }}
    >
      {children}
    </button>
  );
}

function trustLabel(value: DeviceAPI["trust_state"]): string {
  if (value === "verified") return "Verified";
  if (value === "revoked") return "Revoked";
  return "Unverified";
}

function formatDate(value: string): string {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "unknown";
  return new Intl.DateTimeFormat("ru", {
    day: "2-digit",
    month: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
  }).format(date);
}

function formatFingerprint(value: string): string {
  return value.match(/.{1,8}/g)?.slice(0, 4).join(" ") ?? value;
}

function shortKey(value: string): string {
  if (value.length <= 28) return value;
  return `${value.slice(0, 14)}...${value.slice(-10)}`;
}

const s = {
  overlay: {
    position: "fixed",
    inset: 0,
    background: "rgba(0,0,0,0.72)",
    zIndex: 900,
    display: "flex",
    alignItems: "center",
    justifyContent: "center",
    padding: 18,
  } as React.CSSProperties,
  modal: {
    width: "min(760px, 100%)",
    maxHeight: "min(760px, calc(100vh - 36px))",
    overflow: "hidden",
    background: "var(--bg-elevated)",
    border: "1px solid var(--border)",
    borderRadius: 8,
    boxShadow: "0 18px 60px rgba(0,0,0,0.45)",
    display: "flex",
    flexDirection: "column",
  } as React.CSSProperties,
  header: {
    display: "flex",
    alignItems: "center",
    justifyContent: "space-between",
    gap: 12,
    padding: "16px 18px",
    borderBottom: "1px solid var(--border)",
    flexShrink: 0,
  } as React.CSSProperties,
  titleWrap: {
    display: "flex",
    alignItems: "center",
    gap: 12,
    minWidth: 0,
  } as React.CSSProperties,
  titleIcon: {
    width: 34,
    height: 34,
    borderRadius: 8,
    display: "flex",
    alignItems: "center",
    justifyContent: "center",
    background: "rgba(139,157,255,0.16)",
    color: "var(--accent)",
    flexShrink: 0,
  } as React.CSSProperties,
  title: {
    fontSize: 16,
    fontWeight: 800,
    color: "var(--text-primary)",
  } as React.CSSProperties,
  subtitle: {
    fontSize: 12,
    color: "var(--text-muted)",
    marginTop: 2,
  } as React.CSSProperties,
  headerActions: {
    display: "flex",
    gap: 6,
    flexShrink: 0,
  } as React.CSSProperties,
  iconButton: {
    width: 32,
    height: 32,
    borderRadius: 8,
    border: "1px solid var(--border)",
    background: "var(--bg-input)",
    display: "inline-flex",
    alignItems: "center",
    justifyContent: "center",
    cursor: "pointer",
  } as React.CSSProperties,
  error: {
    margin: "12px 18px 0",
    padding: "10px 12px",
    borderRadius: 8,
    border: "1px solid rgba(240,79,94,0.35)",
    background: "rgba(240,79,94,0.12)",
    color: "#ff8d98",
    fontSize: 12,
    display: "flex",
    gap: 8,
    alignItems: "center",
  } as React.CSSProperties,
  warning: {
    margin: "12px 18px 0",
    padding: 12,
    borderRadius: 8,
    border: "1px solid rgba(255,190,92,0.28)",
    background: "rgba(255,190,92,0.1)",
    display: "flex",
    alignItems: "center",
    flexWrap: "wrap",
    gap: 12,
    color: "#ffc875",
  } as React.CSSProperties,
  warningTitle: {
    fontSize: 13,
    fontWeight: 700,
    color: "var(--text-primary)",
  } as React.CSSProperties,
  warningText: {
    fontSize: 12,
    color: "var(--text-muted)",
    marginTop: 2,
  } as React.CSSProperties,
  list: {
    padding: 18,
    overflow: "auto",
    display: "flex",
    flexDirection: "column",
    gap: 10,
  } as React.CSSProperties,
  empty: {
    padding: 28,
    textAlign: "center",
    color: "var(--text-muted)",
    fontSize: 13,
    border: "1px dashed var(--border)",
    borderRadius: 8,
  } as React.CSSProperties,
  deviceRow: {
    display: "grid",
    gridTemplateColumns: "40px minmax(0, 1fr) auto",
    gap: 12,
    alignItems: "start",
    padding: 14,
    border: "1px solid var(--border)",
    borderRadius: 8,
    background: "var(--bg-surface)",
  } as React.CSSProperties,
  deviceIcon: {
    width: 40,
    height: 40,
    borderRadius: 8,
    background: "rgba(168,255,218,0.12)",
    color: "#A8FFDA",
    display: "flex",
    alignItems: "center",
    justifyContent: "center",
  } as React.CSSProperties,
  deviceMain: {
    minWidth: 0,
  } as React.CSSProperties,
  deviceTop: {
    display: "flex",
    alignItems: "center",
    gap: 8,
    minWidth: 0,
    marginBottom: 10,
  } as React.CSSProperties,
  deviceName: {
    fontSize: 14,
    fontWeight: 750,
    color: "var(--text-primary)",
    overflow: "hidden",
    textOverflow: "ellipsis",
    whiteSpace: "nowrap",
  } as React.CSSProperties,
  currentBadge: {
    display: "inline-flex",
    alignItems: "center",
    gap: 4,
    padding: "3px 7px",
    borderRadius: 8,
    background: "rgba(62,207,142,0.13)",
    color: "#3ecf8e",
    fontSize: 11,
    fontWeight: 700,
    flexShrink: 0,
  } as React.CSSProperties,
  metaGrid: {
    display: "grid",
    gridTemplateColumns: "repeat(auto-fit, minmax(86px, 1fr))",
    gap: 8,
    marginBottom: 10,
  } as React.CSSProperties,
  metaItem: {
    minWidth: 0,
  } as React.CSSProperties,
  metaLabel: {
    display: "block",
    fontSize: 10,
    color: "var(--text-muted)",
    textTransform: "uppercase",
    letterSpacing: 0,
    fontWeight: 700,
  } as React.CSSProperties,
  metaValue: {
    display: "block",
    fontSize: 12,
    color: "var(--text-secondary)",
    marginTop: 2,
    overflow: "hidden",
    textOverflow: "ellipsis",
    whiteSpace: "nowrap",
  } as React.CSSProperties,
  keyLine: {
    display: "grid",
    gridTemplateColumns: "82px minmax(0, 1fr) 28px",
    alignItems: "center",
    gap: 8,
    minHeight: 30,
  } as React.CSSProperties,
  keyLabel: {
    fontSize: 11,
    color: "var(--text-muted)",
    fontWeight: 700,
  } as React.CSSProperties,
  keyValue: {
    fontFamily: "ui-monospace, SFMono-Regular, Menlo, Consolas, monospace",
    fontSize: 11,
    color: "var(--text-secondary)",
    background: "var(--bg-input)",
    border: "1px solid var(--border)",
    borderRadius: 6,
    padding: "5px 7px",
    overflow: "hidden",
    textOverflow: "ellipsis",
    whiteSpace: "nowrap",
  } as React.CSSProperties,
  copyButton: {
    width: 28,
    height: 28,
    borderRadius: 7,
    border: "1px solid var(--border)",
    background: "var(--bg-input)",
    color: "var(--text-muted)",
    display: "inline-flex",
    alignItems: "center",
    justifyContent: "center",
    cursor: "pointer",
  } as React.CSSProperties,
  deviceActions: {
    display: "flex",
    alignItems: "center",
    gap: 6,
  } as React.CSSProperties,
  primaryButton: {
    padding: "8px 10px",
    borderRadius: 8,
    border: "none",
    background: "linear-gradient(135deg,var(--accent),var(--accent-hover))",
    color: "#fff",
    cursor: "pointer",
    fontSize: 12,
    fontWeight: 800,
    flexShrink: 0,
  } as React.CSSProperties,
  secondaryButton: {
    padding: "8px 10px",
    borderRadius: 8,
    border: "1px solid var(--border)",
    background: "var(--bg-input)",
    color: "var(--text-secondary)",
    cursor: "pointer",
    fontSize: 12,
    fontWeight: 700,
    whiteSpace: "nowrap",
  } as React.CSSProperties,
};
