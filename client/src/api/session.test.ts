// @vitest-environment happy-dom
import { beforeEach, describe, expect, it, vi } from "vitest";
import { AUTH_SESSION_EXPIRED_EVENT } from "../store/auth";

describe("API session refresh", () => {
  beforeEach(() => {
    vi.resetModules();
    localStorage.clear();
  });

  it("refreshes the access token with HttpOnly cookies and retries once on 401", async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(serverListResponse())
      .mockResolvedValueOnce(jsonResponse(401, { error: "unauthorized" }))
      .mockResolvedValueOnce(jsonResponse(200, {
        access_token: "new-access",
        token: "new-access",
        expires_in: 900,
      }))
      .mockResolvedValueOnce(jsonResponse(200, []));
    vi.stubGlobal("fetch", fetchMock);
    localStorage.setItem("zenthril_token", "old-access");

    const { api } = await import("./index");
    await expect(api.users.search("alice")).resolves.toEqual([]);

    expect(localStorage.getItem("zenthril_token")).toBeNull();
    expect(fetchMock).toHaveBeenNthCalledWith(
      3,
      "http://localhost:8080/api/v1/auth/refresh",
      expect.objectContaining({
        credentials: "include",
        method: "POST",
      }),
    );
    expect(fetchMock).toHaveBeenNthCalledWith(
      4,
      "http://localhost:8080/api/v1/users/search?q=alice",
      expect.objectContaining({
        credentials: "include",
        headers: expect.objectContaining({
          Authorization: "Bearer new-access",
        }),
      }),
    );
  });

  it("clears local auth and emits session-expired when refresh fails", async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(serverListResponse())
      .mockResolvedValueOnce(jsonResponse(401, { error: "unauthorized" }))
      .mockResolvedValueOnce(jsonResponse(401, { error: "invalid_refresh_token" }));
    vi.stubGlobal("fetch", fetchMock);
    const expired = vi.fn();
    window.addEventListener(AUTH_SESSION_EXPIRED_EVENT, expired);
    localStorage.setItem("zenthril_token", "old-access");
    localStorage.setItem("zenthril_user", JSON.stringify({
      id: "user-1",
      username: "alice",
      public_key: "public",
    }));

    const { api } = await import("./index");
    await expect(api.users.search("alice")).rejects.toMatchObject({
      status: 401,
    });

    expect(expired).toHaveBeenCalledTimes(1);
    expect(localStorage.getItem("zenthril_token")).toBeNull();
    expect(localStorage.getItem("zenthril_user")).toBeNull();
    window.removeEventListener(AUTH_SESSION_EXPIRED_EVENT, expired);
  });

  it("calls logout-all with credentials and bearer access token", async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(serverListResponse())
      .mockResolvedValueOnce(jsonResponse(204, undefined));
    vi.stubGlobal("fetch", fetchMock);

    const { api } = await import("./index");
    const { saveAccessToken } = await import("../store/auth");
    saveAccessToken("access-token");

    await expect(api.auth.logoutAll()).resolves.toBeUndefined();

    expect(fetchMock).toHaveBeenNthCalledWith(
      2,
      "http://localhost:8080/api/v1/auth/logout-all",
      expect.objectContaining({
        credentials: "include",
        method: "POST",
        headers: expect.objectContaining({
          Authorization: "Bearer access-token",
        }),
      }),
    );
  });
});

function serverListResponse(): Response {
  return jsonResponse(200, {
    servers: [{
      id: "local",
      name: "Local",
      api_base: "http://localhost:8080",
      ws_base: "ws://localhost:8080",
    }],
  });
}

function jsonResponse(status: number, body: unknown): Response {
  return {
    ok: status >= 200 && status < 300,
    status,
    json: async () => body,
  } as Response;
}
