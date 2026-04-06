/**
 * AuthScreen — экран регистрации / логина
 * Требования: 1.1, 1.6
 */

import React, { useState, useCallback } from "react";
import { api } from "../api/index";
import { saveAuth } from "../store/auth";
import {
  generateKeyPair,
  exportPublicKey,
  storePrivateKey,
} from "../crypto/index";

interface AuthScreenProps {
  onAuth: () => void;
}

type Tab = "login" | "register";

const styles = {
  root: {
    display: "flex",
    alignItems: "center",
    justifyContent: "center",
    minHeight: "100vh",
    background: "#202225",
    fontFamily: "'Segoe UI', system-ui, sans-serif",
  } as React.CSSProperties,

  card: {
    background: "#36393f",
    borderRadius: 8,
    padding: "32px 40px",
    width: 400,
    boxShadow: "0 8px 32px rgba(0,0,0,0.5)",
  } as React.CSSProperties,

  logo: {
    textAlign: "center" as const,
    marginBottom: 24,
  } as React.CSSProperties,

  logoText: {
    fontSize: 28,
    fontWeight: 700,
    color: "#7289da",
    letterSpacing: 1,
  } as React.CSSProperties,

  subtitle: {
    fontSize: 13,
    color: "#72767d",
    marginTop: 4,
  } as React.CSSProperties,

  tabs: {
    display: "flex",
    marginBottom: 24,
    borderBottom: "2px solid #2f3136",
  } as React.CSSProperties,

  tab: (active: boolean): React.CSSProperties => ({
    flex: 1,
    padding: "10px 0",
    background: "none",
    border: "none",
    cursor: "pointer",
    fontSize: 14,
    fontWeight: 600,
    color: active ? "#7289da" : "#72767d",
    borderBottom: active ? "2px solid #7289da" : "2px solid transparent",
    marginBottom: -2,
    transition: "color 0.15s",
  }),

  label: {
    display: "block",
    fontSize: 12,
    fontWeight: 700,
    color: "#b9bbbe",
    textTransform: "uppercase" as const,
    letterSpacing: 0.5,
    marginBottom: 8,
    marginTop: 16,
  } as React.CSSProperties,

  input: {
    width: "100%",
    padding: "10px 12px",
    background: "#202225",
    border: "1px solid #040405",
    borderRadius: 4,
    color: "#dcddde",
    fontSize: 16,
    outline: "none",
    boxSizing: "border-box" as const,
    transition: "border-color 0.15s",
  } as React.CSSProperties,

  error: {
    background: "#f04747",
    color: "#fff",
    borderRadius: 4,
    padding: "10px 12px",
    fontSize: 14,
    marginTop: 16,
  } as React.CSSProperties,

  button: (loading: boolean): React.CSSProperties => ({
    width: "100%",
    marginTop: 24,
    padding: "12px 0",
    background: loading ? "#5b6eae" : "#7289da",
    border: "none",
    borderRadius: 4,
    color: "#fff",
    fontSize: 16,
    fontWeight: 600,
    cursor: loading ? "not-allowed" : "pointer",
    transition: "background 0.15s",
  }),
};

export default function AuthScreen({ onAuth }: AuthScreenProps) {
  const [tab, setTab] = useState<Tab>("login");
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  const handleSubmit = useCallback(
    async (e: React.FormEvent) => {
      e.preventDefault();
      setError(null);
      setLoading(true);

      try {
        if (tab === "register") {
          // Генерируем ключевую пару при регистрации (Требование 1.2)
          const keyPair = generateKeyPair();
          const publicKeyB64 = exportPublicKey(keyPair.publicKey);

          const res = await api.auth.register(username, password, publicKeyB64);

          // Сохраняем приватный ключ локально
          storePrivateKey(keyPair.secretKey);

          saveAuth(res.token, {
            id: res.user_id,
            username,
            public_key: publicKeyB64,
          });
        } else {
          const res = await api.auth.login(username, password);
          saveAuth(res.token, {
            id: res.user.id,
            username: res.user.username,
            public_key: res.user.public_key,
          });
        }

        onAuth();
      } catch {
        // Не раскрываем причину ошибки (Требование 1.6)
        setError("Неверные данные. Попробуйте ещё раз.");
      } finally {
        setLoading(false);
      }
    },
    [tab, username, password, onAuth],
  );

  return (
    <div style={styles.root}>
      <div style={styles.card}>
        <div style={styles.logo}>
          <div style={styles.logoText}>Vibrora</div>
          <div style={styles.subtitle}>Децентрализованный мессенджер</div>
        </div>

        <div style={styles.tabs}>
          <button
            style={styles.tab(tab === "login")}
            onClick={() => {
              setTab("login");
              setError(null);
            }}
          >
            Войти
          </button>
          <button
            style={styles.tab(tab === "register")}
            onClick={() => {
              setTab("register");
              setError(null);
            }}
          >
            Зарегистрироваться
          </button>
        </div>

        <form onSubmit={handleSubmit}>
          <label style={styles.label} htmlFor="username">
            Имя пользователя
          </label>
          <input
            id="username"
            style={styles.input}
            type="text"
            value={username}
            onChange={(e) => setUsername(e.target.value)}
            autoComplete="username"
            required
            minLength={1}
            maxLength={32}
            placeholder="username"
          />

          <label style={styles.label} htmlFor="password">
            Пароль
          </label>
          <input
            id="password"
            style={styles.input}
            type="password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            autoComplete={tab === "register" ? "new-password" : "current-password"}
            required
            minLength={1}
            placeholder="••••••••"
          />

          {error && <div style={styles.error}>{error}</div>}

          <button type="submit" style={styles.button(loading)} disabled={loading}>
            {loading
              ? "Загрузка..."
              : tab === "login"
                ? "Войти"
                : "Создать аккаунт"}
          </button>
        </form>
      </div>
    </div>
  );
}
