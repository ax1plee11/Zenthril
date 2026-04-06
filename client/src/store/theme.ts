/**
 * Тема приложения — Context + hooks + CSS переменные
 */

import { createContext, useContext } from "react";

export interface Theme {
  scheme: "dark" | "light" | "custom";
  accentColor: string;
  chatBackground: string | null;
}

const THEME_KEY = "vibrora_theme";

export const DEFAULT_THEME: Theme = {
  scheme: "dark",
  accentColor: "#7289da",
  chatBackground: null,
};

const DARK_VARS = {
  "--accent": "#7289da",
  "--bg-primary": "#36393f",
  "--bg-secondary": "#2f3136",
  "--bg-tertiary": "#202225",
  "--text-primary": "#dcddde",
  "--text-muted": "#72767d",
};

const LIGHT_VARS = {
  "--accent": "#5865f2",
  "--bg-primary": "#ffffff",
  "--bg-secondary": "#f2f3f5",
  "--bg-tertiary": "#e3e5e8",
  "--text-primary": "#2e3338",
  "--text-muted": "#747f8d",
};

export function loadTheme(): Theme {
  try {
    const raw = localStorage.getItem(THEME_KEY);
    if (raw) return { ...DEFAULT_THEME, ...JSON.parse(raw) };
  } catch {
    // ignore
  }
  return { ...DEFAULT_THEME };
}

export function saveTheme(theme: Theme): void {
  localStorage.setItem(THEME_KEY, JSON.stringify(theme));
}

export function applyTheme(theme: Theme): void {
  const root = document.documentElement;
  const vars = theme.scheme === "light" ? LIGHT_VARS : DARK_VARS;

  for (const [key, value] of Object.entries(vars)) {
    root.style.setProperty(key, value);
  }

  // Акцентный цвет всегда из темы
  root.style.setProperty("--accent", theme.accentColor);

  // Фон чата
  if (theme.chatBackground) {
    root.style.setProperty("--chat-bg-image", `url(${theme.chatBackground})`);
  } else {
    root.style.removeProperty("--chat-bg-image");
  }
}

export interface ThemeState {
  theme: Theme;
  setTheme: (theme: Theme) => void;
}

export const ThemeContext = createContext<ThemeState>({
  theme: DEFAULT_THEME,
  setTheme: () => {},
});

export function useTheme() {
  return useContext(ThemeContext);
}
