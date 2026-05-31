import { describe, expect, it, vi } from "vitest";
import { buildRecoveryPlan } from "./selfHealing";

describe("self-healing recovery plan", () => {
  it("uses configured backup endpoints without automatic peer escalation", async () => {
    localStorage.clear();
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({
        servers: [{
          id: "primary",
          name: "Primary",
          api_base: "https://primary.example",
          backup_endpoints: ["https://backup.example"],
        }],
      }),
    }));

    const plan = await buildRecoveryPlan();

    expect(plan.map(item => item.kind)).toEqual(["primary", "backup"]);
  });
});
