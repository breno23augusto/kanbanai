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

// Reload the board on any event that mutates task state/phase. The backend
// publishes phase.* (started/completed/failed/retry), lane.* and task.* events;
// note that task.status_changed is currently never emitted, so we key off the
// phase/lane/harness events to stay in sync. Progress/output events are
// intentionally ignored to avoid refetching on every heartbeat.
const RELOAD_PREFIXES = ['task.', 'lane.', 'phase.', 'harness.error'];

function shouldReload(type: string): boolean {
  if (type === 'phase.done.reached') return true;
  return RELOAD_PREFIXES.some((p) => type.startsWith(p)) && !type.endsWith('.progress');
}

export const Dashboard: React.FC = () => {
  const { tasks, loading, createTask, reload } = useTasks();
  const [createOpen, setCreateOpen] = useState(false);
  const [selectedTask, setSelectedTask] = useState<Task | null>(null);

  const handleSSEEvent = useCallback((event: SSEEvent) => {
    if (shouldReload(event.type)) {
      reload();
    }
  }, [reload]);

  useSSE(handleSSEEvent);

  const handleCreate = useCallback(
    async (title: string, description: string, priority: number) => {
      await createTask(title, description, priority);
    },
    [createTask],
  );

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
        onCreate={handleCreate}
      />

      <TaskDetailDrawer
        task={selectedTask}
        open={!!selectedTask}
        onClose={() => setSelectedTask(null)}
      />
    </Box>
  );
};