import { describe, it, expect } from 'vitest';
import type { CallState, ActiveCall, CallUser } from '../types';

describe('call types', () => {
  it('CallUser has id and username', () => {
    const user: CallUser = { id: 'abc', username: 'alice' };
    expect(user.id).toBe('abc');
    expect(user.username).toBe('alice');
  });

  it('ActiveCall has all required fields', () => {
    const call: ActiveCall = {
      callId: 'call-1',
      state: 'idle',
      caller: { id: 'u1', username: 'alice' },
      callee: { id: 'u2', username: 'bob' },
      isMuted: false,
      isIncoming: false,
    };
    expect(call.callId).toBe('call-1');
    expect(call.state).toBe('idle');
    expect(call.isMuted).toBe(false);
  });

  it('all call states are valid strings', () => {
    const states: CallState[] = [
      'idle', 'calling', 'ringing', 'connecting',
      'connected', 'declined', 'missed', 'failed', 'ended',
    ];
    expect(states).toHaveLength(9);
    states.forEach((s) => expect(typeof s).toBe('string'));
  });
});
