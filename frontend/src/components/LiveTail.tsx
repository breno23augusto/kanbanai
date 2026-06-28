import { useEffect, useRef, useState } from 'react';
import { Box, Typography } from '@mui/material';
import { tokens } from '../theme/theme';
import { Task, PHASE_LABELS } from '../types/task';

const mono = '"JetBrains Mono", ui-monospace, monospace';

const SSE_BASE = import.meta.env.VITE_API_BASE_URL || '';

interface LiveTailProps {
  task: Task;
}

/**
 * LiveTail opens an SSE connection to /tasks/:id/live and renders the running
 * harness's stdout/stderr in real time — the operator can watch the agent
 * think/work instead of staring at a static card. Reconnects whenever the
 * phase changes (each phase run opens a fresh server-side stream).
 */
export function LiveTail({ task }: LiveTailProps) {
  const [text, setText] = useState('');
  const [ended, setEnded] = useState(false);
  const [elapsed, setElapsed] = useState(0);
  const boxRef = useRef<HTMLDivElement>(null);
  // key on id+phase so a phase transition (incl. reopen of the same phase)
  // reopens the stream with a clean buffer.
  const streamKey = `${task.id}:${task.current_phase}:${task.updated_at}`;
  const startedAtRef = useRef<number>(Date.now());

  useEffect(() => {
    setText('');
    setEnded(false);
    startedAtRef.current = Date.now();
    const source = new EventSource(`${SSE_BASE}/api/v1/tasks/${task.id}/live`);
    source.onmessage = (ev) => {
      try {
        const chunk = JSON.parse(ev.data);
        if (chunk.end) {
          setEnded(true);
          source.close();
          return;
        }
        if (chunk.text) setText((prev) => (prev + chunk.text).slice(-65536));
      } catch {
        /* ignore parse errors */
      }
    };
    source.onerror = () => {
      // The server closes the stream when the harness exits; a bare error here
      // is expected. Avoid the browser's automatic noisy reconnect by closing.
      setEnded(true);
      source.close();
    };
    return () => source.close();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [streamKey]);

  // elapsed timer while live
  useEffect(() => {
    if (ended) return;
    const t = setInterval(() => setElapsed(Math.floor((Date.now() - startedAtRef.current) / 1000)), 1000);
    return () => clearInterval(t);
  }, [ended]);

  // autoscroll to bottom on new text
  useEffect(() => {
    const el = boxRef.current;
    if (el) el.scrollTop = el.scrollHeight;
  }, [text]);

  const mm = String(Math.floor(elapsed / 60)).padStart(2, '0');
  const ss = String(elapsed % 60).padStart(2, '0');

  return (
    <Box sx={{ mb: 3 }}>
      <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mb: 1 }}>
        <Box
          component="span"
          sx={{
            width: 7,
            height: 7,
            borderRadius: '50%',
            bgcolor: ended ? tokens.ink.faint : tokens.signal.sage,
            boxShadow: ended ? 'none' : `0 0 6px ${tokens.signal.sage}`,
            animation: ended ? 'none' : 'pulse 1.4s ease-in-out infinite',
            '@keyframes pulse': {
              '0%,100%': { opacity: 1 },
              '50%': { opacity: 0.35 },
            },
          }}
        />
        <Typography sx={{ fontFamily: mono, fontSize: '0.56rem', letterSpacing: '0.14em', textTransform: 'uppercase', color: tokens.ink.faint }}>
          {ended ? 'harness idle' : 'live · harness output'}
        </Typography>
        <Typography sx={{ fontFamily: mono, fontSize: '0.56rem', color: tokens.ink.faint, ml: 'auto' }}>
          {PHASE_LABELS[task.current_phase] ?? task.current_phase} · {mm}:{ss}
        </Typography>
      </Box>
      <Box
        ref={boxRef}
        sx={{
          height: 260,
          overflowY: 'auto',
          p: 1.25,
          bgcolor: '#0b0f14',
          border: `1px solid ${tokens.border.hair}`,
          borderRadius: 1,
          fontFamily: mono,
          fontSize: '0.7rem',
          lineHeight: 1.45,
          color: '#9bb3b8',
          whiteSpace: 'pre-wrap',
          wordBreak: 'break-word',
        }}
      >
        {text || (
          <Box sx={{ color: tokens.ink.faint, fontStyle: 'italic' }}>
            {ended ? 'no live output — harness not running for this task.' : 'waiting for harness output\u2026'}
          </Box>
        )}
      </Box>
    </Box>
  );
}