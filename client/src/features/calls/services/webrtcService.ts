import { signalingService } from './signalingService';
import { useCallStore } from '../store/callStore';

const ICE_SERVERS: RTCIceServer[] = [
  { urls: 'stun:stun.l.google.com:19302' },
  { urls: 'stun:stun1.l.google.com:19302' },
];

class WebRTCService {
  private pc: RTCPeerConnection | null = null;
  private localStream: MediaStream | null = null;
  private remoteStream: MediaStream | null = null;
  private callId: string | null = null;
  private remoteAudioEl: HTMLAudioElement | null = null;

  async startCall(callId: string): Promise<void> {
    this.callId = callId;
    await this.initLocalStream();
    this.createPeerConnection();

    const offer = await this.pc!.createOffer();
    await this.pc!.setLocalDescription(offer);
    signalingService.sendOffer(callId, offer);
  }

  async handleOffer(callId: string, sdp: RTCSessionDescriptionInit): Promise<void> {
    this.callId = callId;
    await this.initLocalStream();
    this.createPeerConnection();

    await this.pc!.setRemoteDescription(new RTCSessionDescription(sdp));
    const answer = await this.pc!.createAnswer();
    await this.pc!.setLocalDescription(answer);
    signalingService.sendAnswer(callId, answer);
  }

  async handleAnswer(sdp: RTCSessionDescriptionInit): Promise<void> {
    if (!this.pc) return;
    await this.pc.setRemoteDescription(new RTCSessionDescription(sdp));
  }

  async handleIceCandidate(candidate: RTCIceCandidateInit): Promise<void> {
    if (!this.pc) return;
    try {
      await this.pc.addIceCandidate(new RTCIceCandidate(candidate));
    } catch {}
  }

  private async initLocalStream(): Promise<void> {
    try {
      this.localStream = await navigator.mediaDevices.getUserMedia({
        audio: {
          echoCancellation: true,
          noiseSuppression: true,
          autoGainControl: true,
        },
        video: false,
      });
    } catch (err) {
      useCallStore.getState().updateCallState('failed');
      throw new Error('Microphone access denied');
    }
  }

  private createPeerConnection(): void {
    this.pc = new RTCPeerConnection({ iceServers: ICE_SERVERS });

    this.localStream?.getTracks().forEach((track) => {
      this.pc!.addTrack(track, this.localStream!);
    });

    this.pc.onicecandidate = (event) => {
      if (event.candidate && this.callId) {
        signalingService.sendIceCandidate(this.callId, event.candidate.toJSON());
      }
    };

    this.pc.ontrack = (event) => {
      this.remoteStream = event.streams[0];
      this.playRemoteAudio();
      useCallStore.getState().updateCallState('connected');
    };

    this.pc.onconnectionstatechange = () => {
      const state = this.pc?.connectionState;
      if (state === 'failed' || state === 'disconnected') {
        useCallStore.getState().updateCallState('failed');
      }
    };

    this.pc.oniceconnectionstatechange = () => {
      const state = this.pc?.iceConnectionState;
      if (state === 'connected' || state === 'completed') {
        useCallStore.getState().updateCallState('connected');
      }
    };
  }

  private playRemoteAudio(): void {
    if (!this.remoteStream) return;
    if (!this.remoteAudioEl) {
      this.remoteAudioEl = document.createElement('audio');
      this.remoteAudioEl.autoplay = true;
      document.body.appendChild(this.remoteAudioEl);
    }
    this.remoteAudioEl.srcObject = this.remoteStream;
  }

  setMuted(muted: boolean): void {
    this.localStream?.getAudioTracks().forEach((t) => {
      t.enabled = !muted;
    });
  }

  cleanup(): void {
    this.localStream?.getTracks().forEach((t) => t.stop());
    this.pc?.close();
    if (this.remoteAudioEl) {
      this.remoteAudioEl.srcObject = null;
      this.remoteAudioEl.remove();
      this.remoteAudioEl = null;
    }
    this.pc = null;
    this.localStream = null;
    this.remoteStream = null;
    this.callId = null;
  }

  getLocalStream(): MediaStream | null {
    return this.localStream;
  }
}

export const webrtcService = new WebRTCService();
