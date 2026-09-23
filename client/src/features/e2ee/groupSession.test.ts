import { describe, expect, it } from "vitest";
import {
  createGroupSession,
  decryptGroupMessage,
  encryptGroupMessage,
  nextGroupMessageKey,
  restoreGroupSession,
  serializeGroupSession,
} from "./groupSession";

describe("group session foundation", () => {
  const members = [
    { userId: "alice", deviceId: "alice-device" },
    { userId: "bob", deviceId: "bob-device" },
    { userId: "charlie", deviceId: "charlie-device" },
  ];

  it("creates a group session with deterministic member list", () => {
    const session = createGroupSession("group-1", members);
    expect(session.groupId).toBe("group-1");
    expect(session.members).toHaveLength(3);
    expect(session.messageCounter).toBe(0);
    expect(session.chainKey.length).toBe(32);
  });

  it("produces unique message keys for consecutive messages", () => {
    let session = createGroupSession("group-1", members);
    const keys = [];
    for (let i = 0; i < 5; i++) {
      const step = nextGroupMessageKey(session);
      keys.push(Array.from(step.key));
      session = step.state;
    }
    const unique = new Set(keys.map((k) => JSON.stringify(k)));
    expect(unique.size).toBe(5);
  });

  it("advances chain key after each message", () => {
    const session = createGroupSession("group-1", members);
    const first = nextGroupMessageKey(session);
    const second = nextGroupMessageKey(first.state);
    expect(Array.from(first.state.chainKey)).not.toEqual(Array.from(session.chainKey));
    expect(Array.from(second.state.chainKey)).not.toEqual(Array.from(first.state.chainKey));
  });

  it("encrypts and decrypts a group message", async () => {
    const senderSession = createGroupSession("group-1", members);
    const receiverSession = {
      ...senderSession,
      chainKey: new Uint8Array(senderSession.chainKey),
      messageCounter: 0,
      members: senderSession.members.map((m) => ({ ...m })),
    };
    const encrypted = await encryptGroupMessage("Hello group!", senderSession, {
      senderUserId: "alice",
      senderDeviceId: "alice-device",
    });
    expect(encrypted.payload.ciphertext).toBeTruthy();
    expect(encrypted.state.messageCounter).toBe(1);

    const decrypted = await decryptGroupMessage(encrypted.payload, receiverSession);
    expect(decrypted).toBe("Hello group!");
    expect(receiverSession.messageCounter).toBe(1);
  });

  it("handles consecutive group messages in order", async () => {
    let senderState = createGroupSession("group-1", members);
    const receiverState = {
      ...senderState,
      chainKey: new Uint8Array(senderState.chainKey),
      messageCounter: 0,
      members: senderState.members.map((m) => ({ ...m })),
    };

    const msg0 = await encryptGroupMessage("msg0", senderState);
    senderState = msg0.state;
    const msg1 = await encryptGroupMessage("msg1", senderState);
    senderState = msg1.state;
    const msg2 = await encryptGroupMessage("msg2", senderState);

    const decrypted0 = await decryptGroupMessage(msg0.payload, receiverState);
    expect(decrypted0).toBe("msg0");
    expect(receiverState.messageCounter).toBe(1);

    const decrypted1 = await decryptGroupMessage(msg1.payload, receiverState);
    expect(decrypted1).toBe("msg1");
    expect(receiverState.messageCounter).toBe(2);

    const decrypted2 = await decryptGroupMessage(msg2.payload, receiverState);
    expect(decrypted2).toBe("msg2");
    expect(receiverState.messageCounter).toBe(3);
  });

  it("serializes and restores group session", () => {
    const session = createGroupSession("group-1", members);
    const step = nextGroupMessageKey(session);
    const stored = serializeGroupSession(step.state);
    const restored = restoreGroupSession(stored);
    expect(restored.groupId).toBe(stored.groupId);
    expect(restored.messageCounter).toBe(stored.messageCounter);
    expect(Array.from(restored.chainKey)).toEqual(Array.from(stored.chainKey ? base64ToBuffer(stored.chainKey) : restored.chainKey));
  });

  it("fail-closed when group exceeds device limit", () => {
    const manyMembers = Array.from({ length: 11 }, (_, i) => ({ userId: `user-${i}`, deviceId: `device-${i}` }));
    expect(() => createGroupSession("big-group", manyMembers)).toThrow("Group exceeds maximum device limit");
  });

  it("fail-closed on unsupported version", () => {
    const session = createGroupSession("group-1", members);
    (session as any).version = 99;
    expect(() => nextGroupMessageKey(session)).toThrow("Unsupported group session version");
  });

  it("rejects unsupported session version on nextGroupMessageKey", () => {
    const session = createGroupSession("group-1", members);
    (session as any).version = 99;
    expect(() => nextGroupMessageKey(session)).toThrow("Unsupported group session version");
  });

  it("rejects unsupported session version on decrypt", async () => {
    const session = createGroupSession("group-1", members);
    const encrypted = await encryptGroupMessage("secret", session);
    const badSession = { ...session, version: 99 as any };
    await expect(decryptGroupMessage(encrypted.payload, badSession)).rejects.toThrow("Unsupported group session version");
  });

  it("rejects excessive skipped group messages", async () => {
    let senderState = createGroupSession("group-1", members);
    const encrypted = await encryptGroupMessage("msg0", senderState);
    const receiverState = {
      ...senderState,
      chainKey: new Uint8Array(senderState.chainKey),
      messageCounter: 0,
      members: senderState.members.map((m) => ({ ...m })),
    };
    await decryptGroupMessage(encrypted.payload, receiverState);

    const farFuturePayload = { ...encrypted.payload, clientMessageId: "group-1:100" };
    await expect(decryptGroupMessage(farFuturePayload, receiverState)).rejects.toThrow("Skipped group message key limit exceeded");
  });
});

function base64ToBuffer(base64: string): Uint8Array {
  const binary = atob(base64);
  const bytes = new Uint8Array(binary.length);
  for (let i = 0; i < binary.length; i++) {
    bytes[i] = binary.charCodeAt(i);
  }
  return bytes;
}
