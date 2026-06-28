import React from 'react';
import { Task, Subtask } from '../types/task';
import { Lamp } from './Lamp';
import { tokens } from '../theme/theme';
import { Box, Typography, Tooltip } from '@mui/material';

interface TaskCardProps {
  task: Task;
  phaseLamp: string;
  onClick: () => void;
}

const STATUS_LABEL: Record<string, string> = {
  pending: 'queued',
  in_progress: 'running',
  completed: 'done',
  failed: 'failed',
  cancelled: 'halt',
  paused: 'paused',
};

const SUBTASK_DOT: Record<string, string> = {
  completed: tokens.signal.sage,
  in_progress: tokens.signal.cyan,
  pending: tokens.border.strong,
};

const SubtaskStrip: React.FC<{ subtasks: Subtask[] }> = ({ subtasks }) => {
  if (!subtasks || subtasks.length === 0) return null;
  const done = subtasks.filter((s) => s.status === 'completed').length;
  const total = subtasks.length;
  const allDone = done === total;
  const color = allDone ? tokens.signal.sage : done === 0 ? tokens.ink.faint : tokens.signal.cyan;
  return (
    <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.75, mt: 1, maxWidth: '100%' }}>
      <Typography
        sx={{
          fontFamily: '"JetBrains Mono", monospace',
          fontSize: '0.6rem',
          letterSpacing: '0.06em',
          color,
          textTransform: 'uppercase',
          flexShrink: 0,
        }}
      >
        {done}/{total}
      </Typography>
      <Box sx={{ display: 'flex', gap: 0.25, flexWrap: 'nowrap', overflow: 'hidden' }}>
        {subtasks.slice(0, 12).map((s) => (
          <Tooltip key={s.id} title={s.title} arrow>
            <Box
              sx={{
                width: 7,
                height: 7,
                borderRadius: 0.5,
                bgcolor: SUBTASK_DOT[s.status] ?? tokens.border.strong,
                flexShrink: 0,
              }}
            />
          </Tooltip>
        ))}
        {subtasks.length > 12 && (
          <Typography sx={{ fontFamily: '"JetBrains Mono", monospace', fontSize: '0.56rem', color: tokens.ink.faint, ml: 0.25 }}>
            +{subtasks.length - 12}
          </Typography>
        )}
      </Box>
    </Box>
  );
};

export const TaskCard: React.FC<TaskCardProps> = ({ task, phaseLamp, onClick }) => {
  const statusLamp = tokens.status[task.status] ?? tokens.ink.faint;
  const live = task.status === 'in_progress';

  return (
    <Box
      onClick={onClick}
      role="button"
      tabIndex={0}
      onKeyDown={(e) => (e.key === 'Enter' || e.key === ' ') && onClick()}
      className={live ? 'kai-scan' : undefined}
      sx={{
        position: 'relative',
        overflow: 'hidden',
        cursor: 'pointer',
        bgcolor: tokens.bg.panel,
        border: `1px solid ${tokens.border.hair}`,
        borderLeft: `3px solid ${phaseLamp}`,
        borderRadius: 1,
        p: 1.25,
        pl: 1.5,
        transition: 'border-color .16s, background-color .16s, transform .16s',
        '&:hover': {
          borderColor: tokens.border.strong,
          bgcolor: tokens.bg.panelAlt,
          transform: 'translateY(-1px)',
        },
        '&:focus-visible': {
          outline: `2px solid ${tokens.signal.cyan}`,
          outlineOffset: 1,
        },
      }}
    >
      <Typography
        sx={{
          fontFamily: '"Space Grotesk", sans-serif',
          fontWeight: 600,
          fontSize: '0.9rem',
          color: tokens.ink.text,
          lineHeight: 1.25,
          overflow: 'hidden',
          textOverflow: 'ellipsis',
          whiteSpace: 'nowrap',
        }}
      >
        {task.title}
      </Typography>

      {task.description && (
        <Typography
          sx={{
            mt: 0.25,
            fontSize: '0.78rem',
            color: tokens.ink.dim,
            lineHeight: 1.35,
            display: '-webkit-box',
            WebkitLineClamp: 2,
            WebkitBoxOrient: 'vertical',
            overflow: 'hidden',
          }}
        >
          {task.description}
        </Typography>
      )}

      <Box sx={{ mt: 1.25, display: 'flex', alignItems: 'center', gap: 1, flexWrap: 'wrap' }}>
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5 }}>
          <Lamp color={statusLamp} size={7} pulse={live} />
          <Typography
            sx={{
              fontFamily: '"JetBrains Mono", monospace',
              fontSize: '0.64rem',
              letterSpacing: '0.06em',
              color: statusLamp,
              textTransform: 'uppercase',
            }}
          >
            {STATUS_LABEL[task.status] ?? task.status}
          </Typography>
        </Box>

        {task.priority > 0 && (
          <Typography
            sx={{
              ml: 'auto',
              fontFamily: '"JetBrains Mono", monospace',
              fontSize: '0.62rem',
              color: tokens.ink.dim,
              border: `1px solid ${tokens.border.hair}`,
              borderRadius: 1,
              px: 0.5,
              lineHeight: '16px',
            }}
          >
            P{task.priority}
          </Typography>
        )}
      </Box>

      <SubtaskStrip subtasks={task.subtasks ?? []} />
    </Box>
  );
};