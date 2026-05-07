import { useEffect, useState } from 'react';
import type { CallState } from '../types';

interface Props {
  state: CallState;
  connectedAt?: number;
}

const STATE_LABELS: Record<CallState, string> = {
  idle: '',
  calling: 'Calling...',
  ringing: 'Ringing...',
  connecting: 'Connecting...',
  connected: '',
  declined: 'Call Declined',
  missed: 'Missed Call',
  failed: 'Call Failed',
  ended: 'Call Ended',
};

export function CallStatus({ state, connectedAt }: Props) {
  const [elapsed, setElapsed] = useState(0);

  useEffect(() => {
    if (state !== 'connected' || !connectedAt) return;
    const interval = setInterval(() => {
      setElapsed(Math.floor((Date.now() - connectedAt) / 1000));
    }, 1000);
    return () => clearInterval(interval);
  }, [state, connectedAt]);

  const formatTime = (s: number) => {
    const m = Math.floor(s / 60).toString().padStart(2, '0');
    const sec = (s % 60).toString().padStart(2, '0');
    return `${m}:${sec}`;
  };

  return (
    <div className="text-sm text-white/60 text-center">
      {state === 'connected' ? formatTime(elapsed) : STATE_LABELS[state]}
    </div>
  );
}
