/**
 * Тема приложения — Context + hooks + CSS переменные
 * Анимированный фон только на document.body, цвета через --anim-color-* и сохранение в теме
 */

import { createContext, useContext } from "react";

export type AnimatedBodyClass = "animated-bg" | "animated-bg-custom";

export interface AnimatedBgPreset {
  id: string;
  label: string;
  /** Пустой массив — цвета строятся от акцента (пресет «Акцент») */
  colors: string[];
  bodyClass: AnimatedBodyClass;
}

export const ANIMATED_BG_PRESETS: AnimatedBgPreset[] = [
  { id: "aurora",  label: "Aurora",  colors: ["#0d1b2a","#1b4332","#0d1b2a","#1a472a"], bodyClass: "animated-bg" },
  { id: "galaxy",  label: "Galaxy",  colors: ["#1a1a2e","#16213e","#0f3460","#533483"], bodyClass: "animated-bg" },
  { id: "sunset",  label: "Sunset",  colors: ["#2d1b69","#fc5c7d","#ff9a44","#fc5c7d"], bodyClass: "animated-bg" },
  { id: "ocean",   label: "Ocean",   colors: ["#0f2027","#203a43","#2c5364","#1a6b8a"], bodyClass: "animated-bg" },
  { id: "fire",    label: "Fire",    colors: ["#1a0000","#7f0000","#cc2200","#ff6600"], bodyClass: "animated-bg" },
  { id: "forest",  label: "Forest",  colors: ["#0a2e0a","#1a5c1a","#0d3b0d","#2d7a2d"], bodyClass: "animated-bg" },
  { id: "neon",    label: "Neon",    colors: ["#0d0221","#7c6af7","#0d0221","#f72585"], bodyClass: "animated-bg" },
  { id: "custom",  label: "Accent",  colors: [], bodyClass: "animated-bg-custom" },
];

export type TopbarItemId =
  | "search"
  | "notifications"
  | "friends"
  | "settings"
  | "divider";

export interface TopbarItem {
  id: TopbarItemId;
  visible: boolean;
}

export const DEFAULT_TOPBAR_ITEMS: TopbarItem[] = [
  { id: "search",        visible: true },
  { id: "notifications", visible: true },
  { id: "friends",       visible: true },
  { id: "divider",       visible: true },
  { id: "settings",      visible: true },
];

export interface Theme {
  scheme: "dark" | "light" | "custom";
  accentColor: string;
  chatBackground: string | null;
  chatBgOpacity: number;
  animatedPresetId: string | null;
  animatedColors: [string, string, string, string] | null;
  topbarItems: TopbarItem[];
  overlayDarkness: number; // 0=прозрачный, 100=тёмный — применяется ко всем панелям
}

const THEME_KEY = "vibrora_theme";

export const DEFAULT_THEME: Theme = {
  scheme: "dark",
  accentColor: "#7c6af7",
  chatBackground: null,
  chatBgOpacity: 100,
  animatedPresetId: null,
  animatedColors: null,
  topbarItems: [...DEFAULT_TOPBAR_ITEMS],
  overlayDarkness: 40,
};

const DARK_VARS: Record<string, string> = {
  "--accent":         "#7c6af7",
  "--accent-hover":   "#9b8df9",
  "--accent-dim":     "rgba(124,106,247,0.15)",
  "--bg-base":        "#0d0e14",
  "--bg-surface":     "#13141d",
  "--bg-elevated":    "#1a1b27",
  "--bg-input":       "#0f1018",
  "--border":         "rgba(255,255,255,0.06)",
  "--border-accent":  "rgba(124,106,247,0.4)",
  "--text-primary":   "#e8e9f0",
  "--text-secondary": "#9a9bb0",
  "--text-muted":     "#5a5b70",
  // legacy compat
  "--bg-primary":     "#1a1b27",
  "--bg-secondary":   "#13141d",
  "--bg-tertiary":    "#0d0e14",
};

