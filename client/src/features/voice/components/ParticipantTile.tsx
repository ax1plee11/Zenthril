import { VoiceWaveform } from './VoiceWaveform';
import { ConnectionIndicator } from './ConnectionIndicator';
import type { VoiceParticipant } from '../types';
import { MicOff } from 'lucide-react';

interface Props {
  participant: VoiceParticipant;
  isLocal?: boolean;
}

export function ParticipantTile({ participant, isLocal }: Props) {
  return (
    <div
      className={`relative flex flex-col items-center gap-2 p-3 rounded-xl border transition-all ${
        participant.isSpeaking
          ? 'border-violet-500/60 shadow-lg shadow-violet-500/20'
          : 'border-white/10'
      }`}
      style={{
        background: participant.isSpeaking
          ? 'rgba(124,58,237,0.1)'
          : 'rgba(255,255,255,0.04)',
      }}
    >
      <div className="relative">
        <div
          className={`w-12 h-12 rounded-full flex items-center justify-center text-lg font-bold text-white ${
            participant.isSpeaking ? 'ring-2 ring-violet-500' : ''
          }`}
          style={{ background: 'linear-gradient(135deg, #7c3aed, #06b6d4)' }}
        >
          {participant.username.charAt(0).toUpperCase()}
        </div>
        {participant.isMuted && (
          <div className="absolute -bottom-1 -right-1 w-5 h-5 rounded-full bg-red-500 flex items-center justify-center">
            <MicOff size={10} className="text-white" />
          </div>
        )}
      </div>

      <p className="text-white text-xs font-medium truncate max-w-[80px]">
        {participant.username}{isLocal ? ' (you)' : ''}
      </p>

      <div className="flex items-center gap-2">
        <VoiceWaveform level={participant.audioLevel} speaking={participant.isSpeaking} size="sm" />
        <ConnectionIndicator quality={participant.quality} />
      </div>
    </div>
  );
}
