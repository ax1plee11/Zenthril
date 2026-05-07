const AUDIO_CONSTRAINTS: MediaStreamConstraints = {
  audio: {
    echoCancellation: true,
    noiseSuppression: true,
    autoGainControl: true,
    sampleRate: 48000,
    channelCount: 1,
  },
  video: false,
};

const ICE_SERVERS: RTCIceServer[] = [
  { urls: 'stun:stun.l.google.com:19302' },
  { urls: 'stun:stun1.l.google.com:19302' },
];

class MediaService {
  private localStream: MediaStream | null = null;
  private audioContext: AudioContext | null = null;
  private analyser: AnalyserNode | null = null;
  private speakingCallback: ((speaking: boolean, level: number) => void) | null = null;
  private animFrame = 0;

  async getLocalStream(): Promise<MediaStream> {
    if (this.localStream) return this.localStream;
    this.localStream = await navigator.mediaDevices.getUserMedia(AUDIO_CONSTRAINTS);
    return this.localStream;
  }

  createPeerConnection(): RTCPeerConnection {
    return new RTCPeerConnection({ iceServers: ICE_SERVERS });
  }

  addTracksToPC(pc: RTCPeerConnection, stream: MediaStream): void {
    stream.getTracks().forEach((track) => pc.addTrack(track, stream));
  }

  setMuted(muted: boolean): void {
    this.localStream?.getAudioTracks().forEach((t) => {
      t.enabled = !muted;
    });
  }

  startVoiceActivity(callback: (speaking: boolean, level: number) => void): void {
    if (!this.localStream) return;
    this.speakingCallback = callback;

    this.audioContext = new AudioContext();
    const source = this.audioContext.createMediaStreamSource(this.localStream);
    this.analyser = this.audioContext.createAnalyser();
    this.analyser.fftSize = 256;
    source.connect(this.analyser);

    const data = new Uint8Array(this.analyser.frequencyBinCount);
    const THRESHOLD = 20;

    const tick = () => {
      this.analyser!.getByteFrequencyData(data);
      const avg = data.reduce((a, b) => a + b, 0) / data.length;
      const level = Math.min(100, avg * 2);
      callback(level > THRESHOLD, level);
      this.animFrame = requestAnimationFrame(tick);
    };

    tick();
  }

  stopVoiceActivity(): void {
    cancelAnimationFrame(this.animFrame);
    this.analyser = null;
    this.audioContext?.close();
    this.audioContext = null;
    this.speakingCallback = null;
  }

  measureConnectionQuality(pc: RTCPeerConnection): Promise<'excellent' | 'good' | 'poor' | 'disconnected'> {
    return new Promise((resolve) => {
      const state = pc.connectionState;
      if (state === 'failed' || state === 'closed') {
        resolve('disconnected');
        return;
      }

      pc.getStats().then((stats) => {
        let rtt = 0;
        let packetLoss = 0;

        stats.forEach((report) => {
          if (report.type === 'remote-inbound-rtp') {
            rtt = (report.roundTripTime || 0) * 1000;
            packetLoss = report.fractionLost || 0;
          }
        });

        if (rtt < 100 && packetLoss < 0.02) resolve('excellent');
        else if (rtt < 300 && packetLoss < 0.05) resolve('good');
        else resolve('poor');
      }).catch(() => resolve('poor'));
    });
  }

  cleanup(): void {
    this.stopVoiceActivity();
    this.localStream?.getTracks().forEach((t) => t.stop());
    this.localStream = null;
  }
}

export const mediaService = new MediaService();
