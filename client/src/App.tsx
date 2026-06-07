import { useState, useCallback, useEffect } from "react";
import {
  AUTH_SESSION_EXPIRED_EVENT,
  AuthContext,
  loadStoredAuth,
  saveAuth,
  clearAuth,
} from "./store/auth";
import type { AuthUser } from "./store/auth";
import { ThemeContext, loadTheme, saveTheme, applyTheme, ANIMATED_BG_PRESETS } from "./store/theme";
import type { Theme } from "./store/theme";
import AuthScreen from "./components/AuthScreen";
import MainLayout from "./components/MainLayout";
import OfflineStartup from "./components/OfflineStartup";
import { CallManager } from "./features/calls/components/CallManager";
import { signalingService } from "./features/calls/services/signalingService";
import { getActiveServer, restoreAccessTokenFromRefreshCookie } from "./api/index";
import { shouldStartOnline } from "./store/startupPrivacy";

function getAppBackground(theme: Theme): React.CSSProperties {
  const bg = theme.chatBackground;

  // GIF/фото — на весь экран
  if (bg && !bg.startsWith("__pattern__")) {
    return {
      backgroundImage: `url(${bg})`,
      backgroundSize: "cover",
      backgroundPosition: "center",
      backgroundAttachment: "fixed",
    };
  }

  // App Background анимация (только если нет фото/GIF фона)
  if (theme.animatedPresetId && !bg) {
    const preset = ANIMATED_BG_PRESETS.find(p => p.id === theme.animatedPresetId);
    if (preset) {
      const colors = preset.colors.length === 4
        ? preset.colors
        : [theme.accentColor + "44", theme.accentColor + "22", "#0d0e14", theme.accentColor + "11"];
      return {
        background: `linear-gradient(135deg, ${colors.join(", ")})`,
        backgroundSize: "400% 400%",
        animation: "gradientShift 8s ease infinite",
      };
    }
  }

  return { background: "var(--bg-base, #0d0e14)" };
}

export default function App() {
  const stored = loadStoredAuth();
  const [token, setToken]      = useState<string | null>(stored.token);
  const [user, setUser]        = useState<AuthUser | null>(stored.user);
  const [networkEnabled, setNetworkEnabled] = useState(false);
  const [theme, setThemeState] = useState<Theme>(loadTheme);
  const [sessionNotice, setSessionNotice] = useState<string | null>(null);

  // Применяем тему при старте
  useEffect(() => { applyTheme(theme); }, []); // eslint-disable-line react-hooks/exhaustive-deps

  const setTheme = useCallback((next: Theme) => {
    applyTheme(next);
    saveTheme(next);
    setThemeState(next);
  }, []);

  const login = useCallback((newToken: string, newUser: AuthUser) => {
    setSessionNotice(null);
    saveAuth(newToken, newUser);
    setToken(newToken);
    setUser(newUser);
    setNetworkEnabled(true);
    getActiveServer()
      .then(server => signalingService.connect(server.wsBase, newToken))
      .catch(() => signalingService.connect("ws://localhost:8080", newToken));
  }, []);

  const connectSession = useCallback(async () => {
    if (!user) return;
    let activeToken = token;
    if (!activeToken) {
      activeToken = await restoreAccessTokenFromRefreshCookie();
      if (!activeToken) return;
      setToken(activeToken);
    }
    setSessionNotice(null);
    setNetworkEnabled(true);
    getActiveServer()
      .then(server => signalingService.connect(server.wsBase, activeToken))
      .catch(() => signalingService.connect("ws://localhost:8080", activeToken));
  }, [token, user]);

  const logout = useCallback((notice?: string) => {
    clearAuth();
    signalingService.disconnect();
    setNetworkEnabled(false);
    setToken(null);
    setUser(null);
    setSessionNotice(notice ?? null);
  }, []);

  useEffect(() => {
    const onExpired = () => {
      logout("Your session expired or was revoked. Please sign in again.");
    };
    window.addEventListener(AUTH_SESSION_EXPIRED_EVENT, onExpired);
    return () => window.removeEventListener(AUTH_SESSION_EXPIRED_EVENT, onExpired);
  }, [logout]);

  useEffect(() => {
    if (!user || networkEnabled || !shouldStartOnline(true)) return;
    void connectSession();
  }, [connectSession, networkEnabled, user]);

  const bgStyle = getAppBackground(theme);
  const hasBgImage = !!(theme.chatBackground && !theme.chatBackground.startsWith("__pattern__"));
  // Прозрачность overlay: 0 = полностью прозрачный (фон виден), 1 = полностью тёмный
  const overlayOpacity = hasBgImage ? 1 - (theme.chatBgOpacity ?? 100) / 100 : 0;

  return (
    <ThemeContext.Provider value={{ theme, setTheme }}>
      <AuthContext.Provider value={{ token, user, login, logout }}>
        <div style={{ width: "100%", height: "100%", position: "relative", ...bgStyle }}>
          {/* Тёмный overlay поверх GIF/фото — управляется прозрачностью */}
          {hasBgImage && (
            <div style={{
              position: "absolute", inset: 0, zIndex: 0,
              background: "rgba(0,0,0," + overlayOpacity.toFixed(2) + ")",
              pointerEvents: "none",
              transition: "background 0.3s",
            }} />
          )}
          {/* Контент поверх фона */}
          <div style={{ position: "relative", zIndex: 1, width: "100%", height: "100%" }}>
            {token && user && networkEnabled ? (
              <MainLayout />
            ) : user ? (
              <OfflineStartup
                username={user.username}
                onConnect={connectSession}
                onLogout={logout}
              />
            ) : (
              <AuthScreen onAuth={() => {
                const { token: t, user: u } = loadStoredAuth();
                if (t && u) {
                  setSessionNotice(null);
                  setToken(t);
                  setUser(u);
                  setNetworkEnabled(true);
                }
              }} sessionNotice={sessionNotice} />
            )}
            <CallManager />
          </div>
        </div>
      </AuthContext.Provider>
    </ThemeContext.Provider>
  );
}
