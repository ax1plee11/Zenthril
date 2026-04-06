import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { TransportLayer, type WSEvent } from "./index";

// ─── Мок WebSocket ────────────────────────────────────────────────────────────

class MockWebSocket {
  static OPEN = 1;
  static CLOSING = 2;
  static CLOSED = 3;

  readyState: number;
  url: string;
  onopen: (() => void) | null = null;
  onclose: (() => void) | null = null;
  onerror: ((e: unknown) => void) | null = null;
  onmessage: ((e: { data: string }) => void) | null = null;

  sentMessages: string[] = [];

  constructor(url: string) {
    this.url = url;
    this.readyState = MockWebSocket.OPEN;
  }

  send(data: string): void {
    this.sentMessages.push(data);
  }

  close(): void {
    this.readyState = MockWebSocket.CLOSED;
  }

  simulateOpen(): void {
    this.readyState = MockWebSocket.OPEN;
    this.onopen?.();
  }

  simulateClose(): void {
    this.readyState = MockWebSocket.CLOSED;
    this.onclose?.();
  }

  simulateError(): void {
    this.onerror?.(new Error("connection error"));
  }

  simulateMessage(data: WSEvent): void {
    this.onmessage?.({ data: JSON.stringify(data) });
  }
}

// ─── Хелпер: создать TransportLayer с 3 узлами ───────────────────────────────

function makeTransport(nodes?: string[]): TransportLayer {
  return new TransportLayer(
    nodes ?? ["ws://node1", "ws://node2", "ws://node3"],
  );
}

// ─── Хелпер: подключить transport с мок-WS ───────────────────────────────────

async function connectTransport(
  transport: TransportLayer,
  url = "ws://node1",
): Promise<MockWebSocket> {
  let capturedWs!: MockWebSocket;

  vi.stubGlobal(
    "WebSocket",
    class extends MockWebSocket {
      constructor(u: string) {
        super(u);
        capturedWs = this;
      }
    },
  );

  const promise = transport.connect(url);
  capturedWs.simulateOpen();
  await promise;
  return capturedWs;
}

// ─── Тесты ───────────────────────────────────────────────────────────────────

