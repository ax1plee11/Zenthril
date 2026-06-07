import { createWebRTCConfig, DEFAULT_ICE_SERVERS } from "../webrtc/icePolicy";

export type DirectMessageState = "new" | "connecting" | "open" | "closed" | "failed";

export interface DirectMessageEnvelope {
  type: "p2p.direct.message";
  version: 1;
  messageId: string;
  senderDeviceId: string;
  ciphertext: string;
  sentAt: string;
}

export interface DirectMessagingOptions {
  localDeviceId: string;
  iceServers?: RTCIceServer[];
  onStateChange?: (state: DirectMessageState) => void;
  onMessage?: (message: DirectMessageEnvelope) => void;
  onIceCandidate?: (candidate: RTCIceCandidateInit) => void;
}

export class DirectMessagingPeer {
  private readonly localDeviceId: string;
  private readonly iceServers: RTCIceServer[];
  private readonly onStateChange: ((state: DirectMessageState) => void) | undefined;
  private readonly onMessage: ((message: DirectMessageEnvelope) => void) | undefined;
  private readonly onIceCandidate: ((candidate: RTCIceCandidateInit) => void) | undefined;
  private pc: RTCPeerConnection | null = null;
  private channel: RTCDataChannel | null = null;

  constructor(options: DirectMessagingOptions) {
    this.localDeviceId = options.localDeviceId;
    this.iceServers = options.iceServers ?? DEFAULT_ICE_SERVERS;
    this.onStateChange = options.onStateChange;
    this.onMessage = options.onMessage;
    this.onIceCandidate = options.onIceCandidate;
  }

  async createOffer(): Promise<RTCSessionDescriptionInit> {
    const pc = this.createPeerConnection();
    this.channel = pc.createDataChannel("zenthril-direct-v1", {
      ordered: true,
      maxRetransmits: 5,
    });
    this.bindChannel(this.channel);
    const offer = await pc.createOffer();
    await pc.setLocalDescription(offer);
    this.setState("connecting");
    return offer;
  }

  async acceptOffer(offer: RTCSessionDescriptionInit): Promise<RTCSessionDescriptionInit> {
    const pc = this.createPeerConnection();
    await pc.setRemoteDescription(new RTCSessionDescription(offer));
    const answer = await pc.createAnswer();
    await pc.setLocalDescription(answer);
    this.setState("connecting");
    return answer;
  }

  async acceptAnswer(answer: RTCSessionDescriptionInit): Promise<void> {
    await this.pc?.setRemoteDescription(new RTCSessionDescription(answer));
  }

  async addIceCandidate(candidate: RTCIceCandidateInit): Promise<void> {
    try {
      await this.pc?.addIceCandidate(new RTCIceCandidate(candidate));
    } catch {
      this.setState("failed");
    }
  }

  send(ciphertext: string): DirectMessageEnvelope {
    if (!this.channel || this.channel.readyState !== "open") {
      throw new Error("P2P direct channel is not open");
    }
    const envelope: DirectMessageEnvelope = {
      type: "p2p.direct.message",
      version: 1,
      messageId: crypto.randomUUID(),
      senderDeviceId: this.localDeviceId,
      ciphertext,
      sentAt: new Date().toISOString(),
    };
    // RESILIENCE: Direct messages are data-channel payloads that can be retried
    // through server federation or bridge nodes by higher layers when P2P fails.
    this.channel.send(JSON.stringify(envelope));
    return envelope;
  }

  close(): void {
    this.channel?.close();
    this.pc?.close();
    this.channel = null;
    this.pc = null;
    this.setState("closed");
  }

  private createPeerConnection(): RTCPeerConnection {
    this.close();
    this.pc = new RTCPeerConnection(createWebRTCConfig(this.iceServers));
    this.pc.onicecandidate = event => {
      if (event.candidate) this.onIceCandidate?.(event.candidate.toJSON());
    };
    this.pc.ondatachannel = event => {
      this.channel = event.channel;
      this.bindChannel(event.channel);
    };
    this.pc.onconnectionstatechange = () => {
      const state = this.pc?.connectionState;
      if (state === "connected") this.setState("open");
      if (state === "failed" || state === "disconnected") this.setState("failed");
      if (state === "closed") this.setState("closed");
    };
    return this.pc;
  }

  private bindChannel(channel: RTCDataChannel): void {
    channel.onopen = () => this.setState("open");
    channel.onclose = () => this.setState("closed");
    channel.onerror = () => this.setState("failed");
    channel.onmessage = event => {
      try {
        const message = JSON.parse(event.data as string) as DirectMessageEnvelope;
        if (message.type === "p2p.direct.message" && message.version === 1) {
          this.onMessage?.(message);
        }
      } catch {
        this.setState("failed");
      }
    };
  }

  private setState(state: DirectMessageState): void {
    this.onStateChange?.(state);
  }
}
