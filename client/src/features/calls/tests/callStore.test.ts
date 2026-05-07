import { describe, it, expect, beforeEach } from 'vitest';
import { useCallStore } from '../store/callStore';

const mockCall = {
  callId: 'test-call-1',
  state: 'calling' as const,
  caller: { id: 'user-1', username: 'alice' },
  callee: { id: 'user-2', username: 'bob' },
  startedAt: Date.now(),
  isMuted: false,
  isIncoming: false,
};

describe('callStore', () => {
  beforeEach(() => {
    useCallStore.getState().clearCall();
  });

  it('sets active call', () => {
    useCallStore.getState().setActiveCall(mockCall);
    expect(useCallStore.getState().activeCall).toEqual(mockCall);
  });

  it('sets incoming call', () => {
    const incoming = { ...mockCall, isIncoming: true };
    useCallStore.getState().setIncomingCall(incoming);
    expect(useCallStore.getState().incomingCall).toEqual(incoming);
  });

  it('updates call state', () => {
    useCallStore.getState().setActiveCall(mockCall);
    useCallStore.getState().updateCallState('connected');
    expect(useCallStore.getState().activeCall?.state).toBe('connected');
  });

  it('sets muted', () => {
    useCallStore.getState().setActiveCall(mockCall);
    useCallStore.getState().setMuted(true);
    expect(useCallStore.getState().activeCall?.isMuted).toBe(true);
  });

  it('clears call', () => {
    useCallStore.getState().setActiveCall(mockCall);
    useCallStore.getState().setIncomingCall({ ...mockCall, isIncoming: true });
    useCallStore.getState().clearCall();
    expect(useCallStore.getState().activeCall).toBeNull();
    expect(useCallStore.getState().incomingCall).toBeNull();
  });

  it('tracks online users', () => {
    useCallStore.getState().setUserOnline('user-1', true);
    expect(useCallStore.getState().onlineUsers.has('user-1')).toBe(true);

    useCallStore.getState().setUserOnline('user-1', false);
    expect(useCallStore.getState().onlineUsers.has('user-1')).toBe(false);
  });

  it('does not update state when no active call', () => {
    useCallStore.getState().updateCallState('connected');
    expect(useCallStore.getState().activeCall).toBeNull();
  });

  it('does not set muted when no active call', () => {
    useCallStore.getState().setMuted(true);
    expect(useCallStore.getState().activeCall).toBeNull();
  });
});
