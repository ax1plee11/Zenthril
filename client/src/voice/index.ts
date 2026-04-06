/**
 * VoiceClient — WebRTC голосовые каналы через WebSocket-сигнализацию
 *
 * Требования: 4.1, 4.2, 4.3, 4.4, 4.5, 4.6
 */

import { TransportLayer, WSEvent } from "../transport";

// ─── Конфигурация ICE ─────────────────────────────────────────────────────────

const ICE_SERVERS: RTCIceServer[] = [
  { urls: "stun:stun.l.google.com:19302" },
  { urls: "stun:stun1.l.google.com:19302" },
];

const STATS_INTERVAL_MS = 5_000;
const PACKET_LOSS_THRESHOLD = 0.1; // 10%

// ─── VoiceClient ──────────────────────────────────────────────────────────────

export class VoiceClient {
  private peers: Map<string, RTCPeerConnection> = new Map();
  private localStream: MediaStream | null = null;
  private channelId: string | null = null;
  private muted = false;

  private onParticipantJoinedHandler: ((userId: string) => void) | null = null;
  private onParticipantLeftHandler: ((userId: string) => void) | null = null;
  private onConnectionQualityHandler: ((quality: "good" | "poor") => void) | null = null;

  private unsubscribers: Array<() => void> = [];
  private statsTimer: ReturnType<typeof setInterval> | null = null;

  constructor(private transport: TransportLayer, private currentUserId: string) {}

  // ─── Подключение к голосовому каналу ────────────────────────────────────────

  async joinChannel(channelId: string): Promise<void> {
    if (this.channelId) {
      await this.leaveChannel();
    }

    this.localStream = await navigator.mediaDevices.getUserMedia({ audio: true, video: false });
    this.channelId = channelId;
    this.muted = false;

    // Подписываемся на голосовые события
    this.unsubscribers.push(
      this.transport.subscribe("voice.user_joined", (e) => this._onUserJoined(e)),
      this.transport.subscribe("voice.user_left", (e) => this._onUserLeft(e)),
      this.transport.subscribe("voice.signal", (e) => this._onSignal(e)),
      this.transport.subscribe("voice.ice", (e) => this._onIce(e)),
    );

    // Уведомляем сервер
    this.transport.send({ type: "voice.join", channel_id: channelId });

    // Запускаем мониторинг качества
    this._startStatsMonitor();
  }

  // ─── Отключение от голосового канала ────────────────────────────────────────

  async leaveChannel(): Promise<void> {
    if (!this.channelId) return;

    this.transport.send({ type: "voice.leave", channel_id: this.channelId });

    // Закрываем все peer connections
    this.peers.forEach((pc) => pc.close());
    this.peers.clear();

    // Останавливаем локальный поток
    this.localStream?.getTracks().forEach((t) => t.stop());
    this.localStream = null;

    // Отписываемся от событий
    this.unsubscribers.forEach((unsub) => unsub());
    this.unsubscribers = [];

    this._stopStatsMonitor();
    this.channelId = null;
  }

  // ─── Управление микрофоном ───────────────────────────────────────────────────

  async toggleMute(): Promise<boolean> {
    this.muted = !this.muted;
    this.localStream?.getAudioTracks().forEach((t) => {
      t.enabled = !this.muted;
    });
    return this.muted;
  }

  // ─── Список участников ───────────────────────────────────────────────────────

  getParticipants(): string[] {
    return Array.from(this.peers.keys());
  }

  // ─── Обработчики событий ─────────────────────────────────────────────────────

  onParticipantJoined(handler: (userId: string) => void): void {
    this.onParticipantJoinedHandler = handler;
  }

  onParticipantLeft(handler: (userId: string) => void): void {
    this.onParticipantLeftHandler = handler;
  }

  onConnectionQuality(handler: (quality: "good" | "poor") => void): void {
    this.onConnectionQualityHandler = handler;
  }

  // ─── Внутренние методы ───────────────────────────────────────────────────────

