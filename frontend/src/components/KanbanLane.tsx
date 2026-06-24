import React from 'react';
import { Task } from '../types/task';
import { TaskCard } from './TaskCard';
import { Box } from '@mui/material';

interface KanbanLaneProps {
  tasks: Task[];
  onTaskClick: (task: Task) => void;
}

export const KanbanLane: React.FC<KanbanLaneProps> = ({ tasks, onTaskClick }) => {
  return (
    <Box
      sx={{
        bgcolor: 'background.paper',
        borderRadius: 2,
        p: 1.5,
        minHeight: 400,
        display: 'flex',
        flexDirection: 'column',
        gap: 1.5,
      }}
    >
      {tasks.map((task) => (
        <TaskCard key={task.id} task={task} onClick={() => onTaskClick(task)} />
      ))}
      {tasks.length === 0 && (
        <Box sx={{ color: 'text.disabled', textAlign: 'center', py: 4, fontSize: '0.875rem' }}>
          No tasks
        </Box>
      )}
    </Box>
  );
};