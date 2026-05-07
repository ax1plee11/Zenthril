import { describe, it, expect } from 'vitest';
import type { VoiceMode, ConnectionQuality, VoiceParticipant, VoiceRoom } from '../types';

describe('voice types', () => {
  it('VoiceMode values are correct', () => {
    const modes: VoiceMode[] = ['p2p', 'mesh', 'sfu'];
    expect(modes).toHaveLength(3);
  });

  it('ConnectionQuality values are correct', () => {
    const qualities: ConnectionQuality[] = ['excellent', 'good', 'poor', 'disconnected'];
    expect(qualities).toHaveLength(4);
  });

  it('VoiceParticipant has all fields', () => {
    const p: VoiceParticipant = {
      userId: 'u1',
      username: 'alice',
      isMuted: false,
      isSpeaking: true,
      quality: 'excellent',
      audioLevel: 80,
    };
    expect(p.userId).toBe('u1');
    expect(p.audioLevel).toBe(80);
  });

  it('VoiceRoom has all fields', () => {
    const room: VoiceRoom = {
      roomId: 'r1',
      channelId: 'c1',
      mode: 'mesh',
      participants: [],
      createdAt: 1000,
    };
    expect(room.mode).toBe('mesh');
    expect(room.participants).toHaveLength(0);
  });
});
