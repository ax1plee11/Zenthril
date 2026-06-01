import { beforeEach, describe, expect, it } from "vitest";
import {
  loadAutoConnectOnStartup,
  saveAutoConnectOnStartup,
  shouldStartOnline,
} from "./startupPrivacy";

describe("startup privacy policy", () => {
  beforeEach(() => {
    localStorage.clear();
  });

  it("disables auto-connect by default", () => {
    expect(loadAutoConnectOnStartup()).toBe(false);
    expect(shouldStartOnline(true)).toBe(false);
  });

  it("requires a stored session even when auto-connect is enabled", () => {
    saveAutoConnectOnStartup(true);

    expect(shouldStartOnline(false)).toBe(false);
    expect(shouldStartOnline(true)).toBe(true);
  });

  it("persists explicit auto-connect preference", () => {
    saveAutoConnectOnStartup(true);
    expect(loadAutoConnectOnStartup()).toBe(true);

    saveAutoConnectOnStartup(false);
    expect(loadAutoConnectOnStartup()).toBe(false);
  });
});

