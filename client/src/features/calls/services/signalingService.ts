import { useCallStore } from '../store/callStore';
import type {
  ActiveCall,
  SignalingAnswer,
  SignalingIceCandidate,
  SignalingOffer,
} from '../types';

type EventHandler = (...args: unknown[]) => void;

class SignalingService {
  private ws: WebSocket | null = null;
  private token: string | null = null;
  private handlers: Map<string, EventHandler[]> = new Map();
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null;
  private reconnectAttempts = 0;
  private maxReconnectAttempts = 5;

  connect(wsUrl: string, token: string) {
    this.token = token;
    this.wsUrl = wsUrl;
    this.openConnection();
  }

  private wsUrl = '';

  private openConnection() {
    if (this.ws) {
      this.ws.close();
    }

    const url = `${this.wsUrl}/ws?ticket=${this.token}`;
    this.ws = new WebSocket(url);

    this.ws.onopen = () => {
      this.reconnectAttempts = 0;
      this.emit('connected', {});
    };

    this.ws.onmessage = (event) => {
      try {
        const msg = JSON.parse(event.data);
        this.handleMessage(msg);
      } catch {
        // Ignore malformed signaling messages.
      }
    };

    this.ws.onclose = () => {
      this.scheduleReconnect();
    };

    this.ws.onerror = () => {
      this.ws?.close();
    };
  }

  private scheduleReconnect() {
    if (this.reconnectAttempts >= this.maxReconnectAttempts) return;
    const delay = Math.min(1000 * 2 ** this.reconnectAttempts, 30000);
    this.reconnectAttempts++;
    this.reconnectTimer = setTimeout(() => this.openConnection(), delay);
  }

  private handleMessage(msg: { type: string; [key: string]: unknown }) {
    const store = useCallStore.getState();

    switch (msg.type) {
      case 'call:start': {
        const call = msg.call as ActiveCall;
        store.setIncomingCall({ ...call, isIncoming: true });
        this.emit('call:start', call);
        break;
      }
      case 'call:ringing':
        store.updateCallState('ringing');
        this.emit('call:ringing', msg);
        break;
      case 'call:accept':
        store.updateCallState('connecting');
        this.emit('call:accept', msg);
        break;
      case 'call:decline':
        store.updateCallState('declined');
        this.emit('call:decline', msg);
        break;
      case 'call:end':
        store.updateCallState('ended');
        this.emit('call:end', msg);
        break;
      case 'call:missed':
        store.updateCallState('missed');
        this.emit('call:missed', msg);
        break;
      case 'call:offer':
        this.emit('call:offer', msg as unknown as SignalingOffer);
        break;
      case 'call:answer':
        this.emit('call:answer', msg as unknown as SignalingAnswer);
        break;
      case 'call:ice-candidate':
        this.emit('call:ice-candidate', msg as unknown as SignalingIceCandidate);
        break;
      case 'presence:update':
        store.setUserOnline(msg.userId as string, msg.online as boolean);
        this.emit('presence:update', msg);
        break;
    }
  }

  send(type: string, payload: Record<string, unknown>) {
    if (this.ws?.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify({ type, ...payload }));
    }
  }

  on(event: string, handler: EventHandler) {
    if (!this.handlers.has(event)) this.handlers.set(event, []);
    this.handlers.get(event)!.push(handler);
  }

  off(event: string, handler: EventHandler) {
    const list = this.handlers.get(event);
    if (list) {
      this.handlers.set(event, list.filter((h) => h !== handler));
    }
  }

  emit(event: string, data: unknown) {
    this.handlers.get(event)?.forEach((h) => h(data));
  }

  sendCallStart(callId: string, to: { id: string; username: string }, from: { id: string; username: string }) {
    this.send('call:start', { callId, to, from });
  }

  sendCallAccept(callId: string) {
    this.send('call:accept', { callId });
  }

  sendCallDecline(callId: string) {
    this.send('call:decline', { callId });
  }

  sendCallEnd(callId: string) {
    this.send('call:end', { callId });
  }

  sendOffer(callId: string, sdp: RTCSessionDescriptionInit) {
    this.send('call:offer', { callId, sdp });
  }

  sendAnswer(callId: string, sdp: RTCSessionDescriptionInit) {
    this.send('call:answer', { callId, sdp });
  }

  sendIceCandidate(callId: string, candidate: RTCIceCandidateInit) {
    this.send('call:ice-candidate', { callId, candidate });
  }

  disconnect() {
    if (this.reconnectTimer) clearTimeout(this.reconnectTimer);
    this.ws?.close();
    this.ws = null;
  }
}

export const signalingService = new SignalingService();
