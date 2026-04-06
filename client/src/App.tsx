import { useState, useCallback, useEffect } from "react";
import { AuthContext, loadStoredAuth, saveAuth, clearAuth } from "./store/auth";
import type { AuthUser } from "./store/auth";
import { ThemeContext, loadTheme, saveTheme, applyTheme, ANIMATED_BG_PRESETS } from "./store/theme";
import type { Theme } from "./store/theme";
import AuthScreen from "./components/AuthScreen";
import MainLayout from "./components/MainLayout";

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
  const [theme, setThemeState] = useState<Theme>(loadTheme);

  // Применяем тему при старте
  useEffect(() => { applyTheme(theme); }, []); // eslint-disable-line react-hooks/exhaustive-deps

  const setTheme = useCallback((next: Theme) => {
    applyTheme(next);
    saveTheme(next);
    setThemeState(next);
  }, []);

  const login = useCallback((newToken: string, newUser: AuthUser) => {
    saveAuth(newToken, newUser);
    setToken(newToken);
    setUser(newUser);
  }, []);

  const logout = useCallback(() => {
    clearAuth();
    setToken(null);
    setUser(null);
  }, []);

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
            {token && user ? (
              <MainLayout />
            ) : (
              <AuthScreen onAuth={() => {
                const { token: t, user: u } = loadStoredAuth();
                if (t && u) { setToken(t); setUser(u); }
              }} />
            )}
          </div>
        </div>
      </AuthContext.Provider>
    </ThemeContext.Provider>
  );
}
