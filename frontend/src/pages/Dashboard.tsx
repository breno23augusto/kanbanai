import React, { useState, useCallback, useEffect, useMemo } from 'react';
import { Box, Typography, Button, IconButton, Tooltip } from '@mui/material';
import AddIcon from '@mui/icons-material/Add';
import RefreshIcon from '@mui/icons-material/Refresh';
import { KanbanBoard } from '../components/KanbanBoard';
import { CreateTaskDialog } from '../components/CreateTaskDialog';
import { TaskDetailDrawer } from '../components/TaskDetailDrawer';
import { Lamp } from '../components/Lamp';
import { useTasks } from '../hooks/useTasks';
import { useSSE } from '../hooks/useSSE';
import { Status } from '../types/task';
import { SSEEvent } from '../types/event';
import { tokens } from '../theme/theme';

const RELOAD_PREFIXES = ['task.', 'lane.', 'phase.', 'harness.error'];

function shouldReload(type: string): boolean {
  if (type === 'phase.done.reached') return true;
  return RELOAD_PREFIXES.some((p) => type.startsWith(p)) && !type.endsWith('.progress');
}

const pad2 = (n: number) => String(n).padStart(2, '0');

const TALLY_ORDER: { status: Status; label: string }[] = [
  { status: 'in_progress', label: 'running' },
  { status: 'pending', label: 'queued' },
  { status: 'completed', label: 'done' },
  { status: 'failed', label: 'failed' },
];

const mono = '"JetBrains Mono", monospace';

