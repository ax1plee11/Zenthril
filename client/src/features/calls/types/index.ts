export type CallState =
  | 'idle'
  | 'calling'
  | 'ringing'
  | 'connecting'
  | 'connected'
  | 'declined'
  | 'missed'
  | 'failed'
  | 'ended';

export interface CallUser {
  id: string;
  username: string;
}

export interface ActiveCall {
  callId: string;
  state: CallState;
  caller: CallUser;
  callee: CallUser;
  startedAt?: number;
  connectedAt?: number;
  endedAt?: number;
  isMuted: boolean;
  isIncoming: boolean;
}

export interface SignalingOffer {
  callId: string;
  from: CallUser;
  to: CallUser;
  sdp: RTCSessionDescriptionInit;
}

export interface SignalingAnswer {
  callId: string;
  from: CallUser;
  sdp: RTCSessionDescriptionInit;
}

export interface SignalingIceCandidate {
  callId: string;
  from: string;
  candidate: RTCIceCandidateInit;
}

export interface PresenceUpdate {
  userId: string;
  online: boolean;
}
