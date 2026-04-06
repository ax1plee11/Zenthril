import React, { useState, useCallback } from "react";
import { api } from "../api/index";
import { saveAuth } from "../store/auth";
import { generateKeyPair, exportPublicKey, storePrivateKey } from "../crypto/index";

interface AuthScreenProps { onAuth: () => void; }
type Tab = "login" | "register";

export default function AuthScreen({ onAuth }: AuthScreenProps) {
  const [tab, setTab]           = useState<Tab>("login");
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError]       = useState<string | null>(null);
  const [loading, setLoading]   = useState(false);

  const handleSubmit = useCallback(async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);
    setLoading(true);
    try {
      if (tab === "register") {
        const keyPair    = generateKeyPair();
        const pubKeyB64  = exportPublicKey(keyPair.publicKey);
        const res        = await api.auth.register(username, password, pubKeyB64);
        storePrivateKey(keyPair.secretKey);
        saveAuth(res.token, { id: res.user_id, username, public_key: pubKeyB64 });
      } else {
        const res = await api.auth.login(username, password);
        saveAuth(res.token, { id: res.user.id, username: res.user.username, public_key: res.user.public_key });
      }
      onAuth();
    } catch {
      setError("Неверные данные. Попробуйте ещё раз.");
    } finally {
      setLoading(false);
    }
  }, [tab, username, password, onAuth]);

  return (
    <div style={s.root}>
      {/* Ambient blobs */}
      <div style={s.blob1} />
      <div style={s.blob2} />

      <div style={s.card} className="fade-up">
        {/* Logo */}
        <div style={s.logoWrap}>
          <div style={s.logoIcon}>
            <svg width="28" height="28" viewBox="0 0 28 28" fill="none">
              <path d="M14 2L26 8V20L14 26L2 20V8L14 2Z" fill="url(#lg)" stroke="rgba(255,255,255,0.1)" strokeWidth="0.5"/>
              <path d="M14 8L20 11V17L14 20L8 17V11L14 8Z" fill="rgba(255,255,255,0.15)"/>
              <defs>
                <linearGradient id="lg" x1="2" y1="2" x2="26" y2="26" gradientUnits="userSpaceOnUse">
                  <stop stopColor="#7c6af7"/>
                  <stop offset="1" stopColor="#a78bfa"/>
                </linearGradient>
              </defs>
            </svg>
          </div>
          <div style={s.logoName}>Zenthril</div>
          <div style={s.logoSub}>Decentralized · Encrypted · Free</div>
        </div>

        {/* Tabs */}
        <div style={s.tabs}>
          {(["login","register"] as Tab[]).map(t => (
            <button key={t} style={s.tab(tab === t)} onClick={() => { setTab(t); setError(null); }}>
              {t === "login" ? "Sign In" : "Create Account"}
            </button>
          ))}
          <div style={s.tabIndicator(tab === "register")} />
        </div>

        <form onSubmit={handleSubmit} style={s.form}>
          <div style={s.field}>
            <label style={s.label}>Username</label>
            <div style={s.inputWrap}>
              <span style={s.inputIcon}>
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                  <path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"/><circle cx="12" cy="7" r="4"/>
                </svg>
              </span>
              <input style={s.input} type="text" value={username}
                onChange={e => setUsername(e.target.value)}
                placeholder="your_username" required minLength={1} maxLength={32}
                autoComplete="username" />
            </div>
          </div>

          <div style={s.field}>
            <label style={s.label}>Password</label>
            <div style={s.inputWrap}>
              <span style={s.inputIcon}>
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                  <rect x="3" y="11" width="18" height="11" rx="2"/><path d="M7 11V7a5 5 0 0 1 10 0v4"/>
                </svg>
              </span>
              <input style={s.input} type="password" value={password}
                onChange={e => setPassword(e.target.value)}
                placeholder="••••••••" required minLength={1}
                autoComplete={tab === "register" ? "new-password" : "current-password"} />
            </div>
          </div>

          {error && (
            <div style={s.error} className="fade-in">
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" style={{flexShrink:0}}>
                <circle cx="12" cy="12" r="10"/><line x1="12" y1="8" x2="12" y2="12"/><line x1="12" y1="16" x2="12.01" y2="16"/>
              </svg>
              {error}
            </div>
          )}

          <button type="submit" style={s.btn(loading)} disabled={loading}>
            {loading ? <span style={s.spinner} /> : null}
            {loading ? "Please wait..." : tab === "login" ? "Sign In" : "Create Account"}
          </button>
        </form>

        <div style={s.footer}>
          End-to-end encrypted · Open source · No tracking
        </div>
      </div>
    </div>
  );
}

