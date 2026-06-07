// @vitest-environment happy-dom
import { beforeEach, describe, expect, it, vi } from "vitest";

describe("auth token storage", () => {
  beforeEach(() => {
    vi.resetModules();
    localStorage.clear();
  });

  it("keeps new access tokens in memory instead of localStorage", async () => {
    const { loadAccessToken, saveAccessToken } = await import("./auth");

    saveAccessToken("access-token");

    expect(loadAccessToken()).toBe("access-token");
    expect(localStorage.getItem("zenthril_token")).toBeNull();
  });

  it("migrates legacy localStorage access tokens into memory and removes them", async () => {
    localStorage.setItem("zenthril_token", "legacy-token");
    localStorage.setItem("zenthril_user", JSON.stringify({
      id: "user-1",
      username: "alice",
      public_key: "public",
    }));

    const { loadAccessToken, loadStoredAuth } = await import("./auth");
    const stored = loadStoredAuth();

    expect(stored.token).toBe("legacy-token");
    expect(loadAccessToken()).toBe("legacy-token");
    expect(localStorage.getItem("zenthril_token")).toBeNull();
  });
});
