import React from 'react';
import { Task, PHASE_ORDER, PHASE_LABELS } from '../types/task';
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
} from '@mui/material';
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

export const TaskDetailDrawer: React.FC<TaskDetailDrawerProps> = ({ task, open, onClose }) => {
  if (!task) return null;

  const currentPhaseIndex = PHASE_ORDER.indexOf(task.current_phase);

  return (
    <Drawer anchor="right" open={open} onClose={onClose}>
      <Box sx={{ width: 400, p: 3 }}>
        <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 2 }}>
          <Typography variant="h6">{task.title}</Typography>
          <IconButton onClick={onClose} size="small">
            <CloseIcon />
          </IconButton>
        </Box>

        <Chip
          label={task.status.replace('_', ' ')}
          size="small"
          sx={{ bgcolor: statusColors[task.status], color: '#fff', mb: 2 }}
        />

        <Typography variant="body2" color="text.secondary" sx={{ mb: 3 }}>
          {task.description}
        </Typography>

        <Divider sx={{ mb: 2 }} />
        <Typography variant="subtitle2" sx={{ mb: 2 }}>
          Progress
        </Typography>
        <Stepper activeStep={currentPhaseIndex} alternativeLabel>
          {PHASE_ORDER.map((phase) => (
            <Step key={phase}>
              <StepLabel>{PHASE_LABELS[phase]}</StepLabel>
            </Step>
          ))}
        </Stepper>

        <Box sx={{ mt: 3 }}>
          <Typography variant="caption" color="text.secondary" display="block">
            Created: {new Date(task.created_at).toLocaleString()}
          </Typography>
          <Typography variant="caption" color="text.secondary" display="block">
            Updated: {new Date(task.updated_at).toLocaleString()}
          </Typography>
          <Typography variant="caption" color="text.secondary" display="block">
            Version: {task.version}
          </Typography>
        </Box>
      </Box>
    </Drawer>
  );
};
