import React from 'react';
import { Task, Phase, PHASE_LABELS } from '../types/task';
import { TaskCard } from './TaskCard';
import { Lamp } from './Lamp';
import { tokens } from '../theme/theme';
import { Box, Typography } from '@mui/material';

interface KanbanLaneProps {
  phase: Phase;
  index: number;
  tasks: Task[];
  onTaskClick: (task: Task) => void;
}

const pad2 = (n: number) => String(n).padStart(2, '0');

export const KanbanLane: React.FC<KanbanLaneProps> = ({ phase, index, tasks, onTaskClick }) => {
  const lamp = tokens.phase[phase] ?? tokens.ink.faint;
  // a lane is "live" if any task in it is actively running
  const live = tasks.some((t) => t.status === 'in_progress');

  return (
    <Box
      sx={{
        display: 'flex',
        flexDirection: 'column',
        minHeight: 0,
        height: '100%',
        bgcolor: tokens.bg.panel,
        border: `1px solid ${tokens.border.hair}`,
        borderTop: `2px solid ${lamp}33`,
        borderRadius: 1,
      }}
    >
      {/* station header */}
      <Box
        sx={{
          px: 1.5,
          py: 1,
          borderBottom: `1px solid ${tokens.border.hair}`,
          display: 'flex',
          alignItems: 'center',
          gap: 1,
        }}
      >
        <Lamp color={lamp} size={7} pulse={live} ring={live} />
        <Typography
          component="span"
          sx={{ fontFamily: '"JetBrains Mono", monospace', fontSize: '0.62rem', color: tokens.ink.faint, letterSpacing: '0.1em' }}
        >
          STN&nbsp;{pad2(index + 1)}
        </Typography>
        <Typography
          sx={{
            fontFamily: '"JetBrains Mono", monospace',
            fontSize: '0.72rem',
            fontWeight: 500,
            letterSpacing: '0.08em',
            color: tokens.ink.text,
            textTransform: 'uppercase',
          }}
        >
          {PHASE_LABELS[phase]}
        </Typography>
        <Typography
          sx={{
            ml: 'auto',
            fontFamily: '"JetBrains Mono", monospace',
            fontSize: '0.68rem',
            color: tokens.ink.dim,
            bgcolor: tokens.bg.inset,
            border: `1px solid ${tokens.border.hair}`,
            borderRadius: 1,
            px: 0.75,
            lineHeight: '18px',
          }}
        >
          {tasks.length}
        </Typography>
      </Box>

      {/* card stack */}
      <Box
        sx={{
          flex: 1,
          p: 1,
          display: 'flex',
          flexDirection: 'column',
          gap: 1,
          overflowY: 'auto',
        }}
      >
        {tasks.map((task) => (
          <TaskCard key={task.id} task={task} phaseLamp={lamp} onClick={() => onTaskClick(task)} />
        ))}
        {tasks.length === 0 && (
          <Box
            sx={{
              m: 1,
              mt: 2,
              py: 3,
              textAlign: 'center',
              border: `1px dashed ${tokens.border.hair}`,
              borderRadius: 1,
            }}
          >
            <Typography
              sx={{ fontFamily: '"JetBrains Mono", monospace', fontSize: '0.68rem', color: tokens.ink.faint, letterSpacing: '0.08em' }}
            >
              empty
            </Typography>
          </Box>
        )}
      </Box>
    </Box>
  );
};