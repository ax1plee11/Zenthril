import { useVoiceStore, selectVoiceMode } from '../store/voiceStore';
import { p2pManager } from './p2pManager';
import { meshManager } from './meshManager';
import { mediaService } from './mediaService';
import type { VoiceMode, VoiceParticipant } from '../types';

type EventHandler = (data: unknown) => void;

class VoiceSignalingService {
  private ws: WebSocket | null = null;
  private handlers = new Map<string, EventHandler[]>();
  private currentRoomId = '';

  connect(wsUrl: string, token: string, _userId: string): void {
    void _userId;
    this.ws = new WebSocket(`${wsUrl}/ws?ticket=${token}`);

    this.ws.onmessage = (e) => {
      try {
        const msg = JSON.parse(e.data);
        this.handleMessage(msg);
      } catch {
        // Ignore malformed voice signaling messages.
      }
    };
  }

  private send(type: string, payload: Record<string, unknown>): void {
    if (this.ws?.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify({ type, ...payload }));
    }
  }

  private async handleMessage(msg: { type: string; [k: string]: unknown }): Promise<void> {
    const store = useVoiceStore.getState();

    switch (msg.type) {
      case 'room:joined': {
        const participants = msg.participants as VoiceParticipant[];
        const mode = selectVoiceMode(participants.length + 1);
        store.setMode(mode);

        for (const p of participants) {
          store.addParticipant(p);
          await this.initiateConnectionTo(p.userId, mode);
        }
        break;
      }

      case 'room:participant-joined': {
        const p = msg.participant as VoiceParticipant;
        store.addParticipant(p);
        const newCount = (store.room?.participants.length || 0) + 1;
        const newMode = selectVoiceMode(newCount);
        if (newMode !== store.mode) {
          await this.switchMode(newMode, 'Participant count changed');
        } else {
          await this.initiateConnectionTo(p.userId, store.mode);
        }
        break;
      }

      case 'room:participant-left': {
        const userId = msg.userId as string;
        store.removeParticipant(userId);
        p2pManager.cleanup();
        meshManager.removePeer(userId);
        break;
      }

      case 'voice:mode-switch': {
        const mode = msg.mode as VoiceMode;
        await this.switchMode(mode, msg.reason as string);
        break;
      }

      case 'webrtc:offer': {
        const fromId = msg.fromUserId as string;
        const sdp = msg.sdp as RTCSessionDescriptionInit;
        const stream = store.localStream;
        if (!stream) break;

        let answer: RTCSessionDescriptionInit;
        if (store.mode === 'p2p') {
          answer = await p2pManager.handleOffer(fromId, sdp, stream);
        } else {
          answer = await meshManager.handleOffer(fromId, sdp, stream);
        }

        this.send('webrtc:answer', {
          targetUserId: fromId,
          roomId: this.currentRoomId,
          sdp: answer,
        });
        break;
      }

      case 'webrtc:answer': {
        const fromId = msg.fromUserId as string;
        const sdp = msg.sdp as RTCSessionDescriptionInit;
        if (store.mode === 'p2p') {
          await p2pManager.handleAnswer(sdp);
        } else {
          await meshManager.handleAnswer(fromId, sdp);
        }
        break;
      }

      case 'webrtc:ice-candidate': {
        const fromId = msg.fromUserId as string;
        const candidate = msg.candidate as RTCIceCandidateInit;
        if (store.mode === 'p2p') {
          await p2pManager.addIceCandidate(candidate);
        } else {
          await meshManager.addIceCandidate(fromId, candidate);
        }
        break;
      }
    }

    this.emit(msg.type, msg);
  }

  private async initiateConnectionTo(remoteUserId: string, mode: VoiceMode): Promise<void> {
    const store = useVoiceStore.getState();
    const stream = store.localStream;
    if (!stream) return;

    if (mode === 'p2p') {
      p2pManager.setHandlers(
        (candidate) => this.send('webrtc:ice-candidate', {
          targetUserId: remoteUserId,
          roomId: this.currentRoomId,
          candidate,
        }),
        (userId, remoteStream) => {
          store.updateParticipant(userId, { quality: 'good' });
          void remoteStream;
        }
      );
      const offer = await p2pManager.createOffer(remoteUserId, stream);
      this.send('webrtc:offer', {
        targetUserId: remoteUserId,
        roomId: this.currentRoomId,
        sdp: offer,
      });
    } else if (mode === 'mesh') {
      meshManager.setHandlers(
        (targetId, candidate) => this.send('webrtc:ice-candidate', {
          targetUserId: targetId,
          roomId: this.currentRoomId,
          candidate,
        }),
        (userId, remoteStream) => {
          store.updateParticipant(userId, { quality: 'good' });
          void remoteStream;
        },
        (reason) => {
          this.send('voice:mode-switch', {
            roomId: this.currentRoomId,
            mode: 'sfu',
            reason,
          });
        }
      );
      const offer = await meshManager.createOffer(remoteUserId, stream);
      this.send('webrtc:offer', {
        targetUserId: remoteUserId,
        roomId: this.currentRoomId,
        sdp: offer,
      });
    }
  }

  private async switchMode(newMode: VoiceMode, reason: string): Promise<void> {
    const store = useVoiceStore.getState();
    const oldMode = store.mode;

    if (oldMode === newMode) return;

    p2pManager.cleanup();
    meshManager.cleanup();

    store.setMode(newMode);
    store.setConnecting(true);

    const participants = store.room?.participants || [];
    for (const p of participants) {
      await this.initiateConnectionTo(p.userId, newMode);
    }

    store.setConnecting(false);
    console.log(`[Voice] Mode switched: ${oldMode} → ${newMode} (${reason})`);
  }

  async joinRoom(roomId: string, channelId: string, userId: string, username: string): Promise<void> {
    this.currentRoomId = roomId;
    const store = useVoiceStore.getState();

    store.setConnecting(true);

    const stream = await mediaService.getLocalStream();
    store.setLocalStream(stream);

    mediaService.startVoiceActivity((speaking, level) => {
      store.setSpeaking(speaking);
      store.updateParticipant(userId, { isSpeaking: speaking, audioLevel: level });
    });

    this.send('room:join', { roomId, channelId, userId, username });

    store.setRoom({
      roomId,
      channelId,
      mode: 'p2p',
      participants: [],
      createdAt: Date.now(),
    });
  }

  leaveRoom(): void {
    const store = useVoiceStore.getState();
    if (!this.currentRoomId) return;

    this.send('room:leave', { roomId: this.currentRoomId });

    p2pManager.cleanup();
    meshManager.cleanup();
    mediaService.stopVoiceActivity();
    mediaService.cleanup();

    store.clearRoom();
    this.currentRoomId = '';
  }

  toggleMute(): void {
    const store = useVoiceStore.getState();
    const muted = !store.isMuted;
    store.setMuted(muted);
    mediaService.setMuted(muted);
    this.send('voice:mute', { roomId: this.currentRoomId, muted });
  }

  on(event: string, handler: EventHandler): void {
    if (!this.handlers.has(event)) this.handlers.set(event, []);
    this.handlers.get(event)!.push(handler);
  }

  off(event: string, handler: EventHandler): void {
    const list = this.handlers.get(event);
    if (list) this.handlers.set(event, list.filter((h) => h !== handler));
  }

  private emit(event: string, data: unknown): void {
    this.handlers.get(event)?.forEach((h) => h(data));
  }
}

export const voiceSignaling = new VoiceSignalingService();
