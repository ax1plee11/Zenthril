import { Mic, MicOff, PhoneOff } from 'lucide-react';
import type { VoiceMode } from '../types';

interface Props {
  isMuted: boolean;
  mode: VoiceMode;
  onToggleMute: () => void;
  onLeave: () => void;
}

const MODE_LABELS: Record<VoiceMode, string> = {
  p2p: 'P2P',
  mesh: 'Mesh',
  sfu: 'SFU',
};

const MODE_COLORS: Record<VoiceMode, string> = {
  p2p: '#22c55e',
  mesh: '#eab308',
  sfu: '#7c3aed',
};

export function VoiceControls({ isMuted, mode, onToggleMute, onLeave }: Props) {
  return (
    <div className="flex items-center gap-3">
      <div
        className="px-2 py-0.5 rounded text-xs font-mono"
        style={{ background: MODE_COLORS[mode] + '22', color: MODE_COLORS[mode], border: `1px solid ${MODE_COLORS[mode]}44` }}
      >
        {MODE_LABELS[mode]}
      </div>

      <button
        onClick={onToggleMute}
        className={`w-9 h-9 rounded-full flex items-center justify-center transition-all ${
          isMuted
            ? 'bg-red-500/20 border border-red-500/50 text-red-400'
            : 'bg-white/10 border border-white/20 text-white hover:bg-white/20'
        }`}
      >
        {isMuted ? <MicOff size={16} /> : <Mic size={16} />}
      </button>

      <button
        onClick={onLeave}
        className="w-9 h-9 rounded-full bg-red-500 hover:bg-red-600 flex items-center justify-center text-white transition-all"
      >
        <PhoneOff size={16} />
      </button>
    </div>
  );
}