const LIGHT_VARS: Record<string, string> = {
  "--accent":         "#5865f2",
  "--accent-hover":   "#7983f5",
  "--accent-dim":     "rgba(88,101,242,0.12)",
  "--bg-base":        "#f2f3f5",
  "--bg-surface":     "#ffffff",
  "--bg-elevated":    "#e3e5e8",
  "--bg-input":       "#ebedef",
  "--border":         "rgba(0,0,0,0.08)",
  "--border-accent":  "rgba(88,101,242,0.4)",
  "--text-primary":   "#2e3338",
  "--text-secondary": "#4f5660",
  "--text-muted":     "#747f8d",
  "--bg-primary":     "#ffffff",
  "--bg-secondary":   "#f2f3f5",
  "--bg-tertiary":    "#e3e5e8",
};

export function presetResolvedColors(
  theme: Theme,
): [string, string, string, string] {
  if (theme.animatedColors) {
    return theme.animatedColors;
  }
  const preset = theme.animatedPresetId
    ? ANIMATED_BG_PRESETS.find((p) => p.id === theme.animatedPresetId)
    : undefined;
  if (preset?.colors.length === 4) {
    return preset.colors as [string, string, string, string];
  }
  const a = theme.accentColor;
  return [a + "44", a + "22", "var(--bg-primary)", a + "11"];
}

export function loadTheme(): Theme {
  try {
    const raw = localStorage.getItem(THEME_KEY);
    if (raw) {
      const parsed = JSON.parse(raw);
      // Если в старых данных был base64 — убираем
      if (parsed.chatBackground?.startsWith("data:")) {
        parsed.chatBackground = null;
      }
      return { ...DEFAULT_THEME, ...parsed, topbarItems: parsed.topbarItems ?? [...DEFAULT_TOPBAR_ITEMS], overlayDarkness: parsed.overlayDarkness ?? 40 };
    }
  } catch {
    // ignore
  }
  return { ...DEFAULT_THEME };
}

export function saveTheme(theme: Theme): void {
  // Не сохраняем base64 изображения — они слишком большие для localStorage
  // Сохраняем только URL-ссылки
  const toSave: Theme = {
    ...theme,
    chatBackground: theme.chatBackground?.startsWith("data:")
      ? null  // base64 не сохраняем
      : theme.chatBackground,
  };
  try {
    localStorage.setItem(THEME_KEY, JSON.stringify(toSave));
  } catch {
    // Если всё равно не влезает — сохраняем без фона
    try {
      localStorage.setItem(THEME_KEY, JSON.stringify({ ...toSave, chatBackground: null }));
    } catch {
      // ignore
    }
  }
}

