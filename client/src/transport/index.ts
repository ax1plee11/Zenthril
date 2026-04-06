/**
 * TransportLayer — WebSocket + REST + автопереключение узлов
 *
 * Требования: 3.7, 5.1, 5.2, 5.3, 10.8
 */

// ─── Типы ─────────────────────────────────────────────────────────────────────

export interface WSEvent {
  type: string;
  [key: string]: unknown;
}

// ─── P2P-заглушка ─────────────────────────────────────────────────────────────

class P2PClient {
  async activate(): Promise<void> {
    console.warn("[TransportLayer] All nodes unavailable — activating P2P mode");
  }
}

// ─── Константы ────────────────────────────────────────────────────────────────

const TOKEN_KEY = "veltrix_token";
const HEARTBEAT_INTERVAL_MS = 30_000;
const CONNECT_TIMEOUT_MS = 5_000;

// ─── TransportLayer ───────────────────────────────────────────────────────────

export class TransportLayer {
  private nodes: string[];
  private currentNodeIndex: number;
  private ws: WebSocket | null;
  private handlers: Map<string, Set<(event: WSEvent) => void>>;
  private p2pClient: P2PClient | null;
  private heartbeatTimer: ReturnType<typeof setInterval> | null;

  constructor(nodes: string[] = []) {
    if (nodes.length < 3) {
      throw new Error("TransportLayer requires at least 3 nodes");
    }
    this.nodes = nodes;
    this.currentNodeIndex = 0;
    this.ws = null;
    this.handlers = new Map();
    this.p2pClient = new P2PClient();
    this.heartbeatTimer = null;
  }

  // ─── Подключение ────────────────────────────────────────────────────────────

  connect(nodeUrl: string): Promise<void> {
    return new Promise((resolve, reject) => {
      const timeout = setTimeout(() => {
        reject(new Error(`Connection timeout to ${nodeUrl}`));
      }, CONNECT_TIMEOUT_MS);

      try {
        const ws = new WebSocket(nodeUrl);

        ws.onopen = () => {
          clearTimeout(timeout);
          this.ws = ws;
          this._startHeartbeat();
          resolve();
        };

        ws.onerror = () => {
          clearTimeout(timeout);
          reject(new Error(`WebSocket error connecting to ${nodeUrl}`));
        };

        ws.onclose = () => {
          this._stopHeartbeat();
          // Автопереключение при потере соединения
          this.switchToFallback().catch(() => {
            // Все узлы недоступны — P2P
          });
        };

        ws.onmessage = (event: MessageEvent) => {
          try {
            const data: WSEvent = JSON.parse(event.data as string);
            this._dispatch(data);
          } catch {
            // Игнорируем невалидные сообщения
          }
        };
      } catch (err) {
        clearTimeout(timeout);
        reject(err);
      }
    });
  }

  // ─── Отключение ─────────────────────────────────────────────────────────────

  disconnect(): void {
    this._stopHeartbeat();
    if (this.ws) {
      // Убираем onclose чтобы не триггерить автопереключение
      this.ws.onclose = null;
      this.ws.close();
      this.ws = null;
    }
  }

  // ─── Переключение на резервный узел ─────────────────────────────────────────

  async switchToFallback(): Promise<void> {
    const totalNodes = this.nodes.length;

    for (let attempt = 0; attempt < totalNodes; attempt++) {
      this.currentNodeIndex = (this.currentNodeIndex + 1) % totalNodes;
      const nextNode = this.nodes[this.currentNodeIndex];

      try {
        await this.connect(nextNode);
        return; // Успешно подключились
      } catch {
        // Пробуем следующий
      }
    }

    // Все узлы недоступны — активируем P2P
    await this.p2pClient!.activate();
    throw new Error("All nodes unavailable, P2P mode activated");
  }

  // ─── Отправка события ───────────────────────────────────────────────────────

  send(event: WSEvent): void {
    if (!this.ws || this.ws.readyState !== WebSocket.OPEN) {
      throw new Error("No active WebSocket connection");
    }
    this.ws.send(JSON.stringify(event));
  }

  // ─── Подписка / отписка ─────────────────────────────────────────────────────

  subscribe(
    eventType: string,
    handler: (e: WSEvent) => void,
  ): () => void {
    if (!this.handlers.has(eventType)) {
      this.handlers.set(eventType, new Set());
    }
    this.handlers.get(eventType)!.add(handler);

    // Возвращаем функцию отписки
    return () => {
      const set = this.handlers.get(eventType);
      if (set) {
        set.delete(handler);
        if (set.size === 0) {
          this.handlers.delete(eventType);
        }
      }
    };
  }

  // ─── REST-запросы ────────────────────────────────────────────────────────────

  async request<T>(method: string, path: string, body?: unknown): Promise<T> {
    const baseUrl = this.nodes[this.currentNodeIndex];
    const url = `${baseUrl}${path}`;
    const token = localStorage.getItem(TOKEN_KEY);

    const headers: Record<string, string> = {
      "Content-Type": "application/json",
    };
    if (token) {
      headers["Authorization"] = `Bearer ${token}`;
    }

    const response = await fetch(url, {
      method,
      headers,
      body: body !== undefined ? JSON.stringify(body) : undefined,
    });

    if (!response.ok) {
      throw new Error(`HTTP ${response.status}: ${response.statusText}`);
    }

    return response.json() as Promise<T>;
  }

  // ─── Внутренние методы ───────────────────────────────────────────────────────

  private _dispatch(event: WSEvent): void {
    const handlers = this.handlers.get(event.type);
    if (handlers) {
      handlers.forEach((h) => h(event));
    }
  }

  private _startHeartbeat(): void {
    this._stopHeartbeat();
    this.heartbeatTimer = setInterval(() => {
      try {
        this.send({ type: "ping" });
      } catch {
        // Соединение уже потеряно
      }
    }, HEARTBEAT_INTERVAL_MS);
  }

  private _stopHeartbeat(): void {
    if (this.heartbeatTimer !== null) {
      clearInterval(this.heartbeatTimer);
      this.heartbeatTimer = null;
    }
  }
}

// ─── Singleton ────────────────────────────────────────────────────────────────

export const transport = new TransportLayer([
  "wss://node1.veltrix.app",
  "wss://node2.veltrix.app",
  "wss://node3.veltrix.app",
]);
