
interface Props {
  level: number;
  active?: boolean;
}

export function AudioVisualizer({ level, active = true }: Props) {
  const bars = 5;

  return (
    <div className="flex items-end gap-0.5 h-6">
      {Array.from({ length: bars }).map((_, i) => {
        const height = active
          ? Math.max(4, (level / 100) * 24 * (0.5 + Math.sin(i * 1.2) * 0.5))
          : 4;
        return (
          <div
            key={i}
            className="w-1 rounded-full transition-all duration-75"
            style={{
              height: `${height}px`,
              background: active
                ? `linear-gradient(to top, #7c3aed, #06b6d4)`
                : '#374151',
            }}
          />
        );
      })}
    </div>
  );
}
