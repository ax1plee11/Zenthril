/**
 * Состояние аутентификации — Context + hooks
 */

import { createContext, useContext } from "react";

const TOKEN_KEY = "zenthril_token";
const USER_KEY = "zenthril_user";
export const AUTH_SESSION_EXPIRED_EVENT = "zenthril:auth-session-expired";

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
  const token = localStorage.getItem(TOKEN_KEY);
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
  return localStorage.getItem(TOKEN_KEY);
}

export function saveAccessToken(token: string): void {
  localStorage.setItem(TOKEN_KEY, token);
}

export function clearAuth(): void {
  localStorage.removeItem(TOKEN_KEY);
  localStorage.removeItem(USER_KEY);
}

export function notifySessionExpired(): void {
  clearAuth();
  if (typeof window !== "undefined") {
    window.dispatchEvent(new CustomEvent(AUTH_SESSION_EXPIRED_EVENT));
  }
}
