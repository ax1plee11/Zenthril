export type VoiceMode = 'p2p' | 'mesh' | 'sfu';

export type ConnectionQuality = 'excellent' | 'good' | 'poor' | 'disconnected';

export interface VoiceParticipant {
  userId: string;
  username: string;
  isMuted: boolean;
  isSpeaking: boolean;
  quality: ConnectionQuality;
  audioLevel: number;
}

export interface VoiceRoom {
  roomId: string;
  channelId: string;
  mode: VoiceMode;
  participants: VoiceParticipant[];
  createdAt: number;
}

export interface PeerConnection {
  userId: string;
  pc: RTCPeerConnection;
  stream: MediaStream | null;
  quality: ConnectionQuality;
}

export interface ModeSwitch {
  from: VoiceMode;
  to: VoiceMode;
  reason: string;
  participantCount: number;
}

export interface SignalingMessage {
  type: string;
  roomId?: string;
  userId?: string;
  targetUserId?: string;
  sdp?: RTCSessionDescriptionInit;
  candidate?: RTCIceCandidateInit;
  mode?: VoiceMode;
  participants?: VoiceParticipant[];
}
