import React from 'react';
import { Task, Phase, PHASE_ORDER } from '../types/task';
import { KanbanLane } from './KanbanLane';
import { Box } from '@mui/material';

interface KanbanBoardProps {
  tasks: Task[];
  onTaskClick: (task: Task) => void;
}

export const KanbanBoard: React.FC<KanbanBoardProps> = ({ tasks, onTaskClick }) => {
  const byPhase = (phase: Phase): Task[] => tasks.filter((t) => t.current_phase === phase);

  return (
    <Box
      sx={{
        display: 'flex',
        gap: 1.5,
        overflowX: 'auto',
        overflowY: 'hidden',
        px: 2,
        pb: 2,
        flex: 1,
        minHeight: 0,
      }}
    >
      {PHASE_ORDER.map((phase, i) => (
        <Box
          key={phase}
          sx={{
            flex: '1 0 268px',
            maxWidth: 320,
            display: 'flex',
            flexDirection: 'column',
            minHeight: 0,
          }}
        >
          <KanbanLane
            phase={phase}
            index={i}
            tasks={byPhase(phase)}
            onTaskClick={onTaskClick}
          />
        </Box>
      ))}
    </Box>
  );
};