import { create } from 'zustand';
import type { ActiveCall, CallState } from '../types';

interface CallStore {
  activeCall: ActiveCall | null;
  incomingCall: ActiveCall | null;
  onlineUsers: Set<string>;

  setActiveCall: (call: ActiveCall | null) => void;
  setIncomingCall: (call: ActiveCall | null) => void;
  updateCallState: (state: CallState) => void;
  setMuted: (muted: boolean) => void;
  setUserOnline: (userId: string, online: boolean) => void;
  clearCall: () => void;
}

export const useCallStore = create<CallStore>((set) => ({
  activeCall: null,
  incomingCall: null,
  onlineUsers: new Set(),

  setActiveCall: (call) => set({ activeCall: call }),

  setIncomingCall: (call) => set({ incomingCall: call }),

  updateCallState: (state) =>
    set((s) => ({
      activeCall: s.activeCall ? { ...s.activeCall, state } : null,
    })),

  setMuted: (muted) =>
    set((s) => ({
      activeCall: s.activeCall ? { ...s.activeCall, isMuted: muted } : null,
    })),

  setUserOnline: (userId, online) =>
    set((s) => {
      const next = new Set(s.onlineUsers);
      if (online) next.add(userId);
      else next.delete(userId);
      return { onlineUsers: next };
    }),

  clearCall: () => set({ activeCall: null, incomingCall: null }),
}));
