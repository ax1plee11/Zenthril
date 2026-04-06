/**
 * App — корневой компонент
 * Показывает AuthScreen если нет токена, иначе MainLayout
 */

import { useState, useCallback, useEffect } from "react";
import {
  AuthContext,
  loadStoredAuth,
  saveAuth,
  clearAuth,
} from "./store/auth";
import type { AuthUser } from "./store/auth";
import { ThemeContext, loadTheme, saveTheme, applyTheme } from "./store/theme";
import type { Theme } from "./store/theme";
import AuthScreen from "./components/AuthScreen";
import MainLayout from "./components/MainLayout";

export default function App() {
  const stored = loadStoredAuth();
  const [token, setToken] = useState<string | null>(stored.token);
  const [user, setUser] = useState<AuthUser | null>(stored.user);

  const [theme, setThemeState] = useState<Theme>(loadTheme);

  // Применяем тему при загрузке
  useEffect(() => {
    applyTheme(theme);
  }, []); // eslint-disable-line react-hooks/exhaustive-deps

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

  return (
    <ThemeContext.Provider value={{ theme, setTheme }}>
      <AuthContext.Provider value={{ token, user, login, logout }}>
        {token && user ? (
          <MainLayout />
        ) : (
          <AuthScreen
            onAuth={() => {
              const { token: t, user: u } = loadStoredAuth();
              if (t && u) {
                setToken(t);
                setUser(u);
              }
            }}
          />
        )}
      </AuthContext.Provider>
    </ThemeContext.Provider>
  );
}
