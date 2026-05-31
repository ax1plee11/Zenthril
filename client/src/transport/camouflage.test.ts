import { describe, expect, it, vi } from "vitest";
import {
  decodeCamouflageFrame,
  encodeCamouflageFrame,
} from "./camouflage";

describe("websocket camouflage", () => {
  it("round-trips websocket events through a padded frame", () => {
    const frame = encodeCamouflageFrame({ type: "ping", channel_id: "c1" });
    const decoded = decodeCamouflageFrame(frame);

    expect(decoded).toEqual({ type: "ping", channel_id: "c1" });
    expect(JSON.parse(frame)).toMatchObject({
      type: "transport.frame",
      version: 1,
      mode: "json-padding-v1",
    });
  });

  it("passes plain JSON events through for backwards compatibility", () => {
    expect(decodeCamouflageFrame(JSON.stringify({ type: "pong" }))).toEqual({ type: "pong" });
  });

  it("rejects unknown camouflage modes", () => {
    vi.spyOn(console, "error").mockImplementation(() => {});
    const frame = JSON.stringify({
      type: "transport.frame",
      version: 99,
      mode: "unknown",
      payload: "e30=",
      padding: "",
    });

    expect(() => decodeCamouflageFrame(frame)).toThrow("Unsupported camouflage frame");
  });
});
