import { describe, expect, it, vi } from "vitest";
import { buildFallbackPlan } from "./selfHealing";

describe("self-healing fallback plan", () => {
  it("ends with a P2P fallback attempt", async () => {
    localStorage.clear();
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({
        servers: [{
          id: "primary",
          name: "Primary",
          api_base: "https://primary.example",
          mirrors: ["https://mirror.example"],
          bridges: ["bridge-a"],
        }],
      }),
    }));

    const plan = await buildFallbackPlan();

    expect(plan.map(item => item.kind)).toEqual(["primary", "bridge", "mirror", "bridge", "p2p"]);
  });
});
