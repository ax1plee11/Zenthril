// @vitest-environment happy-dom
import { createRoot } from "react-dom/client";
import { act } from "react-dom/test-utils";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const connectMock = vi.fn();
const getActiveServerMock = vi.fn();
const restoreAccessTokenMock = vi.fn();

vi.mock("./components/MainLayout", () => ({
  default: () => {
    fetch("https://example.invalid/should-not-run");
    return <div>Main layout</div>;
  },
}));

vi.mock("./components/AuthScreen", () => ({
  default: ({ sessionNotice }: { sessionNotice?: string | null }) => (
    <div>
      Auth screen
      {sessionNotice && <span>{sessionNotice}</span>}
    </div>
  ),
}));

vi.mock("./features/calls/components/CallManager", () => ({
  CallManager: () => null,
}));

vi.mock("./features/calls/services/signalingService", () => ({
  signalingService: {
    connect: connectMock,
    disconnect: vi.fn(),
  },
}));

vi.mock("./api/index", () => ({
  getActiveServer: getActiveServerMock,
  restoreAccessTokenFromRefreshCookie: restoreAccessTokenMock,
}));

describe("App privacy-first startup", () => {
  let rootEl: HTMLDivElement;

  beforeEach(() => {
    vi.resetModules();
    localStorage.clear();
    connectMock.mockClear();
    getActiveServerMock.mockClear();
    restoreAccessTokenMock.mockClear();
    vi.stubGlobal("fetch", vi.fn());
    rootEl = document.createElement("div");
    document.body.appendChild(rootEl);
  });

  afterEach(() => {
    document.body.innerHTML = "";
    vi.unstubAllGlobals();
  });

  it("does not mount networked layout or connect signaling for a stored session by default", async () => {
    localStorage.setItem("zenthril_token", "stored-token");
    localStorage.setItem("zenthril_user", JSON.stringify({
      id: "user-1",
      username: "nurbek",
      public_key: "public",
    }));

    const { default: App } = await import("./App");

    await act(async () => {
      createRoot(rootEl).render(<App />);
    });

    expect(rootEl.textContent).toContain("Zenthril is offline");
    expect(rootEl.textContent).not.toContain("Main layout");
    expect(fetch).not.toHaveBeenCalled();
    expect(getActiveServerMock).not.toHaveBeenCalled();
    expect(connectMock).not.toHaveBeenCalled();
  });

  it("keeps a cookie-backed stored user offline without restoring until connect", async () => {
    localStorage.setItem("zenthril_user", JSON.stringify({
      id: "user-1",
      username: "nurbek",
      public_key: "public",
    }));

    const { default: App } = await import("./App");

    await act(async () => {
      createRoot(rootEl).render(<App />);
    });

    expect(rootEl.textContent).toContain("Zenthril is offline");
    expect(rootEl.textContent).not.toContain("Auth screen");
    expect(restoreAccessTokenMock).not.toHaveBeenCalled();
    expect(connectMock).not.toHaveBeenCalled();
  });

  it("restores a memory access token from refresh cookie when auto-connect is enabled", async () => {
    localStorage.setItem("zenthril_auto_connect_on_startup", "true");
    localStorage.setItem("zenthril_user", JSON.stringify({
      id: "user-1",
      username: "nurbek",
      public_key: "public",
    }));
    restoreAccessTokenMock.mockResolvedValue("restored-access");
    getActiveServerMock.mockResolvedValue({ wsBase: "ws://localhost:8080" });

    const { default: App } = await import("./App");

    await act(async () => {
      createRoot(rootEl).render(<App />);
    });
    await act(async () => {
      await Promise.resolve();
    });

    expect(restoreAccessTokenMock).toHaveBeenCalledTimes(1);
    expect(connectMock).toHaveBeenCalledWith("ws://localhost:8080");
    expect(rootEl.textContent).toContain("Main layout");
  });

  it("shows a clear message when the stored session expires", async () => {
    localStorage.setItem("zenthril_token", "stored-token");
    localStorage.setItem("zenthril_user", JSON.stringify({
      id: "user-1",
      username: "nurbek",
      public_key: "public",
    }));

    const { AUTH_SESSION_EXPIRED_EVENT } = await import("./store/auth");
    const { default: App } = await import("./App");

    await act(async () => {
      createRoot(rootEl).render(<App />);
    });

    await act(async () => {
      window.dispatchEvent(new CustomEvent(AUTH_SESSION_EXPIRED_EVENT));
    });

    expect(rootEl.textContent).toContain("Auth screen");
    expect(rootEl.textContent).toContain("Your session expired or was revoked");
    expect(localStorage.getItem("zenthril_token")).toBeNull();
    expect(localStorage.getItem("zenthril_user")).toBeNull();
  });
});
