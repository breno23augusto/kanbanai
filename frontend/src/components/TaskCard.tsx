import React from 'react';
import { Task } from '../types/task';
import { Card, CardContent, Typography, Chip, Box } from '@mui/material';

interface TaskCardProps {
  task: Task;
  onClick: () => void;
}

const statusColors: Record<string, string> = {
  pending: '#ff9800',
  in_progress: '#2196f3',
  completed: '#4caf50',
  failed: '#f44336',
  cancelled: '#9e9e9e',
};

export const TaskCard: React.FC<TaskCardProps> = ({ task, onClick }) => {
  return (
    <Card
      sx={{
        cursor: 'pointer',
        transition: 'box-shadow 0.2s',
        '&:hover': { boxShadow: 4 },
      }}
      onClick={onClick}
    >
      <CardContent sx={{ p: 1.5, '&:last-child': { pb: 1.5 } }}>
        <Typography variant="body2" fontWeight={600} noWrap>
          {task.title}
        </Typography>
        <Typography variant="caption" color="text.secondary" sx={{ display: 'block', mt: 0.5 }} noWrap>
          {task.description}
        </Typography>
        <Box sx={{ display: 'flex', gap: 0.5, mt: 1, alignItems: 'center' }}>
          <Chip
            label={task.status.replace('_', ' ')}
            size="small"
            sx={{
              bgcolor: statusColors[task.status] || '#9e9e9e',
              color: '#fff',
              fontSize: '0.7rem',
              height: 20,
            }}
          />
          {task.priority > 0 && (
            <Chip
              label={`P${task.priority}`}
              size="small"
              variant="outlined"
              sx={{ fontSize: '0.7rem', height: 20 }}
            />
          )}
        </Box>
      </CardContent>
    </Card>
  );
};