describe("TransportLayer", () => {
  beforeEach(() => {
    vi.stubGlobal("localStorage", {
      getItem: vi.fn(() => null),
      setItem: vi.fn(),
      removeItem: vi.fn(),
    });
  });

  afterEach(() => {
    vi.unstubAllGlobals();
    vi.clearAllMocks();
  });

  // ─── Конструктор ────────────────────────────────────────────────────────────

  describe("constructor", () => {
    it("бросает ошибку если узлов меньше 3", () => {
      expect(() => new TransportLayer(["ws://a", "ws://b"])).toThrow(
        "at least 3 nodes",
      );
    });

    it("создаётся с 3 и более узлами", () => {
      expect(() => makeTransport()).not.toThrow();
    });
  });

  // ─── connect ────────────────────────────────────────────────────────────────

  describe("connect", () => {
    it("успешно подключается когда WebSocket открывается", async () => {
      const transport = makeTransport();
      const ws = await connectTransport(transport);
      expect(ws.readyState).toBe(MockWebSocket.OPEN);
    });

    it("отклоняет промис при ошибке WebSocket", async () => {
      const transport = makeTransport();
      let capturedWs!: MockWebSocket;

      vi.stubGlobal(
        "WebSocket",
        class extends MockWebSocket {
          constructor(u: string) {
            super(u);
            capturedWs = this;
          }
        },
      );

      const connectPromise = transport.connect("ws://node1");
      capturedWs.simulateError();
      await expect(connectPromise).rejects.toThrow();
    });

    it("отклоняет промис по таймауту (5 сек)", async () => {
      vi.useFakeTimers();
      const transport = makeTransport();

      vi.stubGlobal(
        "WebSocket",
        class extends MockWebSocket {
          constructor(u: string) {
            super(u);
            // Не вызываем simulateOpen — имитируем зависание
          }
        },
      );

      const connectPromise = transport.connect("ws://node1");
      vi.advanceTimersByTime(5001);
      await expect(connectPromise).rejects.toThrow("timeout");
      vi.useRealTimers();
    });
  });

  // ─── Переключение на резервный узел ─────────────────────────────────────────

  describe("switchToFallback — переключение при недоступности основного узла", () => {
    it("подключается ко второму узлу если первый недоступен", async () => {
      const transport = makeTransport(["ws://node1", "ws://node2", "ws://node3"]);
      let callCount = 0;

      vi.stubGlobal(
        "WebSocket",
        class extends MockWebSocket {
          constructor(u: string) {
            super(u);
            callCount++;
            const self = this;
            if (callCount === 1) {
              // Первый узел — ошибка
              Promise.resolve().then(() => self.simulateError());
            } else {
              // Второй узел — успех
              Promise.resolve().then(() => self.simulateOpen());
            }
          }
        },
      );

      await transport.switchToFallback();
      expect(callCount).toBe(2);
    });

    it("активирует P2P если все узлы недоступны", async () => {
      const transport = makeTransport();
      const p2pActivateSpy = vi.fn().mockResolvedValue(undefined);

      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      (transport as any).p2pClient = { activate: p2pActivateSpy };

      vi.stubGlobal(
        "WebSocket",
        class extends MockWebSocket {
          constructor(u: string) {
            super(u);
            const self = this;
            Promise.resolve().then(() => self.simulateError());
          }
        },
      );

      await expect(transport.switchToFallback()).rejects.toThrow(
        "All nodes unavailable",
      );
      expect(p2pActivateSpy).toHaveBeenCalledOnce();
    });
  });

  // ─── send ────────────────────────────────────────────────────────────────────

  describe("send", () => {
    it("бросает ошибку если нет активного соединения", () => {
      const transport = makeTransport();
      expect(() => transport.send({ type: "message", text: "hello" })).toThrow(
        "No active WebSocket connection",
      );
    });

    it("отправляет JSON-сериализованное событие через WebSocket", async () => {
      const transport = makeTransport();
      const ws = await connectTransport(transport);

      const event: WSEvent = { type: "message", text: "hello" };
      transport.send(event);

      expect(ws.sentMessages).toHaveLength(1);
      // Первое сообщение — heartbeat ping или наше событие
      const parsed = JSON.parse(ws.sentMessages[ws.sentMessages.length - 1]);
      expect(parsed).toEqual(event);
    });

    it("бросает ошибку если WebSocket не в состоянии OPEN", async () => {
      const transport = makeTransport();
      const ws = await connectTransport(transport);

      ws.readyState = MockWebSocket.CLOSED;
      expect(() => transport.send({ type: "ping" })).toThrow(
        "No active WebSocket connection",
      );
    });
  });

  // ─── subscribe / unsubscribe ─────────────────────────────────────────────────

  describe("subscribe / unsubscribe", () => {
    it("обработчик вызывается при получении события нужного типа", async () => {
      const transport = makeTransport();
      const ws = await connectTransport(transport);

      const handler = vi.fn();
      transport.subscribe("message", handler);

      ws.simulateMessage({ type: "message", text: "hi" });
      expect(handler).toHaveBeenCalledOnce();
      expect(handler).toHaveBeenCalledWith({ type: "message", text: "hi" });
    });

    it("обработчик НЕ вызывается для событий другого типа", async () => {
      const transport = makeTransport();
      const ws = await connectTransport(transport);

      const handler = vi.fn();
      transport.subscribe("message", handler);

      ws.simulateMessage({ type: "presence", userId: "123" });
      expect(handler).not.toHaveBeenCalled();
    });

    it("unsubscribe прекращает вызов обработчика", async () => {
      const transport = makeTransport();
      const ws = await connectTransport(transport);

      const handler = vi.fn();
      const unsubscribe = transport.subscribe("message", handler);

      ws.simulateMessage({ type: "message", text: "first" });
      expect(handler).toHaveBeenCalledTimes(1);

      unsubscribe();

      ws.simulateMessage({ type: "message", text: "second" });
      expect(handler).toHaveBeenCalledTimes(1); // Не вызван повторно
    });

    it("несколько обработчиков на один тип работают независимо", async () => {
      const transport = makeTransport();
      const ws = await connectTransport(transport);

      const h1 = vi.fn();
      const h2 = vi.fn();
      transport.subscribe("message", h1);
      transport.subscribe("message", h2);

      ws.simulateMessage({ type: "message" });
      expect(h1).toHaveBeenCalledOnce();
      expect(h2).toHaveBeenCalledOnce();
    });

    it("отписка одного обработчика не затрагивает другие", async () => {
      const transport = makeTransport();
      const ws = await connectTransport(transport);

      const h1 = vi.fn();
      const h2 = vi.fn();
      const unsub1 = transport.subscribe("message", h1);
      transport.subscribe("message", h2);

      unsub1();

      ws.simulateMessage({ type: "message" });
      expect(h1).not.toHaveBeenCalled();
      expect(h2).toHaveBeenCalledOnce();
    });
  });

  // ─── disconnect ──────────────────────────────────────────────────────────────

  describe("disconnect", () => {
    it("закрывает WebSocket соединение", async () => {
      const transport = makeTransport();
      const ws = await connectTransport(transport);

      const closeSpy = vi.spyOn(ws, "close");
      transport.disconnect();
      expect(closeSpy).toHaveBeenCalledOnce();
    });

    it("после disconnect send бросает ошибку", async () => {
      const transport = makeTransport();
      await connectTransport(transport);

      transport.disconnect();
      expect(() => transport.send({ type: "ping" })).toThrow(
        "No active WebSocket connection",
      );
    });
  });
});
