import { ParticipantTile } from './ParticipantTile';
import { VoiceControls } from './VoiceControls';
import { useVoiceRoom } from '../hooks/useVoiceRoom';
import type { VoiceParticipant } from '../types';

interface Props {
  localUserId: string;
  localUsername: string;
}

export function GroupVoiceRoom({ localUserId, localUsername }: Props) {
  const { room, mode, isMuted, isSpeaking, isConnecting, participants, toggleMute, leaveRoom } = useVoiceRoom();

  if (!room) return null;

  const localParticipant: VoiceParticipant = {
    userId: localUserId,
    username: localUsername,
    isMuted,
    isSpeaking,
    quality: 'excellent',
    audioLevel: 0,
  };

  return (
    <div className="fixed bottom-6 right-6 z-40 w-80">
      <div
        className="rounded-2xl border border-white/10 overflow-hidden"
        style={{
          background: 'linear-gradient(135deg, rgba(15,15,25,0.97), rgba(30,20,50,0.97))',
          backdropFilter: 'blur(20px)',
          boxShadow: '0 0 40px rgba(124,58,237,0.15)',
        }}
      >
        <div className="flex items-center justify-between px-4 py-3 border-b border-white/10">
          <div>
            <p className="text-white text-sm font-medium">Voice Channel</p>
            <p className="text-white/40 text-xs">{participants.length + 1} participants</p>
          </div>
          {isConnecting && (
            <div className="w-2 h-2 rounded-full bg-yellow-400 animate-pulse" />
          )}
        </div>

        <div className="p-3 grid grid-cols-3 gap-2 max-h-48 overflow-y-auto">
          <ParticipantTile participant={localParticipant} isLocal />
          {participants.map((p) => (
            <ParticipantTile key={p.userId} participant={p} />
          ))}
        </div>

        <div className="px-4 py-3 border-t border-white/10 flex justify-center">
          <VoiceControls
            isMuted={isMuted}
            mode={mode}
            onToggleMute={toggleMute}
            onLeave={leaveRoom}
          />
        </div>
      </div>
    </div>
  );
}