  private async _onUserJoined(event: WSEvent): Promise<void> {
    const userId = event.user_id as string;
    const channelId = event.channel_id as string;

    if (!userId || userId === this.currentUserId || channelId !== this.channelId) return;

    // Создаём peer connection и отправляем offer
    const pc = this._createPeerConnection(userId);
    this.peers.set(userId, pc);

    const offer = await pc.createOffer();
    await pc.setLocalDescription(offer);

    this.transport.send({
      type: "voice.signal",
      channel_id: this.channelId!,
      target_user_id: userId,
      sdp: offer,
    });

    this.onParticipantJoinedHandler?.(userId);
  }

  private async _onUserLeft(event: WSEvent): Promise<void> {
    const userId = event.user_id as string;
    const channelId = event.channel_id as string;

    if (!userId || channelId !== this.channelId) return;

    const pc = this.peers.get(userId);
    if (pc) {
      pc.close();
      this.peers.delete(userId);
    }

    this.onParticipantLeftHandler?.(userId);
  }

  private async _onSignal(event: WSEvent): Promise<void> {
    const fromUserId = event.from_user_id as string;
    const channelId = event.channel_id as string;
    const sdp = event.sdp as RTCSessionDescriptionInit;

    if (!fromUserId || channelId !== this.channelId || !sdp) return;

    let pc = this.peers.get(fromUserId);

    if (sdp.type === "offer") {
      // Получили offer — создаём PC если нет, устанавливаем remote description, отправляем answer
      if (!pc) {
        pc = this._createPeerConnection(fromUserId);
        this.peers.set(fromUserId, pc);
        this.onParticipantJoinedHandler?.(fromUserId);
      }

      await pc.setRemoteDescription(new RTCSessionDescription(sdp));
      const answer = await pc.createAnswer();
      await pc.setLocalDescription(answer);

      this.transport.send({
        type: "voice.signal",
        channel_id: this.channelId!,
        target_user_id: fromUserId,
        sdp: answer,
      });
    } else if (sdp.type === "answer" && pc) {
      await pc.setRemoteDescription(new RTCSessionDescription(sdp));
    }
  }

  private async _onIce(event: WSEvent): Promise<void> {
    const fromUserId = event.from_user_id as string;
    const channelId = event.channel_id as string;
    const candidate = event.candidate as RTCIceCandidateInit;

    if (!fromUserId || channelId !== this.channelId || !candidate) return;

    const pc = this.peers.get(fromUserId);
    if (pc) {
      await pc.addIceCandidate(new RTCIceCandidate(candidate));
    }
  }

  private _createPeerConnection(userId: string): RTCPeerConnection {
    const pc = new RTCPeerConnection({ iceServers: ICE_SERVERS });

    // Добавляем локальные треки
    this.localStream?.getTracks().forEach((track) => {
      pc.addTrack(track, this.localStream!);
    });

    // Отправляем ICE кандидаты
    pc.onicecandidate = (e) => {
      if (e.candidate && this.channelId) {
        this.transport.send({
          type: "voice.ice",
          channel_id: this.channelId,
          target_user_id: userId,
          candidate: e.candidate.toJSON(),
        });
      }
    };

    // Воспроизводим входящий аудиопоток
    pc.ontrack = (e) => {
      const audio = new Audio();
      audio.srcObject = e.streams[0];
      audio.play().catch(() => {/* autoplay policy */});
    };

    return pc;
  }

  // ─── Мониторинг качества соединения ─────────────────────────────────────────

  private _startStatsMonitor(): void {
    this._stopStatsMonitor();
    this.statsTimer = setInterval(() => this._checkStats(), STATS_INTERVAL_MS);
  }

  private _stopStatsMonitor(): void {
    if (this.statsTimer !== null) {
      clearInterval(this.statsTimer);
      this.statsTimer = null;
    }
  }

  private async _checkStats(): Promise<void> {
    if (!this.onConnectionQualityHandler) return;

    for (const pc of this.peers.values()) {
      const stats = await pc.getStats();
      let quality: "good" | "poor" = "good";

      stats.forEach((report) => {
        if (report.type === "inbound-rtp" && report.kind === "audio") {
          const packetsLost: number = report.packetsLost ?? 0;
          const packetsReceived: number = report.packetsReceived ?? 0;
          const total = packetsLost + packetsReceived;
          if (total > 0 && packetsLost / total > PACKET_LOSS_THRESHOLD) {
            quality = "poor";
          }
        }
      });

      this.onConnectionQualityHandler(quality);
    }
  }
}
