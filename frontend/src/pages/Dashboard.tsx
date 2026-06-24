import React, { useState, useCallback } from 'react';
import { Box, AppBar, Toolbar, Typography, Button } from '@mui/material';
import AddIcon from '@mui/icons-material/Add';
import { KanbanBoard } from '../components/KanbanBoard';
import { CreateTaskDialog } from '../components/CreateTaskDialog';
import { TaskDetailDrawer } from '../components/TaskDetailDrawer';
import { useTasks } from '../hooks/useTasks';
import { useSSE } from '../hooks/useSSE';
import { Task } from '../types/task';
import { SSEEvent } from '../types/event';

export const Dashboard: React.FC = () => {
  const { tasks, loading, createTask, reload } = useTasks();
  const [createOpen, setCreateOpen] = useState(false);
  const [selectedTask, setSelectedTask] = useState<Task | null>(null);

  const handleSSEEvent = useCallback((event: SSEEvent) => {
    if (['task.created', 'task.updated', 'task.deleted', 'task.status_changed'].includes(event.type)) {
      reload();
    }
  }, [reload]);

  useSSE(handleSSEEvent);

  return (
    <Box sx={{ minHeight: '100vh', bgcolor: 'background.default' }}>
      <AppBar position="static" elevation={0}>
        <Toolbar>
          <Typography variant="h6" sx={{ flexGrow: 1, fontWeight: 700 }}>
            KanbanAI
          </Typography>
          <Button
            color="inherit"
            startIcon={<AddIcon />}
            onClick={() => setCreateOpen(true)}
          >
            New Task
          </Button>
        </Toolbar>
      </AppBar>

      {loading ? (
        <Box sx={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '60vh' }}>
          <Typography color="text.secondary">Loading...</Typography>
        </Box>
      ) : (
        <KanbanBoard tasks={tasks} onTaskClick={setSelectedTask} />
      )}

      <CreateTaskDialog
        open={createOpen}
        onClose={() => setCreateOpen(false)}
        onCreate={createTask}
      />

      <TaskDetailDrawer
        task={selectedTask}
        open={!!selectedTask}
        onClose={() => setSelectedTask(null)}
      />
    </Box>
  );
};
