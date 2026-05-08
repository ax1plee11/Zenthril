import { mediaService } from './mediaService';
import { useVoiceStore } from '../store/voiceStore';

const MAX_MESH_PEERS = 6;

interface PeerEntry {
  userId: string;
  pc: RTCPeerConnection;
  audio: HTMLAudioElement | null;
  qualityInterval: ReturnType<typeof setInterval> | null;
}

type IceHandler = (targetUserId: string, candidate: RTCIceCandidateInit) => void;
type TrackHandler = (userId: string, stream: MediaStream) => void;
type SwitchHandler = (reason: string) => void;

class MeshManager {
  private peers = new Map<string, PeerEntry>();
  private onIceCandidate: IceHandler | null = null;
  private onRemoteTrack: TrackHandler | null = null;
  private onSwitchNeeded: SwitchHandler | null = null;

  setHandlers(onIce: IceHandler, onTrack: TrackHandler, onSwitch: SwitchHandler): void {
    this.onIceCandidate = onIce;
    this.onRemoteTrack = onTrack;
    this.onSwitchNeeded = onSwitch;
  }

  async createOffer(
    remoteUserId: string,
    localStream: MediaStream
  ): Promise<RTCSessionDescriptionInit> {
    const entry = this.createPeerEntry(remoteUserId, localStream);
    const offer = await entry.pc.createOffer();
    await entry.pc.setLocalDescription(offer);
    return offer;
  }

  async handleOffer(
    remoteUserId: string,
    sdp: RTCSessionDescriptionInit,
    localStream: MediaStream
  ): Promise<RTCSessionDescriptionInit> {
    const entry = this.createPeerEntry(remoteUserId, localStream);
    await entry.pc.setRemoteDescription(new RTCSessionDescription(sdp));
    const answer = await entry.pc.createAnswer();
    await entry.pc.setLocalDescription(answer);
    return answer;
  }

  async handleAnswer(remoteUserId: string, sdp: RTCSessionDescriptionInit): Promise<void> {
    const entry = this.peers.get(remoteUserId);
    await entry?.pc.setRemoteDescription(new RTCSessionDescription(sdp));
  }

  async addIceCandidate(remoteUserId: string, candidate: RTCIceCandidateInit): Promise<void> {
    const entry = this.peers.get(remoteUserId);
    try {
      await entry?.pc.addIceCandidate(new RTCIceCandidate(candidate));
    } catch {
      // Ignore late ICE candidates after this peer has disconnected.
    }
  }

  removePeer(userId: string): void {
    const entry = this.peers.get(userId);
    if (!entry) return;
    if (entry.qualityInterval) clearInterval(entry.qualityInterval);
    entry.pc.close();
    if (entry.audio) {
      entry.audio.srcObject = null;
      entry.audio.remove();
    }
    this.peers.delete(userId);
  }

  getPeerCount(): number {
    return this.peers.size;
  }

  private createPeerEntry(userId: string, localStream: MediaStream): PeerEntry {
    if (this.peers.has(userId)) {
      this.removePeer(userId);
    }

    const pc = mediaService.createPeerConnection();
    mediaService.addTracksToPC(pc, localStream);

    const entry: PeerEntry = { userId, pc, audio: null, qualityInterval: null };
    this.peers.set(userId, entry);

    pc.onicecandidate = (e) => {
      if (e.candidate && this.onIceCandidate) {
        this.onIceCandidate(userId, e.candidate.toJSON());
      }
    };

    pc.ontrack = (e) => {
      const stream = e.streams[0];
      if (!stream) return;

      if (this.onRemoteTrack) this.onRemoteTrack(userId, stream);

      if (!entry.audio) {
        entry.audio = document.createElement('audio');
        entry.audio.autoplay = true;
        document.body.appendChild(entry.audio);
      }
      entry.audio.srcObject = stream;
    };

    pc.onconnectionstatechange = () => {
      const state = pc.connectionState;
      if (state === 'failed') {
        useVoiceStore.getState().updateParticipant(userId, { quality: 'disconnected' });
        this.checkMeshHealth();
      }
    };

    entry.qualityInterval = setInterval(async () => {
      const quality = await mediaService.measureConnectionQuality(pc);
      useVoiceStore.getState().updateParticipant(userId, { quality });
      if (quality === 'poor') this.checkMeshHealth();
    }, 5000);

    if (this.peers.size > MAX_MESH_PEERS) {
      this.onSwitchNeeded?.('Too many peers for mesh');
    }

    return entry;
  }

  private checkMeshHealth(): void {
    let poorCount = 0;
    const store = useVoiceStore.getState();
    const participants = store.room?.participants || [];

    participants.forEach((p) => {
      if (p.quality === 'poor' || p.quality === 'disconnected') poorCount++;
    });

    const ratio = poorCount / Math.max(participants.length, 1);
    if (ratio > 0.3) {
      this.onSwitchNeeded?.('Too many poor connections in mesh');
    }
  }

  cleanup(): void {
    this.peers.forEach((_, userId) => this.removePeer(userId));
    this.peers.clear();
  }
}

export const meshManager = new MeshManager();
