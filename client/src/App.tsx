/**
 * App — корневой компонент
 * Показывает AuthScreen если нет токена, иначе MainLayout
 */

import { useState, useCallback } from "react";
import {
  AuthContext,
  loadStoredAuth,
  saveAuth,
  clearAuth,
} from "./store/auth";
import type { AuthUser } from "./store/auth";
import AuthScreen from "./components/AuthScreen";
import MainLayout from "./components/MainLayout";

export default function App() {
  const stored = loadStoredAuth();
  const [token, setToken] = useState<string | null>(stored.token);
  const [user, setUser] = useState<AuthUser | null>(stored.user);

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
    <AuthContext.Provider value={{ token, user, login, logout }}>
      {token && user ? (
        <MainLayout />
      ) : (
        <AuthScreen
          onAuth={() => {
            // Перечитываем из localStorage после успешной аутентификации
            const { token: t, user: u } = loadStoredAuth();
            if (t && u) {
              setToken(t);
              setUser(u);
            }
          }}
        />
      )}
    </AuthContext.Provider>
  );
}
