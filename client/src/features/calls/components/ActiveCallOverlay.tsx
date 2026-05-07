import { useEffect } from 'react';
import { CallControls } from './CallControls';
import { CallStatus } from './CallStatus';
import { AudioVisualizer } from './AudioVisualizer';
import { useMicrophone } from '../hooks/useMicrophone';
import type { ActiveCall } from '../types';

interface Props {
  call: ActiveCall;
  onEndCall: () => void;
  onToggleMute: () => void;
}

export function ActiveCallOverlay({ call, onEndCall, onToggleMute }: Props) {
  const { level, startAnalysis, stopAnalysis } = useMicrophone();

  useEffect(() => {
    if (call.state === 'connected' && !call.isMuted) {
      startAnalysis();
    } else {
      stopAnalysis();
    }
    return () => stopAnalysis();
  }, [call.state, call.isMuted]);

  const isEnding = ['declined', 'missed', 'failed', 'ended'].includes(call.state);
  const peer = call.isIncoming ? call.caller : call.callee;

  return (
    <div className="fixed bottom-6 right-6 z-40">
      <div
        className="w-72 rounded-2xl border border-white/10 p-5"
        style={{
          background: 'linear-gradient(135deg, rgba(15,15,25,0.95), rgba(30,20,50,0.95))',
          backdropFilter: 'blur(20px)',
          boxShadow: '0 0 30px rgba(124,58,237,0.2)',
        }}
      >
        <div className="flex items-center gap-3 mb-4">
          <div className="w-10 h-10 rounded-full bg-gradient-to-br from-violet-500 to-cyan-500 flex items-center justify-center text-white font-bold">
            {peer.username[0].toUpperCase()}
          </div>
          <div className="flex-1">
            <p className="text-white font-medium text-sm">{peer.username}</p>
            <CallStatus state={call.state} connectedAt={call.connectedAt} />
          </div>
          {call.state === 'connected' && (
            <AudioVisualizer level={level} active={!call.isMuted} />
          )}
        </div>

        {!isEnding && (
          <CallControls
            isMuted={call.isMuted}
            onToggleMute={onToggleMute}
            onEndCall={onEndCall}
          />
        )}

        {isEnding && (
          <div className="text-center text-white/50 text-sm py-2">
            <CallStatus state={call.state} />
          </div>
        )}
      </div>
    </div>
  );
}
