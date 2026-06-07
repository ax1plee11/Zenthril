/**
 * Состояние аутентификации — Context + hooks
 */

import { createContext, useContext } from "react";

const TOKEN_KEY = "zenthril_token";
const USER_KEY = "zenthril_user";
export const AUTH_SESSION_EXPIRED_EVENT = "zenthril:auth-session-expired";

let accessTokenMemory: string | null = null;

export interface AuthUser {
  id: string;
  username: string;
  public_key: string;
}

export interface AuthState {
  token: string | null;
  user: AuthUser | null;
  login: (token: string, user: AuthUser) => void;
  logout: () => void;
}

export const AuthContext = createContext<AuthState>({
  token: null,
  user: null,
  login: () => {},
  logout: () => {},
});

export function useAuth() {
  return useContext(AuthContext);
}

export function loadStoredAuth(): { token: string | null; user: AuthUser | null } {
  const legacyToken = localStorage.getItem(TOKEN_KEY);
  if (legacyToken) {
    // SECURITY-HARDENING: migrate legacy localStorage access tokens into memory
    // and remove the persistent copy. Refresh tokens stay HttpOnly cookie-backed.
    accessTokenMemory = legacyToken;
    localStorage.removeItem(TOKEN_KEY);
  }
  const token = accessTokenMemory;
  const raw = localStorage.getItem(USER_KEY);
  let user: AuthUser | null = null;
  if (raw) {
    try {
      user = JSON.parse(raw) as AuthUser;
    } catch {
      user = null;
    }
  }
  return { token, user };
}

export function saveAuth(token: string, user: AuthUser): void {
  saveAccessToken(token);
  localStorage.setItem(USER_KEY, JSON.stringify(user));
}

export function loadAccessToken(): string | null {
  return accessTokenMemory;
}

export function saveAccessToken(token: string): void {
  // SECURITY-HARDENING: access tokens are intentionally process-memory only.
  // They are restored through the HttpOnly refresh cookie after a reload.
  accessTokenMemory = token;
  localStorage.removeItem(TOKEN_KEY);
}

export function clearAuth(): void {
  accessTokenMemory = null;
  localStorage.removeItem(TOKEN_KEY);
  localStorage.removeItem(USER_KEY);
}

export function notifySessionExpired(): void {
  clearAuth();
  if (typeof window !== "undefined") {
    window.dispatchEvent(new CustomEvent(AUTH_SESSION_EXPIRED_EVENT));
  }
}
