import { describe, expect, it } from "vitest";
import {
  decodePaddedFrame,
  encodePaddedFrame,
} from "./padding";

describe("websocket JSON padding", () => {
  it("round-trips websocket events through a padded frame", () => {
    const frame = encodePaddedFrame({ type: "ping", channel_id: "c1" });
    const decoded = decodePaddedFrame(frame);
    expect(decoded).toEqual({ type: "ping", channel_id: "c1" });
  });

  it("preserves legacy plain JSON frames", () => {
    expect(decodePaddedFrame(JSON.stringify({ type: "pong" }))).toEqual({ type: "pong" });
  });

  it("rejects unknown padded frame modes", () => {
    const frame = JSON.stringify({
      type: "transport.frame",
      version: 1,
      mode: "unknown",
      payload: "e30=",
    });
    expect(() => decodePaddedFrame(frame)).toThrow("Unsupported padded frame");
  });
});
