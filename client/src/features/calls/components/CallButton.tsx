import { Phone } from 'lucide-react';
import { useCallStore } from '../store/callStore';

interface Props {
  userId: string;
  username: string;
  currentUserId: string;
  currentUsername: string;
  onCall: (to: { id: string; username: string }, from: { id: string; username: string }) => void;
}

export function CallButton({ userId, username, currentUserId, currentUsername, onCall }: Props) {
  const { onlineUsers, activeCall } = useCallStore();
  const isOnline = onlineUsers.has(userId);
  const isBusy = !!activeCall;

  if (!isOnline || isBusy) return null;

  return (
    <button
      onClick={() =>
        onCall(
          { id: userId, username },
          { id: currentUserId, username: currentUsername }
        )
      }
      className="w-8 h-8 rounded-full bg-green-500/20 border border-green-500/40 flex items-center justify-center text-green-400 hover:bg-green-500/30 transition-all"
      title={`Call ${username}`}
    >
      <Phone size={14} />
    </button>
  );
}
