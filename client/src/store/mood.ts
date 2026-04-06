/**
 * Mood Indicator — отслеживает активность пользователя
 * и вычисляет "настроение" на основе паттернов поведения
 */

export type Mood = "focused" | "active" | "chill" | "away" | "creative" | "social";

export interface MoodState {
  mood: Mood;
  label: string;
  color: string;
  glowColor: string;
  emoji: string;
  description: string;
}

export const MOODS: Record<Mood, MoodState> = {
  focused: {
    mood: "focused",
    label: "Focused",
    color: "#7c6af7",
    glowColor: "rgba(124,106,247,0.5)",
    emoji: "🎯",
    description: "Deep in work",
  },
  active: {
    mood: "active",
    label: "Active",
    color: "#3ecf8e",
    glowColor: "rgba(62,207,142,0.5)",
    emoji: "⚡",
    description: "On fire today",
  },
  chill: {
    mood: "chill",
    label: "Chill",
    color: "#06b6d4",
    glowColor: "rgba(6,182,212,0.5)",
    emoji: "😌",
    description: "Taking it easy",
  },
  away: {
    mood: "away",
    label: "Away",
    color: "#5a5b70",
    glowColor: "rgba(90,91,112,0.3)",
    emoji: "💤",
    description: "Stepped away",
  },
  creative: {
    mood: "creative",
    label: "Creative",
    color: "#ec4899",
    glowColor: "rgba(236,72,153,0.5)",
    emoji: "✨",
    description: "In the zone",
  },
  social: {
    mood: "social",
    label: "Social",
    color: "#f5a623",
    glowColor: "rgba(245,166,35,0.5)",
    emoji: "🎉",
    description: "Let's chat!",
  },
};

interface ActivityTracker {
  messageCount: number;
  lastMessageTime: number;
  sessionStart: number;
  typingBursts: number;
  lastActivityTime: number;
}

const tracker: ActivityTracker = {
  messageCount: 0,
  lastMessageTime: 0,
  sessionStart: Date.now(),
  typingBursts: 0,
  lastActivityTime: Date.now(),
};

export function trackMessage(): void {
  const now = Date.now();
  const timeSinceLast = now - tracker.lastMessageTime;

  tracker.messageCount++;
  tracker.lastMessageTime = now;
  tracker.lastActivityTime = now;

  // Быстрые сообщения подряд = typing burst
  if (timeSinceLast < 10_000) {
    tracker.typingBursts++;
  } else {
    tracker.typingBursts = Math.max(0, tracker.typingBursts - 1);
  }
}

export function trackTyping(): void {
  tracker.lastActivityTime = Date.now();
}

export function computeMood(): Mood {
  const now = Date.now();
  const idleMs = now - tracker.lastActivityTime;
  const sessionMs = now - tracker.sessionStart;
  const msgsPerMin = tracker.messageCount / Math.max(1, sessionMs / 60_000);

  // Away — нет активности > 5 минут
  if (idleMs > 5 * 60_000) return "away";

  // Social — много сообщений быстро
  if (tracker.typingBursts >= 5 || msgsPerMin > 3) return "social";

  // Active — хорошая активность
  if (msgsPerMin > 1.5 || tracker.messageCount > 10) return "active";

  // Creative — средняя активность, недавно печатал
  if (idleMs < 30_000 && tracker.messageCount > 3) return "creative";

  // Focused — печатает медленно, вдумчиво
  if (idleMs < 2 * 60_000 && msgsPerMin < 1) return "focused";

  // Chill — по умолчанию
  return "chill";
}

export function getMoodState(): MoodState {
  return MOODS[computeMood()];
}

// Подписка на изменения настроения
type MoodListener = (mood: MoodState) => void;
const listeners = new Set<MoodListener>();

export function subscribeMood(fn: MoodListener): () => void {
  listeners.add(fn);
  return () => listeners.delete(fn);
}

// Обновляем настроение каждые 30 секунд
setInterval(() => {
  const mood = getMoodState();
  listeners.forEach(fn => fn(mood));
}, 30_000);
