import { useEffect, useRef } from 'react';
import { useCallStore } from '../store/callStore';
import { signalingService } from '../services/signalingService';
import { webrtcService } from '../services/webrtcService';
import type {
  SignalingAnswer,
  SignalingIceCandidate,
  SignalingOffer,
} from '../types';

export function useCallState() {
  const store = useCallStore();
  const callTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  useEffect(() => {
    const handleOffer = async (data: unknown) => {
      const offer = data as SignalingOffer;
      await webrtcService.handleOffer(offer.callId, offer.sdp);
    };

    const handleAnswer = async (data: unknown) => {
      const answer = data as SignalingAnswer;
      await webrtcService.handleAnswer(answer.sdp);
    };

    const handleIce = async (data: unknown) => {
      const ice = data as SignalingIceCandidate;
      await webrtcService.handleIceCandidate(ice.candidate);
    };

    const handleCallEnd = () => {
      webrtcService.cleanup();
      if (callTimeoutRef.current) clearTimeout(callTimeoutRef.current);
      setTimeout(() => store.clearCall(), 2000);
    };

    const handleDecline = () => {
      webrtcService.cleanup();
      if (callTimeoutRef.current) clearTimeout(callTimeoutRef.current);
      setTimeout(() => store.clearCall(), 2000);
    };

    const handleAccept = async () => {
      const call = store.activeCall;
      if (!call) return;
      if (!call.isIncoming) {
        await webrtcService.startCall(call.callId);
      }
    };

    signalingService.on('call:offer', handleOffer);
    signalingService.on('call:answer', handleAnswer);
    signalingService.on('call:ice-candidate', handleIce);
    signalingService.on('call:end', handleCallEnd);
    signalingService.on('call:decline', handleDecline);
    signalingService.on('call:accept', handleAccept);

    return () => {
      signalingService.off('call:offer', handleOffer);
      signalingService.off('call:answer', handleAnswer);
      signalingService.off('call:ice-candidate', handleIce);
      signalingService.off('call:end', handleCallEnd);
      signalingService.off('call:decline', handleDecline);
      signalingService.off('call:accept', handleAccept);
    };
  }, [store]);

  const startCall = (to: { id: string; username: string }, from: { id: string; username: string }) => {
    const callId = `call_${Date.now()}_${Math.random().toString(36).slice(2)}`;

    store.setActiveCall({
      callId,
      state: 'calling',
      caller: from,
      callee: to,
      startedAt: Date.now(),
      isMuted: false,
      isIncoming: false,
    });

    signalingService.sendCallStart(callId, to, from);

    callTimeoutRef.current = setTimeout(() => {
      store.updateCallState('missed');
      signalingService.sendCallEnd(callId);
      webrtcService.cleanup();
      setTimeout(() => store.clearCall(), 2000);
    }, 30000);
  };

  const acceptCall = () => {
    const call = store.incomingCall;
    if (!call) return;

    if (callTimeoutRef.current) clearTimeout(callTimeoutRef.current);

    store.setActiveCall({ ...call, state: 'connecting' });
    store.setIncomingCall(null);
    signalingService.sendCallAccept(call.callId);
  };

  const declineCall = () => {
    const call = store.incomingCall;
    if (!call) return;

    signalingService.sendCallDecline(call.callId);
    store.setIncomingCall(null);
  };

  const endCall = () => {
    const call = store.activeCall;
    if (!call) return;

    signalingService.sendCallEnd(call.callId);
    webrtcService.cleanup();
    store.updateCallState('ended');
    if (callTimeoutRef.current) clearTimeout(callTimeoutRef.current);
    setTimeout(() => store.clearCall(), 2000);
  };

  const toggleMute = () => {
    const muted = !store.activeCall?.isMuted;
    store.setMuted(muted);
    webrtcService.setMuted(muted);
  };

  return {
    activeCall: store.activeCall,
    incomingCall: store.incomingCall,
    onlineUsers: store.onlineUsers,
    startCall,
    acceptCall,
    declineCall,
    endCall,
    toggleMute,
  };
}
