import React, { useEffect, useState } from 'react';
import { Task, PhaseOutput, PHASE_ORDER, PHASE_LABELS } from '../types/task';
import { api } from '../services/api';
import { Lamp } from './Lamp';
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
} from '@mui/material';
import ExpandMoreIcon from '@mui/icons-material/ExpandMore';
import CloseIcon from '@mui/icons-material/Close';

interface TaskDetailDrawerProps {
  task: Task | null;
  open: boolean;
  onClose: () => void;
}

const mono = '"JetBrains Mono", monospace';
const sans = '"Space Grotesk", sans-serif';

const STATUS_LABEL: Record<string, string> = {
  pending: 'QUEUED',
  in_progress: 'RUNNING',
  completed: 'COMPLETED',
  failed: 'FAILED',
  cancelled: 'HALTED',
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

export const TaskDetailDrawer: React.FC<TaskDetailDrawerProps> = ({ task, open, onClose }) => {
  const [phaseOutputs, setPhaseOutputs] = useState<PhaseOutput[]>([]);
  const [loading, setLoading] = useState(false);
  const [expanded, setExpanded] = useState<string | false>(task?.current_phase ?? false);

  useEffect(() => {
    if (!open || !task) return;
    let cancelled = false;
    setLoading(true);
    api
      .getTaskDetail(task.id)
      .then((detail) => {
        if (cancelled) return;
        setPhaseOutputs(detail?.phase_outputs ?? []);
        // default-open the current (or last) phase that has output
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

  const statusLamp = tokens.status[task.status] ?? tokens.ink.faint;
  const live = task.status === 'in_progress';
  const outputsByPhase = new Map<string, PhaseOutput>();
  for (const po of phaseOutputs) outputsByPhase.set(po.phase, po);

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
              {STATUS_LABEL[task.status] ?? task.status.toUpperCase()}
            </Typography>
            <IconButton onClick={onClose} size="small" sx={{ ml: 'auto' }}>
              <CloseIcon fontSize="small" />
            </IconButton>
          </Box>

          <Typography sx={{ fontFamily: sans, fontWeight: 700, fontSize: '1.15rem', lineHeight: 1.2, color: tokens.ink.text, pr: 4 }}>
            {task.title}
          </Typography>

          {/* transit strip */}
          <Box sx={{ pt: 1 }}>
            <Typography sx={{ fontFamily: mono, fontSize: '0.56rem', letterSpacing: '0.14em', color: tokens.ink.faint, textTransform: 'uppercase', mb: 1.5 }}>
              transit
            </Typography>
            <TransitStrip task={task} />
          </Box>
        </Box>

        {/* scroll body */}
        <Box sx={{ flex: 1, overflowY: 'auto', p: 2.5 }}>
          {/* description */}
          <Box sx={{ mb: 3 }}>
            <Typography sx={{ fontFamily: mono, fontSize: '0.56rem', letterSpacing: '0.14em', color: tokens.ink.faint, textTransform: 'uppercase', mb: 1 }}>
              brief
            </Typography>
            <Typography sx={{ fontFamily: sans, fontSize: '0.86rem', color: tokens.ink.dim, whiteSpace: 'pre-wrap', lineHeight: 1.6 }}>
              {task.description || 'No description provided.'}
            </Typography>
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
            <Meta label="created" value={new Date(task.created_at).toLocaleString()} />
            <Meta label="updated" value={new Date(task.updated_at).toLocaleString()} />
            <Meta label="version" value={`v${task.version}`} />
            <Meta label="phase" value={task.current_phase} />
          </Box>

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
                    <Box
                      component="pre"
                      sx={{
                        fontFamily: mono,
                        fontSize: '0.74rem',
                        lineHeight: 1.6,
                        color: tokens.ink.dim,
                        whiteSpace: 'pre-wrap',
                        wordBreak: 'break-word',
                        m: 0,
                        p: 1.25,
                        bgcolor: tokens.bg.inset,
                        border: `1px solid ${tokens.border.hair}`,
                        borderRadius: 1,
                        maxHeight: 420,
                        overflowY: 'auto',
                      }}
                    >
                      {text}
                    </Box>
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