import { IncomingCallModal } from './IncomingCallModal';
import { ActiveCallOverlay } from './ActiveCallOverlay';
import { useCallState } from '../hooks/useCallState';

export function CallManager() {
  const { activeCall, incomingCall, acceptCall, declineCall, endCall, toggleMute } = useCallState();

  return (
    <>
      {incomingCall && !activeCall && (
        <IncomingCallModal
          call={incomingCall}
          onAccept={acceptCall}
          onDecline={declineCall}
        />
      )}
      {activeCall && (
        <ActiveCallOverlay
          call={activeCall}
          onEndCall={endCall}
          onToggleMute={toggleMute}
        />
      )}
    </>
  );
}
