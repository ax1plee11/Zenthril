import { useState, useEffect, useRef } from 'react';

export function useMicrophone() {
  const [level, setLevel] = useState(0);
  const [hasPermission, setHasPermission] = useState<boolean | null>(null);
  const analyserRef = useRef<AnalyserNode | null>(null);
  const animFrameRef = useRef<number>(0);
  const streamRef = useRef<MediaStream | null>(null);

  const startAnalysis = async () => {
    try {
      const stream = await navigator.mediaDevices.getUserMedia({ audio: true });
      streamRef.current = stream;
      setHasPermission(true);

      const ctx = new AudioContext();
      const source = ctx.createMediaStreamSource(stream);
      const analyser = ctx.createAnalyser();
      analyser.fftSize = 256;
      source.connect(analyser);
      analyserRef.current = analyser;

      const data = new Uint8Array(analyser.frequencyBinCount);

      const tick = () => {
        analyser.getByteFrequencyData(data);
        const avg = data.reduce((a, b) => a + b, 0) / data.length;
        setLevel(Math.min(100, avg * 2));
        animFrameRef.current = requestAnimationFrame(tick);
      };

      tick();
    } catch {
      setHasPermission(false);
    }
  };

  const stopAnalysis = () => {
    cancelAnimationFrame(animFrameRef.current);
    streamRef.current?.getTracks().forEach((t) => t.stop());
    streamRef.current = null;
    analyserRef.current = null;
    setLevel(0);
  };

  useEffect(() => {
    return () => stopAnalysis();
  }, []);

  return { level, hasPermission, startAnalysis, stopAnalysis };
}
