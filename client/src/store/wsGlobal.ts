/**
 * Единственное WebSocket-соединение для всего приложения.
 * ChatView, уведомления о друзьях, инвайты — всё через один сокет.
 */
import { api, getWebSocketUrl } from "../api/index";

type Handler = (event: Record<string, unknown>) => void;

const WS_URL = getWebSocketUrl("/ws");
const handlers = new Map<string, Set<Handler>>();
let ws: WebSocket | null = null;
let reconnectTimer: ReturnType<typeof setTimeout> | null = null;
let connecting = false;

// ─── Подписка на события ─────────────────────────────────────────────────────

export function onWSEvent(type: string, handler: Handler): () => void {
  if (!handlers.has(type)) handlers.set(type, new Set());
  handlers.get(type)!.add(handler);
  return () => {
    handlers.get(type)?.delete(handler);
  };
}

// ─── Отправка событий ────────────────────────────────────────────────────────

export function sendWSEvent(event: Record<string, unknown>): void {
  if (ws?.readyState === WebSocket.OPEN) {
    ws.send(JSON.stringify(event));
  }
}

export function isWSConnected(): boolean {
  return ws?.readyState === WebSocket.OPEN;
}

// ─── Подключение ─────────────────────────────────────────────────────────────

export async function connectGlobalWS(): Promise<void> {
  if (connecting) return;
  if (ws && (ws.readyState === WebSocket.OPEN || ws.readyState === WebSocket.CONNECTING)) return;

  connecting = true;
  try {
    const { ticket } = await api.auth.wsTicket();
    const socket = new WebSocket(`${WS_URL}?ticket=${encodeURIComponent(ticket)}`);
    ws = socket;

    socket.onopen = () => {
      connecting = false;
      // Уведомляем подписчиков что соединение установлено
      handlers.get("ws.connected")?.forEach(h => h({}));
    };

    socket.onmessage = (e: MessageEvent) => {
      try {
        const data = JSON.parse(e.data as string) as Record<string, unknown>;
        const type = data.type as string;
        // Диспатчим конкретный тип + wildcard "*"
        handlers.get(type)?.forEach(h => h(data));
        handlers.get("*")?.forEach(h => h(data));
      } catch { /* ignore */ }
    };

    socket.onclose = () => {
      connecting = false;
      ws = null;
      if (reconnectTimer) clearTimeout(reconnectTimer);
      reconnectTimer = setTimeout(() => connectGlobalWS(), 4000);
    };

    socket.onerror = () => {
      connecting = false;
      socket.close();
    };
  } catch {
    connecting = false;
    if (reconnectTimer) clearTimeout(reconnectTimer);
    reconnectTimer = setTimeout(() => connectGlobalWS(), 4000);
  }
}

export function disconnectGlobalWS(): void {
  if (reconnectTimer) clearTimeout(reconnectTimer);
  if (ws) {
    ws.onclose = null;
    ws.close();
    ws = null;
  }
}
