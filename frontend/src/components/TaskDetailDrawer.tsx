import React, { useEffect, useState } from 'react';
import { Task, PhaseOutput, Subtask, PHASE_ORDER, PHASE_LABELS } from '../types/task';
import { api } from '../services/api';
import { Lamp } from './Lamp';
import { MarkdownView, SubtaskSummary, extractSubtasks } from './MarkdownView';
import { tokens } from '../theme/theme';
import {
  Drawer,
  Box,
  Typography,
  IconButton,
  CircularProgress,
  Accordion,
  AccordionSummary,
  AccordionDetails,
  Button,
  TextField,
  Tooltip,
} from '@mui/material';
import ExpandMoreIcon from '@mui/icons-material/ExpandMore';
import CloseIcon from '@mui/icons-material/Close';
import PauseIcon from '@mui/icons-material/Pause';
import PlayArrowIcon from '@mui/icons-material/PlayArrow';
import EditIcon from '@mui/icons-material/Edit';
import RestartAltIcon from '@mui/icons-material/RestartAlt';
import SaveIcon from '@mui/icons-material/Save';
import DeleteOutlineIcon from '@mui/icons-material/DeleteOutline';

interface TaskDetailDrawerProps {
  task: Task | null;
  open: boolean;
  onClose: () => void;
  /** Called after any mutation so the board list refreshes. */
  onTaskChanged?: () => void;
}

const mono = '"JetBrains Mono", monospace';
const sans = '"Space Grotesk", sans-serif';

const STATUS_LABEL: Record<string, string> = {
  pending: 'QUEUED',
  in_progress: 'RUNNING',
  completed: 'COMPLETED',
  failed: 'FAILED',
  cancelled: 'HALTED',
  paused: 'PAUSED',
};

const pad2 = (n: number) => String(n).padStart(2, '0');

const STATION_COUNT = PHASE_ORDER.length; // 6
// Each station is 1/6 of the row. A gap connector between station (i-1) and i
// spans center-to-center: left = (i - 0.5)/6, width = 1/6 (in %).
const gapLeft = (i: number) => ((i - 0.5) / STATION_COUNT) * 100;
const GAP_WIDTH = (1 / STATION_COUNT) * 100;

/**
 * A custom pipeline transit strip — the drawer's signature element.
 * Replaces the generic MUI Stepper with station ticks + signal lamps.
 *
 * Layout: a dedicated connector layer sits BEHIND the stations (zIndex 0,
 * pointer-events none), drawn as per-gap hairlines aligned to the node band
 * center. Each station node is an opaque panel-colored disc on top (zIndex 1)
 * so the line is cleanly masked under the nodes instead of cutting through
 * them — which was producing the overlapping artifacts.
 */
