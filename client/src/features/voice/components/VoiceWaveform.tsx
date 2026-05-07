interface Props {
  level: number;
  speaking: boolean;
  size?: 'sm' | 'md';
}

export function VoiceWaveform({ level, speaking, size = 'md' }: Props) {
  const bars = size === 'sm' ? 3 : 5;
  const maxH = size === 'sm' ? 12 : 20;

  return (
    <div className="flex items-center gap-0.5" style={{ height: `${maxH}px` }}>
      {Array.from({ length: bars }).map((_, i) => {
        const h = speaking
          ? Math.max(3, (level / 100) * maxH * (0.4 + Math.sin(i * 1.5 + Date.now() / 200) * 0.6))
          : 3;
        return (
          <div
            key={i}
            className="rounded-full transition-all duration-75"
            style={{
              width: size === 'sm' ? '2px' : '3px',
              height: `${h}px`,
              background: speaking
                ? 'linear-gradient(to top, #7c3aed, #06b6d4)'
                : '#374151',
            }}
          />
        );
      })}
    </div>
  );
}
