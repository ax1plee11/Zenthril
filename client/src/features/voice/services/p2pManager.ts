import { mediaService } from './mediaService';
import { useVoiceStore } from '../store/voiceStore';

type IceHandler = (candidate: RTCIceCandidateInit) => void;
type TrackHandler = (userId: string, stream: MediaStream) => void;

class P2PManager {
  private pc: RTCPeerConnection | null = null;
  private remoteUserId: string | null = null;
  private onIceCandidate: IceHandler | null = null;
  private onRemoteTrack: TrackHandler | null = null;
  private remoteAudio: HTMLAudioElement | null = null;
  private qualityInterval: ReturnType<typeof setInterval> | null = null;

  setHandlers(onIce: IceHandler, onTrack: TrackHandler): void {
    this.onIceCandidate = onIce;
    this.onRemoteTrack = onTrack;
  }

  async createOffer(remoteUserId: string, localStream: MediaStream): Promise<RTCSessionDescriptionInit> {
    this.remoteUserId = remoteUserId;
    this.pc = mediaService.createPeerConnection();
    this.setupPC();
    mediaService.addTracksToPC(this.pc, localStream);

    const offer = await this.pc.createOffer();
    await this.pc.setLocalDescription(offer);
    this.startQualityMonitor();
    return offer;
  }

  async handleOffer(
    remoteUserId: string,
    sdp: RTCSessionDescriptionInit,
    localStream: MediaStream
  ): Promise<RTCSessionDescriptionInit> {
    this.remoteUserId = remoteUserId;
    this.pc = mediaService.createPeerConnection();
    this.setupPC();
    mediaService.addTracksToPC(this.pc, localStream);

    await this.pc.setRemoteDescription(new RTCSessionDescription(sdp));
    const answer = await this.pc.createAnswer();
    await this.pc.setLocalDescription(answer);
    this.startQualityMonitor();
    return answer;
  }

  async handleAnswer(sdp: RTCSessionDescriptionInit): Promise<void> {
    await this.pc?.setRemoteDescription(new RTCSessionDescription(sdp));
  }

  async addIceCandidate(candidate: RTCIceCandidateInit): Promise<void> {
    try {
      await this.pc?.addIceCandidate(new RTCIceCandidate(candidate));
    } catch {
      // Ignore late ICE candidates after this call has disconnected.
    }
  }

  private setupPC(): void {
    if (!this.pc) return;

    this.pc.onicecandidate = (e) => {
      if (e.candidate && this.onIceCandidate) {
        this.onIceCandidate(e.candidate.toJSON());
      }
    };

    this.pc.ontrack = (e) => {
      const stream = e.streams[0];
      if (stream && this.remoteUserId && this.onRemoteTrack) {
        this.onRemoteTrack(this.remoteUserId, stream);
        this.playRemoteAudio(stream);
      }
    };

    this.pc.onconnectionstatechange = () => {
      const state = this.pc?.connectionState;
      if (state === 'connected') {
        useVoiceStore.getState().setConnecting(false);
      } else if (state === 'failed') {
        useVoiceStore.getState().updateParticipant(this.remoteUserId!, { quality: 'disconnected' });
      }
    };
  }

  private playRemoteAudio(stream: MediaStream): void {
    if (!this.remoteAudio) {
      this.remoteAudio = document.createElement('audio');
      this.remoteAudio.autoplay = true;
      document.body.appendChild(this.remoteAudio);
    }
    this.remoteAudio.srcObject = stream;
  }

  private startQualityMonitor(): void {
    this.qualityInterval = setInterval(async () => {
      if (!this.pc || !this.remoteUserId) return;
      const quality = await mediaService.measureConnectionQuality(this.pc);
      useVoiceStore.getState().updateParticipant(this.remoteUserId, { quality });
    }, 5000);
  }

  cleanup(): void {
    if (this.qualityInterval) clearInterval(this.qualityInterval);
    this.pc?.close();
    this.pc = null;
    if (this.remoteAudio) {
      this.remoteAudio.srcObject = null;
      this.remoteAudio.remove();
      this.remoteAudio = null;
    }
    this.remoteUserId = null;
  }
}

export const p2pManager = new P2PManager();
