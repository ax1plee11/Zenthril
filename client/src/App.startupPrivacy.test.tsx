// @vitest-environment happy-dom
import { createRoot } from "react-dom/client";
import { act } from "react-dom/test-utils";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const connectMock = vi.fn();
const getActiveServerMock = vi.fn();

vi.mock("./components/MainLayout", () => ({
  default: () => {
    fetch("https://example.invalid/should-not-run");
    return <div>Main layout</div>;
  },
}));

vi.mock("./components/AuthScreen", () => ({
  default: () => <div>Auth screen</div>,
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
}));

describe("App privacy-first startup", () => {
  let rootEl: HTMLDivElement;

  beforeEach(() => {
    localStorage.clear();
    connectMock.mockClear();
    getActiveServerMock.mockClear();
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
});
