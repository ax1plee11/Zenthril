import { describe, it, expect, beforeEach } from 'vitest';
import { useVoiceStore, selectVoiceMode } from '../store/voiceStore';

const mockRoom = {
  roomId: 'room-1',
  channelId: 'ch-1',
  mode: 'p2p' as const,
  participants: [],
  createdAt: Date.now(),
};

const mockParticipant = {
  userId: 'user-1',
  username: 'alice',
  isMuted: false,
  isSpeaking: false,
  quality: 'good' as const,
  audioLevel: 0,
};

describe('voiceStore', () => {
  beforeEach(() => {
    useVoiceStore.getState().clearRoom();
  });

  it('sets room', () => {
    useVoiceStore.getState().setRoom(mockRoom);
    expect(useVoiceStore.getState().room).toEqual(mockRoom);
  });

  it('clears room', () => {
    useVoiceStore.getState().setRoom(mockRoom);
    useVoiceStore.getState().clearRoom();
    expect(useVoiceStore.getState().room).toBeNull();
  });

  it('adds participant', () => {
    useVoiceStore.getState().setRoom(mockRoom);
    useVoiceStore.getState().addParticipant(mockParticipant);
    expect(useVoiceStore.getState().room?.participants).toHaveLength(1);
  });

  it('does not add duplicate participant', () => {
    useVoiceStore.getState().setRoom(mockRoom);
    useVoiceStore.getState().addParticipant(mockParticipant);
    useVoiceStore.getState().addParticipant(mockParticipant);
    expect(useVoiceStore.getState().room?.participants).toHaveLength(1);
  });

  it('removes participant', () => {
    useVoiceStore.getState().setRoom(mockRoom);
    useVoiceStore.getState().addParticipant(mockParticipant);
    useVoiceStore.getState().removeParticipant('user-1');
    expect(useVoiceStore.getState().room?.participants).toHaveLength(0);
  });

  it('updates participant', () => {
    useVoiceStore.getState().setRoom(mockRoom);
    useVoiceStore.getState().addParticipant(mockParticipant);
    useVoiceStore.getState().updateParticipant('user-1', { isSpeaking: true, audioLevel: 75 });
    const p = useVoiceStore.getState().room?.participants[0];
    expect(p?.isSpeaking).toBe(true);
    expect(p?.audioLevel).toBe(75);
  });

  it('sets muted', () => {
    useVoiceStore.getState().setMuted(true);
    expect(useVoiceStore.getState().isMuted).toBe(true);
  });

  it('sets mode', () => {
    useVoiceStore.getState().setMode('sfu');
    expect(useVoiceStore.getState().mode).toBe('sfu');
  });
});

describe('selectVoiceMode', () => {
  it('returns p2p for 1 participant', () => {
    expect(selectVoiceMode(1)).toBe('p2p');
  });

  it('returns p2p for 2 participants', () => {
    expect(selectVoiceMode(2)).toBe('p2p');
  });

  it('returns mesh for 3 participants', () => {
    expect(selectVoiceMode(3)).toBe('mesh');
  });

  it('returns mesh for 6 participants', () => {
    expect(selectVoiceMode(6)).toBe('mesh');
  });

  it('returns sfu for 7 participants', () => {
    expect(selectVoiceMode(7)).toBe('sfu');
  });

  it('returns sfu for 100 participants', () => {
    expect(selectVoiceMode(100)).toBe('sfu');
  });
});
