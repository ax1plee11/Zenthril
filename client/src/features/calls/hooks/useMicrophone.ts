import { useState, useEffect, useRef, useCallback } from 'react';

export function useMicrophone() {
  const [level, setLevel] = useState(0);
  const [hasPermission, setHasPermission] = useState<boolean | null>(null);
  const analyserRef = useRef<AnalyserNode | null>(null);
  const audioContextRef = useRef<AudioContext | null>(null);
  const animFrameRef = useRef<number>(0);
  const streamRef = useRef<MediaStream | null>(null);

  const stopAnalysis = useCallback(() => {
    if (animFrameRef.current) {
      cancelAnimationFrame(animFrameRef.current);
      animFrameRef.current = 0;
    }
    streamRef.current?.getTracks().forEach((track) => track.stop());
    streamRef.current = null;
    analyserRef.current = null;
    audioContextRef.current?.close().catch(() => {});
    audioContextRef.current = null;
    setLevel(0);
  }, []);

  const startAnalysis = useCallback(async () => {
    try {
      stopAnalysis();
      const stream = await navigator.mediaDevices.getUserMedia({ audio: true });
      streamRef.current = stream;
      setHasPermission(true);

      const ctx = new AudioContext();
      audioContextRef.current = ctx;
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
      stopAnalysis();
    }
  }, [stopAnalysis]);

  useEffect(() => {
    return () => stopAnalysis();
  }, [stopAnalysis]);

  return { level, hasPermission, startAnalysis, stopAnalysis };
}
