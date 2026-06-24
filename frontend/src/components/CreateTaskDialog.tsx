import React, { useState } from 'react';
import {
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  TextField,
  Button,
  MenuItem,
  Box,
  Typography,
  IconButton,
} from '@mui/material';
import CloseIcon from '@mui/icons-material/Close';
import { tokens } from '../theme/theme';

interface CreateTaskDialogProps {
  open: boolean;
  onClose: () => void;
  onCreate: (title: string, description: string, priority: number) => Promise<void>;
}

const mono = '"JetBrains Mono", monospace';

const PRIORITIES = [
  { value: 0, label: 'None', note: 'no priority' },
  { value: 1, label: 'Low', note: 'P1' },
  { value: 2, label: 'Medium', note: 'P2' },
  { value: 3, label: 'High', note: 'P3' },
];

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
      <Box sx={{ display: 'flex', alignItems: 'center', pr: 1.5 }}>
        <DialogTitle sx={{ flex: 1 }}>
          <Typography sx={{ fontFamily: mono, fontSize: '0.58rem', letterSpacing: '0.14em', color: tokens.ink.faint, textTransform: 'uppercase' }}>
            new task · intake
          </Typography>
          <Typography sx={{ fontFamily: '"Space Grotesk", sans-serif', fontWeight: 600, fontSize: '1.05rem', color: tokens.ink.text, mt: 0.25 }}>
            Dispatch a task to the agent
          </Typography>
        </DialogTitle>
        <IconButton onClick={onClose} size="small">
          <CloseIcon fontSize="small" />
        </IconButton>
      </Box>

      <DialogContent sx={{ pt: '0 !important' }}>
        <Typography sx={{ fontFamily: mono, fontSize: '0.56rem', letterSpacing: '0.12em', color: tokens.ink.faint, textTransform: 'uppercase', mt: 1, mb: 0.75 }}>
          title
        </Typography>
        <TextField
          autoFocus
          fullWidth
          placeholder="e.g. Build a REST endpoint for user signup"
          value={title}
          onChange={(e) => setTitle(e.target.value)}
        />

        <Typography sx={{ fontFamily: mono, fontSize: '0.56rem', letterSpacing: '0.12em', color: tokens.ink.faint, textTransform: 'uppercase', mt: 2, mb: 0.75 }}>
          brief
        </Typography>
        <TextField
          fullWidth
          multiline
          rows={4}
          placeholder="Describe what the agent should build — acceptance criteria, constraints, scope…"
          value={description}
          onChange={(e) => setDescription(e.target.value)}
        />

        <Typography sx={{ fontFamily: mono, fontSize: '0.56rem', letterSpacing: '0.12em', color: tokens.ink.faint, textTransform: 'uppercase', mt: 2, mb: 0.75 }}>
          priority
        </Typography>
        <TextField select fullWidth value={priority} onChange={(e) => setPriority(Number(e.target.value))}>
          {PRIORITIES.map((p) => (
            <MenuItem key={p.value} value={p.value} sx={{ fontFamily: mono, fontSize: '0.78rem' }}>
              {p.label}
              <Typography component="span" sx={{ ml: 1, fontFamily: mono, fontSize: '0.66rem', color: tokens.ink.faint }}>
                · {p.note}
              </Typography>
            </MenuItem>
          ))}
        </TextField>

        <Box
          sx={{
            mt: 2.5,
            p: 1.25,
            border: `1px solid ${tokens.border.hair}`,
            borderRadius: 1,
            bgcolor: tokens.bg.inset,
          }}
        >
          <Typography sx={{ fontFamily: mono, fontSize: '0.64rem', color: tokens.ink.dim, lineHeight: 1.6 }}>
            → the agent will run this through the pipeline:
          </Typography>
          <Typography sx={{ fontFamily: mono, fontSize: '0.62rem', color: tokens.ink.faint, mt: 0.25, letterSpacing: '0.04em' }}>
            planning → todo → doing → validating → testing → done
          </Typography>
        </Box>
      </DialogContent>

      <DialogActions sx={{ p: 2, pt: 1 }}>
        <Button onClick={onClose} size="small">
          Cancel
        </Button>
        <Button
          variant="contained"
          size="small"
          onClick={handleSubmit}
          disabled={!title.trim() || submitting}
          sx={{ bgcolor: tokens.signal.cyan, color: tokens.bg.base, fontWeight: 600, '&:hover': { bgcolor: '#5fdccd' }, '&.Mui-disabled': { bgcolor: tokens.border.hair, color: tokens.ink.faint } }}
        >
          {submitting ? 'Dispatching…' : 'Dispatch task'}
        </Button>
      </DialogActions>
    </Dialog>
  );
};