const s = {
  root: {
    display: "flex", alignItems: "center", justifyContent: "center",
    minHeight: "100vh", background: "var(--bg-base)", position: "relative" as const, overflow: "hidden",
  } as React.CSSProperties,
  blob1: {
    position: "absolute" as const, width: 500, height: 500, borderRadius: "50%",
    background: "radial-gradient(circle, rgba(124,106,247,0.12) 0%, transparent 70%)",
    top: -100, left: -100, pointerEvents: "none" as const,
  } as React.CSSProperties,
  blob2: {
    position: "absolute" as const, width: 400, height: 400, borderRadius: "50%",
    background: "radial-gradient(circle, rgba(167,139,250,0.08) 0%, transparent 70%)",
    bottom: -80, right: -80, pointerEvents: "none" as const,
  } as React.CSSProperties,
  card: {
    position: "relative" as const, width: 420, padding: "40px 36px",
    background: "var(--bg-surface)",
    border: "1px solid var(--border)",
    borderRadius: "var(--radius-xl)",
    boxShadow: "var(--shadow-lg), inset 0 1px 0 rgba(255,255,255,0.05)",
    backdropFilter: "blur(20px)",
  } as React.CSSProperties,
  logoWrap: { textAlign: "center" as const, marginBottom: 32 } as React.CSSProperties,
  logoIcon: {
    width: 56, height: 56, borderRadius: 16, margin: "0 auto 12px",
    background: "linear-gradient(135deg, rgba(124,106,247,0.2), rgba(167,139,250,0.1))",
    border: "1px solid rgba(124,106,247,0.3)",
    display: "flex", alignItems: "center", justifyContent: "center",
    boxShadow: "0 0 24px rgba(124,106,247,0.2)",
  } as React.CSSProperties,
  logoName: { fontSize: 26, fontWeight: 700, color: "#fff", letterSpacing: -0.5 } as React.CSSProperties,
  logoSub: { fontSize: 12, color: "var(--text-muted)", marginTop: 4, letterSpacing: 0.5 } as React.CSSProperties,
  tabs: {
    display: "flex", position: "relative" as const, marginBottom: 28,
    background: "var(--bg-input)", borderRadius: "var(--radius-md)", padding: 4,
  } as React.CSSProperties,
  tab: (active: boolean): React.CSSProperties => ({
    flex: 1, padding: "8px 0", background: active ? "var(--bg-elevated)" : "none",
    border: "none", borderRadius: "var(--radius-sm)", cursor: "pointer",
    fontSize: 13, fontWeight: 600,
    color: active ? "var(--text-primary)" : "var(--text-muted)",
    transition: "all 0.2s", position: "relative" as const, zIndex: 1,
    boxShadow: active ? "var(--shadow-sm)" : "none",
  }),
  tabIndicator: (_right: boolean): React.CSSProperties => ({ display: "none" }),
  form: { display: "flex", flexDirection: "column" as const, gap: 16 } as React.CSSProperties,
  field: { display: "flex", flexDirection: "column" as const, gap: 6 } as React.CSSProperties,
  label: { fontSize: 12, fontWeight: 600, color: "var(--text-secondary)", letterSpacing: 0.3 } as React.CSSProperties,
  inputWrap: { position: "relative" as const } as React.CSSProperties,
  inputIcon: {
    position: "absolute" as const, left: 12, top: "50%", transform: "translateY(-50%)",
    color: "var(--text-muted)", display: "flex", alignItems: "center",
  } as React.CSSProperties,
  input: {
    width: "100%", padding: "11px 12px 11px 36px",
    background: "var(--bg-input)", border: "1px solid var(--border)",
    borderRadius: "var(--radius-sm)", color: "var(--text-primary)",
    fontSize: 14, outline: "none", transition: "border-color 0.2s, box-shadow 0.2s",
    fontFamily: "inherit",
  } as React.CSSProperties,
  error: {
    display: "flex", alignItems: "center", gap: 8,
    background: "rgba(240,79,94,0.1)", border: "1px solid rgba(240,79,94,0.3)",
    borderRadius: "var(--radius-sm)", padding: "10px 12px",
    fontSize: 13, color: "#f04f5e",
  } as React.CSSProperties,
  btn: (loading: boolean): React.CSSProperties => ({
    padding: "12px 0", marginTop: 4,
    background: loading ? "rgba(124,106,247,0.5)" : "linear-gradient(135deg, #7c6af7, #a78bfa)",
    border: "none", borderRadius: "var(--radius-sm)", color: "#fff",
    fontSize: 14, fontWeight: 600, cursor: loading ? "not-allowed" : "pointer",
    transition: "opacity 0.2s, transform 0.1s",
    display: "flex", alignItems: "center", justifyContent: "center", gap: 8,
    boxShadow: loading ? "none" : "0 4px 16px rgba(124,106,247,0.4)",
  }),
  spinner: {
    width: 14, height: 14, border: "2px solid rgba(255,255,255,0.3)",
    borderTopColor: "#fff", borderRadius: "50%", animation: "spin 0.7s linear infinite",
    display: "inline-block",
  } as React.CSSProperties,
  footer: {
    textAlign: "center" as const, marginTop: 24,
    fontSize: 11, color: "var(--text-muted)", letterSpacing: 0.3,
  } as React.CSSProperties,
};
