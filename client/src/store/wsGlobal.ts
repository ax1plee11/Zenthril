/**
 * Единственное WebSocket-соединение для всего приложения.
 * ChatView, уведомления о друзьях, инвайты — всё через один сокет.
 */
import { api, getActiveWebSocketUrl, clearActiveServer } from "../api/index";
import {
  paddingEnabled,
  decodePaddedFrame,
  encodePaddedFrame,
} from "../transport/padding";
import { applyTransportJitter } from "../transport/connectivityPolicy";

type Handler = (event: Record<string, unknown>) => void;

const handlers = new Map<string, Set<Handler>>();
let ws: WebSocket | null = null;
let reconnectTimer: ReturnType<typeof setTimeout> | null = null;
let connecting = false;
let reconnectAttempt = 0;
let intentionalDisconnect = false;

const RECONNECT_BASE_MS = 1000;
const RECONNECT_MAX_MS = 30000;

function scheduleReconnect(): void {
  if (intentionalDisconnect) return;
  const expDelay = Math.min(RECONNECT_BASE_MS * 2 ** reconnectAttempt, RECONNECT_MAX_MS);
  const jitter = Math.random() * 0.3 * expDelay;
  reconnectAttempt++;
  if (reconnectTimer) clearTimeout(reconnectTimer);
  // SECURITY: exponential backoff + jitter prevents reconnect storms against ws-ticket and /ws.
  reconnectTimer = setTimeout(() => {
    void connectGlobalWS();
  }, expDelay + jitter);
}

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
    const payload = paddingEnabled() ? encodePaddedFrame(event) : JSON.stringify(event);
    applyTransportJitter()
      .then(() => {
        if (ws?.readyState === WebSocket.OPEN) ws.send(payload);
      })
      .catch(() => {
        if (ws?.readyState === WebSocket.OPEN) ws.send(payload);
      });
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
    // WEAKNESS FIXED: fetch a fresh one-time ticket on every connect/reconnect attempt.
    const { ticket } = await api.auth.wsTicket();
    const wsUrl = await getActiveWebSocketUrl("/ws");
    const socket = new WebSocket(`${wsUrl}?ticket=${encodeURIComponent(ticket)}`);
    ws = socket;

    socket.onopen = () => {
      connecting = false;
      reconnectAttempt = 0;
      handlers.get("ws.connected")?.forEach(h => h({}));
    };

    socket.onmessage = (e: MessageEvent) => {
      try {
        const data = decodePaddedFrame(e.data as string);
        const type = data.type as string;
        handlers.get(type)?.forEach(h => h(data));
        handlers.get("*")?.forEach(h => h(data));
      } catch { /* ignore */ }
    };

    socket.onclose = () => {
      connecting = false;
      ws = null;
      scheduleReconnect();
    };

    socket.onerror = () => {
      connecting = false;
      clearActiveServer();
      socket.close();
    };
  } catch {
    connecting = false;
    clearActiveServer();
    scheduleReconnect();
  }
}

export function disconnectGlobalWS(): void {
  intentionalDisconnect = true;
  if (reconnectTimer) clearTimeout(reconnectTimer);
  reconnectTimer = null;
  reconnectAttempt = 0;
  if (ws) {
    ws.onclose = null;
    ws.close();
    ws = null;
  }
}

export function resetGlobalWSReconnectState(): void {
  intentionalDisconnect = false;
  reconnectAttempt = 0;
}
