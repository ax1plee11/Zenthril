import { describe, it, expect, vi, beforeEach } from 'vitest';
import { signalingService } from '../services/signalingService';

describe('signalingService', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('registers and fires event handlers', () => {
    const handler = vi.fn();
    signalingService.on('test:event', handler);
    (signalingService as any).emit('test:event', { data: 1 });
    expect(handler).toHaveBeenCalledWith({ data: 1 });
    signalingService.off('test:event', handler);
  });

  it('removes event handlers with off()', () => {
    const handler = vi.fn();
    signalingService.on('test:event2', handler);
    signalingService.off('test:event2', handler);
    (signalingService as any).emit('test:event2', {});
    expect(handler).not.toHaveBeenCalled();
  });

  it('does not throw when sending without connection', () => {
    expect(() => {
      signalingService.sendCallEnd('test-call');
    }).not.toThrow();
  });

  it('does not throw when sending offer without connection', () => {
    expect(() => {
      signalingService.sendOffer('test-call', { type: 'offer', sdp: 'test' });
    }).not.toThrow();
  });

  it('does not throw when sending ice candidate without connection', () => {
    expect(() => {
      signalingService.sendIceCandidate('test-call', {
        candidate: 'candidate:...',
        sdpMid: '0',
        sdpMLineIndex: 0,
      });
    }).not.toThrow();
  });
});