export const Dashboard: React.FC = () => {
  const { tasks, loading, createTask, reload } = useTasks();
  const [createOpen, setCreateOpen] = useState(false);
  const [selectedTaskId, setSelectedTaskId] = useState<string | null>(null);
  const [now, setNow] = useState(() => new Date());

  // Derive the open task from the live board list so the drawer reflects
  // state changes (status, error_message) that arrive via SSE reloads, instead
  // of the stale object captured at click time.
  const selectedTask = selectedTaskId ? tasks.find((t) => t.id === selectedTaskId) ?? null : null;

  // live console clock — the one ambient motion on the chrome.
  useEffect(() => {
    const t = setInterval(() => setNow(new Date()), 1000);
    return () => clearInterval(t);
  }, []);

  const handleSSEEvent = useCallback(
    (event: SSEEvent) => {
      if (shouldReload(event.type)) reload();
    },
    [reload],
  );
  useSSE(handleSSEEvent);

  const handleCreate = useCallback(
    async (title: string, description: string, priority: number) => {
      await createTask(title, description, priority);
    },
    [createTask],
  );

  const tally = useMemo(() => {
    const counts: Record<string, number> = {};
    for (const t of tasks) counts[t.status] = (counts[t.status] ?? 0) + 1;
    return counts;
  }, [tasks]);

  const liveCount = tally.in_progress ?? 0;

  const clock = `${pad2(now.getHours())}:${pad2(now.getMinutes())}:${pad2(now.getSeconds())}`;
  const stamp = `${now.getFullYear()}.${pad2(now.getMonth() + 1)}.${pad2(now.getDate())}`;

  return (
    <Box sx={{ display: 'flex', minHeight: '100vh', bgcolor: tokens.bg.base }}>
      {/* ── status mast ─────────────────────────────────────────────── */}
      <Box
        component="aside"
        sx={{
          width: 232,
          flexShrink: 0,
          borderRight: `1px solid ${tokens.border.hair}`,
          bgcolor: tokens.bg.panel,
          display: { xs: 'none', lg: 'flex' },
          flexDirection: 'column',
          p: 2,
          gap: 2,
        }}
      >
        {/* monogram + identity */}
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.25 }}>
          <Box
            sx={{
              width: 34,
              height: 34,
              border: `1px solid ${tokens.signal.cyan}`,
              borderRadius: 1,
              display: 'grid',
              placeItems: 'center',
              color: tokens.signal.cyan,
              fontFamily: mono,
              fontWeight: 700,
              fontSize: '1rem',
              boxShadow: `0 0 12px ${tokens.signal.cyan}22`,
            }}
          >
            K
          </Box>
          <Box>
            <Typography sx={{ fontFamily: '"Space Grotesk", sans-serif', fontWeight: 700, fontSize: '1.05rem', lineHeight: 1, color: tokens.ink.text }}>
              KanbanAI
            </Typography>
            <Typography sx={{ fontFamily: mono, fontSize: '0.6rem', letterSpacing: '0.12em', color: tokens.ink.faint, mt: 0.5, textTransform: 'uppercase' }}>
              agent pipeline
            </Typography>
          </Box>
        </Box>

        {/* agent online */}
        <Box
          sx={{
            border: `1px solid ${tokens.border.hair}`,
            borderRadius: 1,
            p: 1.25,
            bgcolor: tokens.bg.inset,
          }}
        >
          <Typography sx={{ fontFamily: mono, fontSize: '0.58rem', letterSpacing: '0.14em', color: tokens.ink.faint, textTransform: 'uppercase' }}>
            harness
          </Typography>
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mt: 0.75 }}>
            <Lamp color={tokens.signal.cyan} size={8} pulse ring />
            <Typography sx={{ fontFamily: mono, fontSize: '0.72rem', color: tokens.signal.cyan, letterSpacing: '0.06em' }}>
              AGENT&nbsp;ONLINE
            </Typography>
          </Box>
          <Typography sx={{ fontFamily: mono, fontSize: '0.6rem', color: tokens.ink.faint, mt: 0.5 }}>
            pi · ollama/deepseek-v4-flash:cloud
          </Typography>
        </Box>

        {/* tally */}
        <Box>
          <Typography sx={{ fontFamily: mono, fontSize: '0.58rem', letterSpacing: '0.14em', color: tokens.ink.faint, textTransform: 'uppercase', mb: 1 }}>
            tally
          </Typography>
          {TALLY_ORDER.map(({ status, label }) => (
            <Box key={status} sx={{ display: 'flex', alignItems: 'center', gap: 1, py: 0.4 }}>
              <Lamp color={tokens.status[status] ?? tokens.ink.faint} size={6} pulse={status === 'in_progress' && (tally[status] ?? 0) > 0} />
              <Typography sx={{ fontFamily: mono, fontSize: '0.7rem', color: tokens.ink.dim, flex: 1 }}>{label}</Typography>
              <Typography sx={{ fontFamily: mono, fontSize: '0.7rem', color: tokens.ink.text, fontWeight: 500 }}>
                {pad2(tally[status] ?? 0)}
              </Typography>
            </Box>
          ))}
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, py: 0.4, mt: 0.5, borderTop: `1px solid ${tokens.border.hair}`, pt: 1 }}>
            <Typography sx={{ fontFamily: mono, fontSize: '0.7rem', color: tokens.ink.dim, flex: 1 }}>total</Typography>
            <Typography sx={{ fontFamily: mono, fontSize: '0.7rem', color: tokens.ink.text, fontWeight: 500 }}>{pad2(tasks.length)}</Typography>
          </Box>
        </Box>

        <Box sx={{ mt: 'auto' }}>
          <Typography sx={{ fontFamily: mono, fontSize: '0.6rem', color: tokens.ink.faint, letterSpacing: '0.08em' }}>
            {stamp} · {clock}
          </Typography>
          <Typography sx={{ fontFamily: mono, fontSize: '0.56rem', color: tokens.ink.faint, mt: 0.25 }}>
            console v0.1
          </Typography>
        </Box>
      </Box>

      {/* ── main column ─────────────────────────────────────────────── */}
      <Box sx={{ flex: 1, display: 'flex', flexDirection: 'column', minWidth: 0 }}>
        {/* console header */}
        <Box
          component="header"
          sx={{
            display: 'flex',
            alignItems: 'center',
            gap: 2,
            px: 2,
            py: 1.25,
            borderBottom: `1px solid ${tokens.border.hair}`,
            bgcolor: tokens.bg.panel,
            flexShrink: 0,
          }}
        >
          {/* mobile identity */}
          <Box sx={{ display: { xs: 'flex', lg: 'none' }, alignItems: 'center', gap: 1 }}>
            <Box
              sx={{
                width: 26, height: 26, border: `1px solid ${tokens.signal.cyan}`, borderRadius: 1,
                display: 'grid', placeItems: 'center', color: tokens.signal.cyan, fontFamily: mono, fontWeight: 700, fontSize: '0.85rem',
              }}
            >
              K
            </Box>
            <Typography sx={{ fontFamily: '"Space Grotesk", sans-serif', fontWeight: 700, fontSize: '0.95rem', color: tokens.ink.text }}>
              KanbanAI
            </Typography>
          </Box>

          <Typography
            sx={{
              display: { xs: 'none', lg: 'block' },
              fontFamily: mono,
              fontSize: '0.66rem',
              letterSpacing: '0.14em',
              color: tokens.ink.faint,
              textTransform: 'uppercase',
            }}
          >
            pipeline&nbsp;console
          </Typography>

          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, ml: { xs: 'auto', lg: 2 } }}>
            <Lamp color={liveCount > 0 ? tokens.signal.cyan : tokens.ink.faint} size={7} pulse={liveCount > 0} />
            <Typography sx={{ fontFamily: mono, fontSize: '0.68rem', color: tokens.ink.dim, letterSpacing: '0.06em' }}>
              {liveCount > 0 ? `${liveCount} RUNNING` : 'IDLE'}
            </Typography>
          </Box>

          <Typography
            sx={{
              ml: 'auto',
              display: { xs: 'none', sm: 'block' },
              fontFamily: mono,
              fontSize: '0.68rem',
              color: tokens.ink.dim,
              letterSpacing: '0.08em',
            }}
          >
            {clock}
          </Typography>

          <Tooltip title="Reload board">
            <IconButton size="small" onClick={reload} disabled={loading}>
              <RefreshIcon fontSize="small" />
            </IconButton>
          </Tooltip>

          <Button
            variant="outlined"
            size="small"
            startIcon={<AddIcon />}
            onClick={() => setCreateOpen(true)}
            sx={{ borderColor: tokens.signal.cyan, color: tokens.signal.cyan, '&:hover': { borderColor: tokens.signal.cyan, bgcolor: 'rgba(79,209,197,0.08)' } }}
          >
            New&nbsp;Task
          </Button>
        </Box>

        {/* board */}
        {loading && tasks.length === 0 ? (
          <Box sx={{ flex: 1, display: 'grid', placeItems: 'center' }}>
            <Box sx={{ textAlign: 'center' }}>
              <Lamp color={tokens.signal.cyan} size={10} pulse ring />
              <Typography sx={{ fontFamily: mono, fontSize: '0.7rem', color: tokens.ink.dim, mt: 1.5, letterSpacing: '0.1em' }}>
                SYNCING&nbsp;BOARD…
              </Typography>
            </Box>
          </Box>
        ) : tasks.length === 0 ? (
          <Box sx={{ flex: 1, display: 'grid', placeItems: 'center', p: 4 }}>
            <Box sx={{ textAlign: 'center', maxWidth: 420 }}>
              <Box
                sx={{
                  width: 56, height: 56, mx: 'auto', mb: 2,
                  border: `1px solid ${tokens.border.strong}`, borderRadius: 1,
                  display: 'grid', placeItems: 'center', color: tokens.signal.cyan,
                }}
              >
                <AddIcon />
              </Box>
              <Typography sx={{ fontFamily: '"Space Grotesk", sans-serif', fontWeight: 600, fontSize: '1.1rem', color: tokens.ink.text }}>
                No tasks on the pipeline
              </Typography>
              <Typography sx={{ fontFamily: mono, fontSize: '0.72rem', color: tokens.ink.dim, mt: 1, lineHeight: 1.6 }}>
                Create a task and the agent will pick it up — moving it through
                <br />
                planning → todo → doing → validating → testing → done.
              </Typography>
              <Button
                variant="outlined"
                size="small"
                startIcon={<AddIcon />}
                onClick={() => setCreateOpen(true)}
                sx={{ mt: 2.5, borderColor: tokens.signal.cyan, color: tokens.signal.cyan }}
              >
                New&nbsp;Task
              </Button>
            </Box>
          </Box>
        ) : (
          <KanbanBoard tasks={tasks} onTaskClick={(t) => setSelectedTaskId(t.id)} />
        )}
      </Box>

      <CreateTaskDialog open={createOpen} onClose={() => setCreateOpen(false)} onCreate={handleCreate} />

      <TaskDetailDrawer task={selectedTask} open={!!selectedTask} onClose={() => setSelectedTaskId(null)} onTaskChanged={reload} />
    </Box>
  );
};