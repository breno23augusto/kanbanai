import React, { useState } from 'react';
import {
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  TextField,
  Button,
  MenuItem,
} from '@mui/material';

interface CreateTaskDialogProps {
  open: boolean;
  onClose: () => void;
  onCreate: (title: string, description: string, priority: number) => Promise<void>;
}

export const CreateTaskDialog: React.FC<CreateTaskDialogProps> = ({ open, onClose, onCreate }) => {
  const [title, setTitle] = useState('');
  const [description, setDescription] = useState('');
  const [priority, setPriority] = useState(0);
  const [submitting, setSubmitting] = useState(false);

  const handleSubmit = async () => {
    if (!title.trim()) return;
    setSubmitting(true);
    try {
      await onCreate(title, description, priority);
      setTitle('');
      setDescription('');
      setPriority(0);
      onClose();
    } catch (err) {
      console.error('Failed to create task:', err);
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <Dialog open={open} onClose={onClose} maxWidth="sm" fullWidth>
      <DialogTitle>Create Task</DialogTitle>
      <DialogContent>
        <TextField
          autoFocus
          label="Title"
          fullWidth
          value={title}
          onChange={(e) => setTitle(e.target.value)}
          sx={{ mt: 1 }}
        />
        <TextField
          label="Description"
          fullWidth
          multiline
          rows={3}
          value={description}
          onChange={(e) => setDescription(e.target.value)}
          sx={{ mt: 2 }}
        />
        <TextField
          select
          label="Priority"
          fullWidth
          value={priority}
          onChange={(e) => setPriority(Number(e.target.value))}
          sx={{ mt: 2 }}
        >
          <MenuItem value={0}>None</MenuItem>
          <MenuItem value={1}>Low</MenuItem>
          <MenuItem value={2}>Medium</MenuItem>
          <MenuItem value={3}>High</MenuItem>
        </TextField>
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose}>Cancel</Button>
        <Button variant="contained" onClick={handleSubmit} disabled={!title.trim() || submitting}>
          Create
        </Button>
      </DialogActions>
    </Dialog>
  );
};
