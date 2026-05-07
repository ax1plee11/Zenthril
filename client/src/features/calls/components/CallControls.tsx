import { Mic, MicOff, PhoneOff } from 'lucide-react';

interface Props {
  isMuted: boolean;
  onToggleMute: () => void;
  onEndCall: () => void;
}

export function CallControls({ isMuted, onToggleMute, onEndCall }: Props) {
  return (
    <div className="flex items-center gap-4 justify-center">
      <button
        onClick={onToggleMute}
        className={`w-12 h-12 rounded-full flex items-center justify-center transition-all ${
          isMuted
            ? 'bg-red-500/20 border border-red-500/50 text-red-400'
            : 'bg-white/10 border border-white/20 text-white hover:bg-white/20'
        }`}
      >
        {isMuted ? <MicOff size={20} /> : <Mic size={20} />}
      </button>

      <button
        onClick={onEndCall}
        className="w-14 h-14 rounded-full bg-red-500 hover:bg-red-600 flex items-center justify-center transition-all shadow-lg shadow-red-500/30"
      >
        <PhoneOff size={22} className="text-white" />
      </button>
    </div>
  );
}