export function applyTheme(theme: Theme): void {
  const root = document.documentElement;
  const vars = theme.scheme === "light" ? LIGHT_VARS : DARK_VARS;

  // 1. Применяем базовые CSS переменные
  for (const [key, value] of Object.entries(vars)) {
    root.style.setProperty(key, value);
  }

  // 2. Акцентный цвет
  root.style.setProperty("--accent", theme.accentColor);
  root.style.setProperty("--accent-hover", theme.accentColor + "dd");
  root.style.setProperty("--accent-dim", theme.accentColor + "26");
  root.style.setProperty("--border-accent", theme.accentColor + "66");

  // 2b. Overlay darkness — прозрачность всех панелей через отдельные переменные
  const darkness = (theme.overlayDarkness ?? 40) / 100;
  const panelAlpha = (1 - darkness * 0.85).toFixed(2);
  const hasAnyBg = !!(theme.animatedPresetId || (theme.chatBackground && !theme.chatBackground.startsWith("__pattern__")));

  if (hasAnyBg) {
    if (theme.scheme === "light") {
      root.style.setProperty("--panel-bg",         `rgba(242,243,245,${panelAlpha})`);
      root.style.setProperty("--panel-bg-elevated", `rgba(227,229,232,${Math.min(1, parseFloat(panelAlpha) + 0.08).toFixed(2)})`);
      root.style.setProperty("--panel-topbar",      `rgba(235,236,240,${Math.min(1, parseFloat(panelAlpha) + 0.1).toFixed(2)})`);
    } else {
      root.style.setProperty("--panel-bg",         `rgba(13,14,20,${panelAlpha})`);
      root.style.setProperty("--panel-bg-elevated", `rgba(26,27,39,${Math.min(1, parseFloat(panelAlpha) + 0.08).toFixed(2)})`);
      root.style.setProperty("--panel-topbar",      `rgba(10,11,16,${Math.min(1, parseFloat(panelAlpha) + 0.1).toFixed(2)})`);
    }
  } else {
    // Без фона — панели непрозрачные, используют стандартные цвета
    root.style.setProperty("--panel-bg",          theme.scheme === "light" ? "#ffffff" : "#13141d");
    root.style.setProperty("--panel-bg-elevated", theme.scheme === "light" ? "#e3e5e8" : "#1a1b27");
    root.style.setProperty("--panel-topbar",      theme.scheme === "light" ? "#f2f3f5" : "#0d0e14");
  }

  // 3. App background — применяем на #root, body делаем прозрачным
  const appRoot = document.getElementById("root");
  document.body.classList.remove("animated-bg", "animated-bg-custom", "has-app-bg");
  document.body.style.background = "";
  document.body.style.backgroundSize = "";
  document.body.style.animation = "";

  if (theme.animatedPresetId) {
    const preset = ANIMATED_BG_PRESETS.find(p => p.id === theme.animatedPresetId);
    if (preset) {
      const colors = preset.colors.length === 4
        ? preset.colors
        : [theme.accentColor + "44", theme.accentColor + "22", "#0d0e14", theme.accentColor + "11"];

      const gradient = `linear-gradient(135deg, ${colors.join(", ")})`;

      if (appRoot) {
        appRoot.style.background = gradient;
        appRoot.style.backgroundSize = "400% 400%";
        appRoot.style.animation = "gradientShift 8s ease infinite";
      }
      document.body.classList.add("has-app-bg");
    }
  } else {
    if (appRoot) {
      appRoot.style.background = "";
      appRoot.style.backgroundSize = "";
      appRoot.style.animation = "";
    }
  }

  // 4. Chat background — сохраняем в CSS переменных для ChatView
  const bg = theme.chatBackground;
  const opacity = (theme.chatBgOpacity ?? 100) / 100;

  root.style.setProperty("--chat-bg-opacity", String(opacity));

  if (!bg) {
    root.style.setProperty("--chat-bg-image", "none");
    root.style.setProperty("--chat-bg-pattern", "none");
    root.style.setProperty("--chat-bg-size", "auto");
  } else if (bg.startsWith("__pattern__")) {
    const id = bg.replace("__pattern__", "");
    const patterns: Record<string, { image: string; size: string }> = {
      dots:     { image: "radial-gradient(circle, rgba(255,255,255,0.07) 1px, transparent 1px)", size: "24px 24px" },
      grid:     { image: "linear-gradient(rgba(255,255,255,0.05) 1px, transparent 1px), linear-gradient(90deg, rgba(255,255,255,0.05) 1px, transparent 1px)", size: "32px 32px" },
      diagonal: { image: "repeating-linear-gradient(45deg, rgba(255,255,255,0.03) 0px, rgba(255,255,255,0.03) 1px, transparent 1px, transparent 12px)", size: "auto" },
      noise:    { image: "none", size: "auto" },
    };
    const p = patterns[id] ?? { image: "none", size: "auto" };
    root.style.setProperty("--chat-bg-image", "none");
    root.style.setProperty("--chat-bg-pattern", p.image);
    root.style.setProperty("--chat-bg-size", p.size);
  } else {
    root.style.setProperty("--chat-bg-image", `url(${bg})`);
    root.style.setProperty("--chat-bg-pattern", "none");
    root.style.setProperty("--chat-bg-size", "cover");
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
