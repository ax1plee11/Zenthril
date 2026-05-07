import type { ConnectionQuality } from '../types';

interface Props {
  quality: ConnectionQuality;
}

const COLORS: Record<ConnectionQuality, string> = {
  excellent: '#22c55e',
  good: '#eab308',
  poor: '#ef4444',
  disconnected: '#6b7280',
};

const LABELS: Record<ConnectionQuality, string> = {
  excellent: 'Excellent',
  good: 'Good',
  poor: 'Poor',
  disconnected: 'Disconnected',
};

export function ConnectionIndicator({ quality }: Props) {
  const color = COLORS[quality];
  return (
    <div className="flex items-center gap-1" title={LABELS[quality]}>
      {[1, 2, 3].map((bar) => (
        <div
          key={bar}
          className="w-1 rounded-sm transition-all"
          style={{
            height: `${bar * 4}px`,
            background: bar <= (quality === 'excellent' ? 3 : quality === 'good' ? 2 : quality === 'poor' ? 1 : 0)
              ? color
              : '#374151',
          }}
        />
      ))}
    </div>
  );
}
