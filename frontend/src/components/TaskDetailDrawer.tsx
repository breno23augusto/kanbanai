import React, { useEffect, useState } from 'react';
import { Task, PhaseOutput, PHASE_ORDER, PHASE_LABELS } from '../types/task';
import { api } from '../services/api';
import {
  Drawer,
  Box,
  Typography,
  Chip,
  IconButton,
  Divider,
  Stepper,
  Step,
  StepLabel,
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

const statusColors: Record<string, string> = {
  pending: '#ff9800',
  in_progress: '#2196f3',
  completed: '#4caf50',
  failed: '#f44336',
  cancelled: '#9e9e9e',
};

const phaseAccent: Record<string, string> = {
  planning: '#90caf9',
  todo: '#ce93d8',
  doing: '#a5d6a7',
  validating: '#fff59d',
  testing: '#ffab91',
  done: '#80cbc4',
};

export const TaskDetailDrawer: React.FC<TaskDetailDrawerProps> = ({ task, open, onClose }) => {
  const [phaseOutputs, setPhaseOutputs] = useState<PhaseOutput[]>([]);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    if (!open || !task) return;
    let cancelled = false;
    setLoading(true);
    api
      .getTaskDetail(task.id)
      .then((detail) => {
        if (cancelled) return;
        setPhaseOutputs(detail?.phase_outputs ?? []);
      })
      .catch((err) => console.error('Failed to load task detail:', err))
      .finally(() => !cancelled && setLoading(false));
    return () => {
      cancelled = true;
    };
  }, [open, task]);

  if (!task) return null;

  const currentPhaseIndex = PHASE_ORDER.indexOf(task.current_phase);
  const outputsByPhase = new Map<string, PhaseOutput>();
  for (const po of phaseOutputs) outputsByPhase.set(po.phase, po);

  return (
    <Drawer anchor="right" open={open} onClose={onClose}>
      <Box sx={{ width: { xs: 320, sm: 440 }, p: 3, overflowY: 'auto' }}>
        <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 2 }}>
          <Typography variant="h6" sx={{ fontWeight: 700 }}>{task.title}</Typography>
          <IconButton onClick={onClose} size="small">
            <CloseIcon />
          </IconButton>
        </Box>

        <Chip
          label={task.status.replace('_', ' ')}
          size="small"
          sx={{ bgcolor: statusColors[task.status] || '#9e9e9e', color: '#fff', mb: 2 }}
        />

        <Typography variant="body2" color="text.secondary" sx={{ mb: 3, whiteSpace: 'pre-wrap' }}>
          {task.description || 'No description'}
        </Typography>

        <Divider sx={{ mb: 2 }} />
        <Typography variant="subtitle2" sx={{ mb: 2 }}>Phase Progress</Typography>
        <Stepper activeStep={currentPhaseIndex} alternativeLabel>
          {PHASE_ORDER.map((phase) => (
            <Step key={phase}>
              <StepLabel>{PHASE_LABELS[phase]}</StepLabel>
            </Step>
          ))}
        </Stepper>

        <Box sx={{ mt: 2, display: 'flex', flexWrap: 'wrap', gap: 1 }}>
          <Typography variant="caption" color="text.secondary" sx={{ width: '100%' }}>
            Created: {new Date(task.created_at).toLocaleString()}
          </Typography>
          <Typography variant="caption" color="text.secondary" sx={{ width: '100%' }}>
            Updated: {new Date(task.updated_at).toLocaleString()}
          </Typography>
          <Typography variant="caption" color="text.secondary">
            Version: {task.version}
          </Typography>
        </Box>

        <Divider sx={{ my: 3 }} />
        <Typography variant="subtitle2" sx={{ mb: 1 }}>Phase Outputs</Typography>
        {loading ? (
          <Box sx={{ display: 'flex', justifyContent: 'center', py: 3 }}>
            <CircularProgress size={24} />
          </Box>
        ) : phaseOutputs.length === 0 ? (
          <Typography variant="body2" color="text.secondary">
            No phase outputs recorded.
          </Typography>
        ) : (
          PHASE_ORDER.map((phase) => {
            const po = outputsByPhase.get(phase);
            if (!po) return null;
            const text = po.summary || po.output || '(empty)';
            return (
              <Accordion key={phase} disableGutters sx={{ mb: 0.5, '&:before': { display: 'none' } }}>
                <AccordionSummary expandIcon={<ExpandMoreIcon />} sx={{ minHeight: 40, '&.Mui-expanded': { minHeight: 40 } }}>
                  <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                    <Box sx={{ width: 8, height: 8, borderRadius: '50%', bgcolor: phaseAccent[phase] || '#9e9e9e' }} />
                    <Typography variant="body2" sx={{ fontWeight: 600, textTransform: 'capitalize' }}>{phase}</Typography>
                  </Box>
                </AccordionSummary>
                <AccordionDetails sx={{ pt: 0 }}>
                  <Box
                    component="pre"
                    sx={{
                      fontFamily: 'inherit',
                      fontSize: '0.8rem',
                      color: 'text.secondary',
                      whiteSpace: 'pre-wrap',
                      wordBreak: 'break-word',
                      m: 0,
                      maxHeight: 360,
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
    </Drawer>
  );
};