const TransitStrip: React.FC<{ task: Task }> = ({ task }) => {
  const currentIdx = PHASE_ORDER.indexOf(task.current_phase);
  const finished = task.status === 'completed' || task.status === 'cancelled';
  const NODE_BAND = 18; // px — node row height; connectors align to its center (9)

  return (
    <Box sx={{ position: 'relative', display: 'flex', alignItems: 'flex-start' }}>
      {/* connector layer (behind) */}
      <Box
        sx={{
          position: 'absolute',
          top: NODE_BAND / 2, // node center
          left: 0,
          right: 0,
          height: 0,
          zIndex: 0,
          pointerEvents: 'none',
        }}
      >
        {PHASE_ORDER.map((_, i) => {
          if (i === 0) return null;
          const passed = i <= currentIdx; // gap reaching station i is traversed
          return (
            <Box
              key={`gap-${i}`}
              sx={{
                position: 'absolute',
                top: -0.5,
                left: `${gapLeft(i)}%`,
                width: `${GAP_WIDTH}%`,
                height: 1,
                bgcolor: passed ? `${tokens.signal.sage}66` : tokens.border.hair,
              }}
            />
          );
        })}
      </Box>

      {/* stations (in front) */}
      {PHASE_ORDER.map((phase, i) => {
        const isCurrent = i === currentIdx;
        const passed = finished ? i <= currentIdx : i < currentIdx;
        const future = i > currentIdx;
        const lit = isCurrent || passed;
        const nodeColor =
          isCurrent && !finished ? tokens.signal.cyan : passed ? tokens.signal.sage : tokens.ink.faint;
        const accentColor = passed ? tokens.signal.sage : isCurrent ? tokens.phase[phase] : tokens.ink.faint;
        return (
          <Box
            key={phase}
            sx={{
              flex: 1,
              position: 'relative',
              zIndex: 1,
              display: 'flex',
              flexDirection: 'column',
              alignItems: 'center',
            }}
          >
            {/* node band — fixed height so the connector aligns to its center.
                The opaque disc masks the line passing underneath. */}
            <Box
              sx={{
                height: NODE_BAND,
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
              }}
            >
              <Box
                sx={{
                  width: 14,
                  height: 14,
                  borderRadius: '50%',
                  bgcolor: tokens.bg.panel, // masks the connector under the node
                  display: 'grid',
                  placeItems: 'center',
                }}
              >
                <Lamp
                  color={nodeColor}
                  size={isCurrent && !finished ? 9 : 7}
                  pulse={isCurrent && !finished}
                  ring={isCurrent && !finished}
                />
              </Box>
            </Box>

            <Typography
              sx={{
                fontFamily: mono,
                fontSize: '0.56rem',
                color: tokens.ink.faint,
                mt: 1,
                letterSpacing: '0.1em',
              }}
            >
              {pad2(i + 1)}
            </Typography>
            <Typography
              sx={{
                fontFamily: mono,
                fontSize: '0.58rem',
                letterSpacing: '0.06em',
                textTransform: 'uppercase',
                color: lit ? tokens.ink.text : tokens.ink.faint,
                mt: 0.25,
                textAlign: 'center',
                lineHeight: 1.2,
              }}
            >
              {PHASE_LABELS[phase]}
            </Typography>
            <Box
              sx={{
                height: 2,
                width: 14,
                mt: 0.5,
                bgcolor: accentColor,
                opacity: lit ? 1 : 0.3,
              }}
            />
            {/* spacer keeps future columns the same height as passed ones */}
            {future && <Box sx={{ height: 14 }} />}
          </Box>
        );
      })}
    </Box>
  );
};

