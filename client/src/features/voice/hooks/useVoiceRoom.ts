import { useCallback } from 'react';
import { useVoiceStore } from '../store/voiceStore';
import { voiceSignaling } from '../services/voiceSignaling';

export function useVoiceRoom() {
  const store = useVoiceStore();

  const joinRoom = useCallback(
    async (roomId: string, channelId: string, userId: string, username: string) => {
      await voiceSignaling.joinRoom(roomId, channelId, userId, username);
    },
    []
  );

  const leaveRoom = useCallback(() => {
    voiceSignaling.leaveRoom();
  }, []);

  const toggleMute = useCallback(() => {
    voiceSignaling.toggleMute();
  }, []);

  return {
    room: store.room,
    mode: store.mode,
    isMuted: store.isMuted,
    isSpeaking: store.isSpeaking,
    isConnecting: store.isConnecting,
    participants: store.room?.participants ?? [],
    joinRoom,
    leaveRoom,
    toggleMute,
  };
}
