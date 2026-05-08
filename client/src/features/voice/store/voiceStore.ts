import { create } from 'zustand';
import type { VoiceMode, VoiceRoom, VoiceParticipant } from '../types';

interface VoiceStore {
  room: VoiceRoom | null;
  localStream: MediaStream | null;
  isMuted: boolean;
  isSpeaking: boolean;
  mode: VoiceMode;
  isConnecting: boolean;

  setRoom: (room: VoiceRoom | null) => void;
  setLocalStream: (stream: MediaStream | null) => void;
  setMuted: (muted: boolean) => void;
  setSpeaking: (speaking: boolean) => void;
  setMode: (mode: VoiceMode) => void;
  setConnecting: (connecting: boolean) => void;
  addParticipant: (p: VoiceParticipant) => void;
  removeParticipant: (userId: string) => void;
  updateParticipant: (userId: string, update: Partial<VoiceParticipant>) => void;
  clearRoom: () => void;
}

export const useVoiceStore = create<VoiceStore>((set) => ({
  room: null,
  localStream: null,
  isMuted: false,
  isSpeaking: false,
  mode: 'p2p',
  isConnecting: false,

  setRoom: (room) => set({ room }),
  setLocalStream: (stream) => set({ localStream: stream }),
  setMuted: (muted) => set({ isMuted: muted }),
  setSpeaking: (speaking) => set({ isSpeaking: speaking }),
  setMode: (mode) => set({ mode }),
  setConnecting: (connecting) => set({ isConnecting: connecting }),

  addParticipant: (p) =>
    set((s) => {
      if (!s.room) return s;
      const exists = s.room.participants.find((x) => x.userId === p.userId);
      if (exists) return s;
      return { room: { ...s.room, participants: [...s.room.participants, p] } };
    }),

  removeParticipant: (userId) =>
    set((s) => {
      if (!s.room) return s;
      return {
        room: {
          ...s.room,
          participants: s.room.participants.filter((p) => p.userId !== userId),
        },
      };
    }),

  updateParticipant: (userId, update) =>
    set((s) => {
      if (!s.room) return s;
      return {
        room: {
          ...s.room,
          participants: s.room.participants.map((p) =>
            p.userId === userId ? { ...p, ...update } : p
          ),
        },
      };
    }),

  clearRoom: () =>
    set({
      room: null,
      localStream: null,
      isMuted: false,
      isSpeaking: false,
      mode: 'p2p',
      isConnecting: false,
    }),
}));

export function selectVoiceMode(participantCount: number): VoiceMode {
  if (participantCount <= 2) return 'p2p';
  if (participantCount <= 6) return 'mesh';
  return 'sfu';
}