export const TaskDetailDrawer: React.FC<TaskDetailDrawerProps> = ({ task, open, onClose, onTaskChanged }) => {
  const [phaseOutputs, setPhaseOutputs] = useState<PhaseOutput[]>([]);
  const [subtasks, setSubtasks] = useState<Subtask[]>([]);
  const [loading, setLoading] = useState(false);
  const [expanded, setExpanded] = useState<string | false>(task?.current_phase ?? false);

  // local, mutable copy of the task so the drawer reflects mutations
  // (pause/resume/edit) immediately without waiting for the board to reload.
  const [localTask, setLocalTask] = useState<Task | null>(task);
  const [busy, setBusy] = useState(false);
  const [editing, setEditing] = useState(false);
  const [editTitle, setEditTitle] = useState('');
  const [editDesc, setEditDesc] = useState('');
  const [editPriority, setEditPriority] = useState(0);
  const [editError, setEditError] = useState<string | null>(null);

  // sync local task + load detail whenever a different task is opened.
  useEffect(() => {
    setLocalTask(task);
    setEditing(false);
    setEditError(null);
    if (!open || !task) return;
    let cancelled = false;
    setLoading(true);
    api
      .getTaskDetail(task.id)
      .then((detail) => {
        if (cancelled) return;
        setPhaseOutputs(detail?.phase_outputs ?? []);
        setSubtasks(detail?.subtasks ?? []);
        if (detail?.task) setLocalTask(detail.task);
        const have = (detail?.phase_outputs ?? []).map((p) => p.phase);
        setExpanded(have.includes(task.current_phase) ? task.current_phase : (have[have.length - 1] ?? false));
      })
      .catch((err) => console.error('Failed to load task detail:', err))
      .finally(() => !cancelled && setLoading(false));
    return () => {
      cancelled = true;
    };
  }, [open, task]);

  if (!task) return null;

  const t = localTask ?? task;
  const statusLamp = tokens.status[t.status] ?? tokens.ink.faint;
  const live = t.status === 'in_progress';
  const paused = t.status === 'paused';
  const outputsByPhase = new Map<string, PhaseOutput>();
  for (const po of phaseOutputs) outputsByPhase.set(po.phase, po);

  const refresh = async () => {
    const detail = await api.getTaskDetail(t.id);
    if (detail?.task) setLocalTask(detail.task);
    setPhaseOutputs(detail?.phase_outputs ?? []);
    setSubtasks(detail?.subtasks ?? []);
    onTaskChanged?.();
  };

  const runAction = async (label: string, fn: () => Promise<unknown>) => {
    setBusy(true);
    try {
      await fn();
      await refresh();
    } catch (err) {
      console.error(`${label} failed:`, err);
      alert(`${label} failed: ${(err as Error).message}`);
    } finally {
      setBusy(false);
    }
  };

  const handlePause = () => runAction('Pause', () => api.pauseTask(t.id));
  const handleResume = () => runAction('Resume', () => api.resumeTask(t.id));
  const handleRetry = () => runAction('Retry', () => api.retryTask(t.id));

  const handleDelete = async () => {
    const confirmed = window.confirm(
      `Delete task "${t.title}"?\nThis permanently removes the task, its phase outputs, and subtasks. This cannot be undone.`,
    );
    if (!confirmed) return;
    setBusy(true);
    try {
      await api.deleteTask(t.id);
      onTaskChanged?.();
      onClose();
    } catch (err) {
      console.error('Delete failed:', err);
      alert(`Delete failed: ${(err as Error).message}`);
    } finally {
      setBusy(false);
    }
  };

  const startEdit = () => {
    setEditTitle(t.title);
    setEditDesc(t.description);
    setEditPriority(t.priority);
    setEditError(null);
    setEditing(true);
  };

  const cancelEdit = () => {
    setEditing(false);
    setEditError(null);
  };

  const saveEdit = async () => {
    setBusy(true);
    setEditError(null);
    try {
      await api.updateTask(t.id, {
        title: editTitle.trim(),
        description: editDesc,
        priority: editPriority,
        version: t.version,
      });
      setEditing(false);
      await refresh();
    } catch (err) {
      const msg = (err as Error).message ?? '';
      if (msg.includes('409')) {
        setEditError('The task was modified elsewhere. Reloading — please try again.');
        await refresh();
      } else {
        setEditError(`Save failed: ${msg}`);
      }
    } finally {
      setBusy(false);
    }
  };

  return (
    <Drawer
      anchor="right"
      open={open}
      onClose={onClose}
      PaperProps={{ sx: { width: { xs: 320, sm: 460 } } }}
    >
      <Box sx={{ height: '100%', display: 'flex', flexDirection: 'column' }}>
        {/* header */}
        <Box
          sx={{
            p: 2.5,
            borderBottom: `1px solid ${tokens.border.hair}`,
            display: 'flex',
            flexDirection: 'column',
            gap: 1.5,
          }}
        >
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
            <Lamp color={statusLamp} size={9} pulse={live} ring={live} />
            <Typography sx={{ fontFamily: mono, fontSize: '0.66rem', letterSpacing: '0.14em', color: statusLamp }}>
              {STATUS_LABEL[t.status] ?? t.status.toUpperCase()}
            </Typography>
            <IconButton onClick={onClose} size="small" sx={{ ml: 'auto' }}>
              <CloseIcon fontSize="small" />
            </IconButton>
          </Box>

          <Typography sx={{ fontFamily: sans, fontWeight: 700, fontSize: '1.15rem', lineHeight: 1.2, color: tokens.ink.text, pr: 4 }}>
            {t.title}
          </Typography>

          {/* transit strip */}
          <Box sx={{ pt: 1 }}>
            <Typography sx={{ fontFamily: mono, fontSize: '0.56rem', letterSpacing: '0.14em', color: tokens.ink.faint, textTransform: 'uppercase', mb: 1.5 }}>
              transit
            </Typography>
            <TransitStrip task={t} />
          </Box>

          {/* action bar — pause / resume / edit / retry */}
          <Box sx={{ display: 'flex', gap: 1, flexWrap: 'wrap', pt: 0.5 }}>
            {live && (
              <Tooltip title="Pause the running harness for this task">
                <Button
                  size="small"
                  variant="outlined"
                  startIcon={<PauseIcon />}
                  disabled={busy}
                  onClick={handlePause}
                  sx={actionSx(tokens.signal.amber)}
                >
                  Pause
                </Button>
              </Tooltip>
            )}
            {paused && (
              <Tooltip title="Resume the current phase">
                <Button
                  size="small"
                  variant="outlined"
                  startIcon={<PlayArrowIcon />}
                  disabled={busy}
                  onClick={handleResume}
                  sx={actionSx(tokens.signal.cyan)}
                >
                  Resume
                </Button>
              </Tooltip>
            )}
            {(t.status === 'failed' || paused) && (
              <Tooltip title="Re-dispatch the current phase from scratch">
                <Button
                  size="small"
                  variant="outlined"
                  startIcon={<RestartAltIcon />}
                  disabled={busy}
                  onClick={handleRetry}
                  sx={actionSx(tokens.signal.sage)}
                >
                  Retry
                </Button>
              </Tooltip>
            )}
            {!editing && t.status !== 'completed' && t.status !== 'cancelled' && (
              <Tooltip title="Edit title, description and priority">
                <Button
                  size="small"
                  variant="outlined"
                  startIcon={<EditIcon />}
                  disabled={busy}
                  onClick={startEdit}
                  sx={actionSx(tokens.ink.dim)}
                >
                  Edit
                </Button>
              </Tooltip>
            )}
            {!editing && (
              <Tooltip title="Permanently delete this task, its outputs and subtasks">
                <Button
                  size="small"
                  variant="outlined"
                  startIcon={<DeleteOutlineIcon />}
                  disabled={busy}
                  onClick={handleDelete}
                  sx={actionSx(tokens.signal.coral)}
                >
                  Delete
                </Button>
              </Tooltip>
            )}
            {busy && <CircularProgress size={18} sx={{ color: tokens.signal.cyan, alignSelf: 'center' }} />}
          </Box>
        </Box>

        {/* scroll body */}
        <Box sx={{ flex: 1, overflowY: 'auto', p: 2.5 }}>
          {/* description / edit form */}
          <Box sx={{ mb: 3 }}>
            <Typography sx={{ fontFamily: mono, fontSize: '0.56rem', letterSpacing: '0.14em', color: tokens.ink.faint, textTransform: 'uppercase', mb: 1 }}>
              brief
            </Typography>
            {editing ? (
              <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1.5 }}>
                <TextField
                  label="Title"
                  value={editTitle}
                  onChange={(e) => setEditTitle(e.target.value)}
                  size="small"
                  fullWidth
                />
                <TextField
                  label="Description"
                  value={editDesc}
                  onChange={(e) => setEditDesc(e.target.value)}
                  size="small"
                  fullWidth
                  multiline
                  minRows={3}
                />
                <TextField
                  label="Priority"
                  type="number"
                  value={editPriority}
                  onChange={(e) => setEditPriority(Number(e.target.value))}
                  size="small"
                  sx={{ maxWidth: 140 }}
                />
                {editError && (
                  <Typography sx={{ fontFamily: mono, fontSize: '0.66rem', color: tokens.signal.coral }}>
                    {editError}
                  </Typography>
                )}
                <Box sx={{ display: 'flex', gap: 1 }}>
                  <Button
                    size="small"
                    variant="contained"
                    startIcon={<SaveIcon />}
                    disabled={busy || editTitle.trim().length === 0}
                    onClick={saveEdit}
                    sx={{ bgcolor: tokens.signal.sage, '&:hover': { bgcolor: tokens.signal.sage } }}
                  >
                    Save
                  </Button>
                  <Button size="small" variant="outlined" disabled={busy} onClick={cancelEdit}>
                    Cancel
                  </Button>
                </Box>
              </Box>
            ) : (
              <Typography sx={{ fontFamily: sans, fontSize: '0.86rem', color: tokens.ink.dim, whiteSpace: 'pre-wrap', lineHeight: 1.6 }}>
                {t.description || 'No description provided.'}
              </Typography>
            )}
          </Box>

          {/* metadata */}
          <Box
            sx={{
              display: 'grid',
              gridTemplateColumns: '1fr 1fr',
              gap: 1,
              mb: 3,
              p: 1.25,
              border: `1px solid ${tokens.border.hair}`,
              borderRadius: 1,
              bgcolor: tokens.bg.inset,
            }}
          >
            <Meta label="created" value={new Date(t.created_at).toLocaleString()} />
            <Meta label="updated" value={new Date(t.updated_at).toLocaleString()} />
            <Meta label="version" value={`v${t.version}`} />
            <Meta label="phase" value={t.current_phase} />
          </Box>

          {/* failure reason — surfaced when the task entered the failed state.
              The backend captures the harness stdout/stderr that caused the
              final retry to exhaust, so the operator can see *why* it failed. */}
          {t.status === 'failed' && (
            <Box
              sx={{
                mb: 3,
                border: `1px solid ${tokens.signal.coral}55`,
                borderRadius: 1,
                bgcolor: `${tokens.signal.coral}0a`,
                overflow: 'hidden',
              }}
            >
              <Box
                sx={{
                  display: 'flex',
                  alignItems: 'center',
                  gap: 1,
                  px: 1.25,
                  py: 0.75,
                  borderBottom: `1px solid ${tokens.signal.coral}33`,
                  bgcolor: `${tokens.signal.coral}12`,
                }}
              >
                <Lamp color={tokens.signal.coral} size={7} />
                <Typography sx={{ fontFamily: mono, fontSize: '0.6rem', letterSpacing: '0.14em', color: tokens.signal.coral, textTransform: 'uppercase' }}>
                  failure · {PHASE_LABELS[t.current_phase] ?? t.current_phase}
                </Typography>
              </Box>
              {t.error_message ? (
                <Box
                  component="pre"
                  sx={{
                    m: 0,
                    p: 1.25,
                    maxHeight: 240,
                    overflowY: 'auto',
                    fontFamily: mono,
                    fontSize: '0.7rem',
                    lineHeight: 1.5,
                    color: tokens.ink.dim,
                    whiteSpace: 'pre-wrap',
                    wordBreak: 'break-word',
                  }}
                >
                  {t.error_message}
                </Box>
              ) : (
                <Typography sx={{ p: 1.25, fontFamily: mono, fontSize: '0.7rem', color: tokens.ink.faint }}>
                  No failure reason was captured for this task.
                </Typography>
              )}
            </Box>
          )}

          {/* subtasks — the tracked checklist created in planning, with live
              per-subtask status reported by the harness via MCP as it works. */}
          <Typography sx={{ fontFamily: mono, fontSize: '0.56rem', letterSpacing: '0.14em', color: tokens.ink.faint, textTransform: 'uppercase', mb: 1 }}>
            subtasks · {(subtasks ?? []).length} {subtasks?.length === 1 ? 'item' : 'items'}
          </Typography>

          {subtasks && subtasks.length > 0 ? (
            <Box
              sx={{
                mb: 3,
                border: `1px solid ${tokens.border.hair}`,
                borderRadius: 1,
                overflow: 'hidden',
              }}
            >
              {subtasks.map((st, idx) => {
                const done = st.status === 'completed';
                const active = st.status === 'in_progress';
                const dotColor = done ? tokens.signal.sage : active ? tokens.signal.cyan : tokens.border.strong;
                return (
                  <Box
                    key={st.id}
                    sx={{
                      display: 'flex',
                      alignItems: 'flex-start',
                      gap: 1,
                      px: 1.25,
                      py: 0.85,
                      borderBottom: idx < subtasks.length - 1 ? `1px solid ${tokens.border.hair}` : 'none',
                      bgcolor: active ? `${tokens.signal.cyan}0a` : 'transparent',
                    }}
                  >
                    <Box
                      sx={{
                        mt: '2px',
                        width: 9,
                        height: 9,
                        borderRadius: 0.5,
                        border: `1px solid ${dotColor}`,
                        bgcolor: done ? dotColor : 'transparent',
                        flexShrink: 0,
                        ...(active ? { boxShadow: `0 0 6px ${dotColor}88` } : {}),
                      }}
                    />
                    <Box sx={{ minWidth: 0, flex: 1 }}>
                      <Typography
                        sx={{
                          fontFamily: sans,
                          fontSize: '0.8rem',
                          color: done ? tokens.ink.dim : tokens.ink.text,
                          textDecoration: done ? 'line-through' : 'none',
                          lineHeight: 1.35,
                          wordBreak: 'break-word',
                        }}
                      >
                        {st.title}
                      </Typography>
                      <Typography
                        sx={{
                          fontFamily: mono,
                          fontSize: '0.56rem',
                          letterSpacing: '0.08em',
                          textTransform: 'uppercase',
                          color: dotColor,
                          mt: 0.25,
                        }}
                      >
                        {st.status.replace('_', ' ')}
                      </Typography>
                    </Box>
                  </Box>
                );
              })}
            </Box>
          ) : (
            <Box sx={{ py: 2, mb: 3, textAlign: 'center', border: `1px dashed ${tokens.border.hair}`, borderRadius: 1 }}>
              <Typography sx={{ fontFamily: mono, fontSize: '0.68rem', color: tokens.ink.faint }}>
                no subtasks created yet
              </Typography>
            </Box>
          )}

          {/* phase outputs */}
          <Typography sx={{ fontFamily: mono, fontSize: '0.56rem', letterSpacing: '0.14em', color: tokens.ink.faint, textTransform: 'uppercase', mb: 1 }}>
            agent output · {phaseOutputs.length} {phaseOutputs.length === 1 ? 'phase' : 'phases'}
          </Typography>

          {loading ? (
            <Box sx={{ display: 'flex', justifyContent: 'center', py: 4 }}>
              <CircularProgress size={22} sx={{ color: tokens.signal.cyan }} />
            </Box>
          ) : phaseOutputs.length === 0 ? (
            <Box sx={{ py: 3, textAlign: 'center', border: `1px dashed ${tokens.border.hair}`, borderRadius: 1 }}>
              <Typography sx={{ fontFamily: mono, fontSize: '0.68rem', color: tokens.ink.faint }}>
                no output recorded yet
              </Typography>
            </Box>
          ) : (
            PHASE_ORDER.map((phase, i) => {
              const po = outputsByPhase.get(phase);
              if (!po) return null;
              const text = po.summary || po.output || '(empty)';
              return (
                <Accordion
                  key={phase}
                  expanded={expanded === phase}
                  onChange={(_, isExp) => setExpanded(isExp ? phase : false)}
                  disableGutters
                >
                  <AccordionSummary expandIcon={<ExpandMoreIcon sx={{ fontSize: 18, color: tokens.ink.dim }} />}>
                    <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, width: '100%' }}>
                      <Typography sx={{ fontFamily: mono, fontSize: '0.58rem', color: tokens.ink.faint, letterSpacing: '0.08em' }}>
                        {pad2(i + 1)}
                      </Typography>
                      <Lamp color={tokens.phase[phase]} size={7} />
                      <Typography sx={{ fontFamily: mono, fontSize: '0.72rem', letterSpacing: '0.06em', textTransform: 'uppercase', color: tokens.ink.text, fontWeight: 500 }}>
                        {PHASE_LABELS[phase]}
                      </Typography>
                      <Typography sx={{ ml: 'auto', fontFamily: mono, fontSize: '0.58rem', color: tokens.ink.faint }}>
                        {text.length}c
                      </Typography>
                    </Box>
                  </AccordionSummary>
                  <AccordionDetails sx={{ pt: 0, pb: 1.5 }}>
                    {/* Subtask status summary — only renders when the phase
                        output contains GFM task-list items (`- [ ]`/`- [x]`). */}
                    {extractSubtasks(text).length > 0 && <SubtaskSummary source={text} />}
                    {/* Rendered markdown — headings, lists, tables, fenced
                        code, and the task-list checkboxes that surface
                        subtask status visually. */}
                    <MarkdownView source={text} />
                  </AccordionDetails>
                </Accordion>
              );
            })
          )}
        </Box>
      </Box>
    </Drawer>
  );
};

const actionSx = (color: string) => ({
  borderColor: `${color}88`,
  color,
  textTransform: 'none',
  fontFamily: '"JetBrains Mono", monospace',
  fontSize: '0.66rem',
  letterSpacing: '0.04em',
  '&:hover': { borderColor: color, bgcolor: `${color}14` },
});

const Meta: React.FC<{ label: string; value: string }> = ({ label, value }) => (
  <Box>
    <Typography sx={{ fontFamily: mono, fontSize: '0.56rem', letterSpacing: '0.12em', color: tokens.ink.faint, textTransform: 'uppercase' }}>
      {label}
    </Typography>
    <Typography sx={{ fontFamily: mono, fontSize: '0.72rem', color: tokens.ink.text, mt: 0.25, textTransform: 'capitalize' }}>
      {value}
    </Typography>
  </Box>
);