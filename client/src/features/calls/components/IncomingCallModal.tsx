import { useEffect } from 'react';
import { Phone, PhoneOff } from 'lucide-react';
import type { ActiveCall } from '../types';

interface Props {
  call: ActiveCall;
  onAccept: () => void;
  onDecline: () => void;
}

export function IncomingCallModal({ call, onAccept, onDecline }: Props) {
  useEffect(() => {
    const audio = new Audio('/sounds/ringtone.mp3');
    audio.loop = true;
    audio.play().catch(() => {});
    return () => {
      audio.pause();
      audio.currentTime = 0;
    };
  }, []);

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      <div className="absolute inset-0 bg-black/60 backdrop-blur-sm" />
      <div
        className="relative z-10 w-80 rounded-2xl border border-white/10 p-6 text-center"
        style={{
          background: 'linear-gradient(135deg, rgba(124,58,237,0.15), rgba(6,182,212,0.1))',
          backdropFilter: 'blur(20px)',
          boxShadow: '0 0 40px rgba(124,58,237,0.3)',
        }}
      >
        <div className="relative mx-auto mb-4 w-20 h-20">
          <div className="absolute inset-0 rounded-full bg-violet-500/20 animate-ping" />
          <div className="relative w-20 h-20 rounded-full bg-gradient-to-br from-violet-500 to-cyan-500 flex items-center justify-center text-2xl font-bold text-white">
            {call.caller.username[0].toUpperCase()}
          </div>
        </div>

        <p className="text-white/60 text-sm mb-1">Incoming call</p>
        <p className="text-white text-xl font-semibold mb-6">{call.caller.username}</p>

        <div className="flex items-center justify-center gap-8">
          <button
            onClick={onDecline}
            className="w-14 h-14 rounded-full bg-red-500/20 border border-red-500/50 flex items-center justify-center text-red-400 hover:bg-red-500/30 transition-all"
          >
            <PhoneOff size={22} />
          </button>
          <button
            onClick={onAccept}
            className="w-14 h-14 rounded-full bg-green-500 flex items-center justify-center text-white hover:bg-green-600 transition-all shadow-lg shadow-green-500/30"
          >
            <Phone size={22} />
          </button>
        </div>
      </div>
    </div>
  );
}
