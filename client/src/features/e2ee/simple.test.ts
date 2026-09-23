import { describe, expect, it } from "vitest";
import { encrypt, decrypt } from "../../crypto";

describe("encrypt decrypt simple", () => {
  it("works with basic AAD", async () => {
    const key = await crypto.subtle.generateKey(
      { name: "AES-GCM", length: 256 },
      false,
      ["encrypt", "decrypt"],
    );

    const payload = await encrypt("hello", key, {
      channelId: "ch1",
      senderUserId: "u1",
      senderDeviceId: "d1",
      sessionId: "s1",
      clientMessageId: "m1",
    });

    const decrypted = await decrypt(payload, key);
    expect(decrypted).toBe("hello");
  });
});
