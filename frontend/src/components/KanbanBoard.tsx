import React from 'react';
import { Task, Phase, PHASE_ORDER, PHASE_LABELS } from '../types/task';
import { KanbanLane } from './KanbanLane';
import { Box, Typography } from '@mui/material';

interface KanbanBoardProps {
  tasks: Task[];
  onTaskClick: (task: Task) => void;
}

export const KanbanBoard: React.FC<KanbanBoardProps> = ({ tasks, onTaskClick }) => {
  const getTasksByPhase = (phase: Phase): Task[] =>
    tasks.filter((t) => t.current_phase === phase);

  return (
    <Box sx={{ display: 'flex', gap: 2, overflowX: 'auto', p: 2, minHeight: '80vh' }}>
      {PHASE_ORDER.map((phase) => (
        <Box key={phase} sx={{ minWidth: 280, flex: 1 }}>
          <Typography variant="subtitle2" sx={{ mb: 1, color: 'text.secondary', textTransform: 'uppercase', fontSize: '0.75rem', letterSpacing: 1 }}>
            {PHASE_LABELS[phase]} ({getTasksByPhase(phase).length})
          </Typography>
          <KanbanLane
            tasks={getTasksByPhase(phase)}
            onTaskClick={onTaskClick}
          />
        </Box>
      ))}
    </Box>
  );
};
