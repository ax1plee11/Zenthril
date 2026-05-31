import { describe, expect, it, vi } from "vitest";
import {
  checkServerHealth,
  loadServers,
  normalizeApiBase,
  resolveDoH,
  serverFromApiBase,
  wsBaseFromApiBase,
} from "./servers";

describe("server config", () => {
  it("normalizes API bases and rejects unsupported protocols", () => {
    expect(normalizeApiBase("https://example.com///")).toBe("https://example.com");
    expect(() => normalizeApiBase("ftp://example.com")).toThrow("http or https");
  });

  it("derives WebSocket base from API base", () => {
    expect(wsBaseFromApiBase("https://example.com")).toBe("wss://example.com");
    expect(wsBaseFromApiBase("http://localhost:8080")).toBe("ws://localhost:8080");
  });

  it("creates stable custom server entries", () => {
    const server = serverFromApiBase("Mirror", "https://mirror.example");
    expect(server.id).toBe("custom:https://mirror.example");
    expect(server.custom).toBe(true);
    expect(server.healthPath).toBe("/health");
    expect(server.transport).toBe("direct");
  });

  it("marks onion custom servers as Tor transport", () => {
    const server = serverFromApiBase("Onion", "http://zenthrilabc123.onion");

    expect(server.transport).toBe("tor");
    expect(server.onion).toBe(true);
    expect(server.wsBase).toBe("ws://zenthrilabc123.onion");
  });

  it("treats 200 and protected 401 health responses as reachable", async () => {
    vi.stubGlobal("fetch", vi.fn()
      .mockResolvedValueOnce({ ok: true, status: 200 })
      .mockResolvedValueOnce({ ok: false, status: 401 }));
    const server = serverFromApiBase("Test", "https://example.com");

    await expect(checkServerHealth(server)).resolves.toBe(true);
    await expect(checkServerHealth(server)).resolves.toBe(true);
  });

  it("loads mirrors as fallback servers from servers.json", async () => {
    localStorage.clear();
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({
        servers: [{
          id: "primary",
          name: "Primary",
          api_base: "https://primary.example",
          mirrors: ["https://mirror-a.example", "https://mirror-b.example"],
        }],
      }),
    }));

    const servers = await loadServers();

    expect(servers.map(server => server.apiBase)).toEqual([
      "https://primary.example",
      "https://mirror-a.example",
      "https://mirror-b.example",
    ]);
  });

  it("resolves A records through DNS-over-HTTPS JSON", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({
        Answer: [
          { type: 1, data: "203.0.113.10" },
          { type: 28, data: "2001:db8::10" },
        ],
      }),
    }));

    await expect(resolveDoH("example.com", "https://resolver.example/dns-query")).resolves.toEqual([
      "203.0.113.10",
    ]);
  });
});
