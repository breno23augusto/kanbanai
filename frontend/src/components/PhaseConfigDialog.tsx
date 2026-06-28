import React, { useEffect, useState } from 'react';
import {
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  TextField,
  Button,
  Box,
  Typography,
  IconButton,
  Tooltip,
  Stack,
} from '@mui/material';
import CloseIcon from '@mui/icons-material/Close';
import RestartAltIcon from '@mui/icons-material/RestartAlt';
import { tokens } from '../theme/theme';
import { PhaseConfig, PhaseConfigInput } from '../types/config';
import { api } from '../services/api';

const mono = '"JetBrains Mono", monospace';
const sans = '"Space Grotesk", sans-serif';

interface PhaseConfigDialogProps {
  open: boolean;
  onClose: () => void;
}

// Editable per-lane fields (empty string = inherit env default). Retries/timeout
// are kept as strings in the form so the user can clear them; parsed to int on save.
interface FormState {
  model: string;
  harness_cmd: string;
  max_retries: string;
  timeout_sec: string;
}

const PHASE_LABEL: Record<string, string> = {
  planning: 'Planning',
  todo: 'Todo',
  doing: 'Doing',
  validating: 'Validating',
  testing: 'Testing',
};

function formFromConfig(c: PhaseConfig): FormState {
  // Show the effective value (model/cmd), or empty if it equals the default.
  const eff = (v: string, d: string) => (v && v !== d ? v : '');
  return {
    model: eff(c.model, c.default_model),
    harness_cmd: eff(c.harness_cmd, c.default_harness_cmd),
    max_retries: c.max_retries !== c.default_max_retries ? String(c.max_retries) : '',
    timeout_sec: c.timeout_sec !== c.default_timeout_sec ? String(c.timeout_sec) : '',
  };
}

export const PhaseConfigDialog: React.FC<PhaseConfigDialogProps> = ({ open, onClose }) => {
  const [configs, setConfigs] = useState<PhaseConfig[]>([]);
  const [forms, setForms] = useState<Record<string, FormState>>({});
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const load = async () => {
    setLoading(true);
    setError(null);
    try {
      const { phases } = await api.getPhaseConfigs();
      setConfigs(phases);
      const next: Record<string, FormState> = {};
      for (const p of phases) next[p.phase] = formFromConfig(p);
      setForms(next);
    } catch (e) {
      setError(e instanceof Error ? e.message : 'failed to load config');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    if (open) load();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open]);

  const updateField = (phase: string, field: keyof FormState, value: string) => {
    setForms((prev) => ({ ...prev, [phase]: { ...prev[phase], [field]: value } }));
  };

  const resetPhase = (phase: string) => {
    setForms((prev) => ({
      ...prev,
      [phase]: { model: '', harness_cmd: '', max_retries: '', timeout_sec: '' },
    }));
  };

  const resetAll = () => {
    const next: Record<string, FormState> = {};
    for (const c of configs) next[c.phase] = { model: '', harness_cmd: '', max_retries: '', timeout_sec: '' };
    setForms(next);
  };

  const handleSave = async () => {
    setSaving(true);
    setError(null);
    try {
      const inputs: PhaseConfigInput[] = configs.map((c) => {
        const f = forms[c.phase] ?? { model: '', harness_cmd: '', max_retries: '', timeout_sec: '' };
        return {
          phase: c.phase,
          model: f.model.trim(),
          harness_cmd: f.harness_cmd.trim(),
          max_retries: f.max_retries.trim() === '' ? 0 : Math.max(0, parseInt(f.max_retries, 10) || 0),
          timeout_sec: f.timeout_sec.trim() === '' ? 0 : Math.max(0, parseInt(f.timeout_sec, 10) || 0),
        };
      });
      const { phases } = await api.updatePhaseConfigs(inputs);
      setConfigs(phases);
      const next: Record<string, FormState> = {};
      for (const p of phases) next[p.phase] = formFromConfig(p);
      setForms(next);
      onClose();
    } catch (e) {
      setError(e instanceof Error ? e.message : 'failed to save config');
    } finally {
      setSaving(false);
    }
  };

  return (
    <Dialog open={open} onClose={onClose} maxWidth="lg" fullWidth PaperProps={{ sx: { bgcolor: tokens.bg.panel } }}>
      <Box sx={{ display: 'flex', alignItems: 'center', pr: 1.5 }}>
        <DialogTitle sx={{ flex: 1 }}>
          <Typography sx={{ fontFamily: mono, fontSize: '0.58rem', letterSpacing: '0.14em', color: tokens.ink.faint, textTransform: 'uppercase' }}>
            Lane Configuration
          </Typography>
          <Typography sx={{ fontFamily: sans, fontWeight: 700, fontSize: '1.05rem', color: tokens.ink.text, mt: 0.25 }}>
            Harness & Model per Lane
          </Typography>
        </DialogTitle>
        <Tooltip title="Reset all lanes to env defaults">
          <IconButton size="small" onClick={resetAll} sx={{ color: tokens.ink.dim, mr: 0.5 }}>
            <RestartAltIcon fontSize="small" />
          </IconButton>
        </Tooltip>
        <IconButton onClick={onClose} size="small" sx={{ color: tokens.ink.dim }}>
          <CloseIcon fontSize="small" />
        </IconButton>
      </Box>

      <DialogContent dividers sx={{ borderColor: tokens.border.hair }}>
        <Typography sx={{ fontFamily: sans, fontSize: '0.78rem', color: tokens.ink.dim, mb: 2, lineHeight: 1.5 }}>
          Leave a field empty to inherit the server default (shown as placeholder). Changes apply to the
          next phase dispatch — no restart needed.
        </Typography>

        {error && (
          <Box sx={{ mb: 2, p: 1, border: `1px solid ${tokens.signal.coral}55`, borderRadius: 1, bgcolor: `${tokens.signal.coral}0a` }}>
            <Typography sx={{ fontFamily: mono, fontSize: '0.72rem', color: tokens.signal.coral }}>{error}</Typography>
          </Box>
        )}

        <Stack spacing={2}>
          {configs.map((c) => {
            const f = forms[c.phase] ?? { model: '', harness_cmd: '', max_retries: '', timeout_sec: '' };
            return (
              <Box key={c.phase} sx={{ border: `1px solid ${tokens.border.hair}`, borderRadius: 1, p: 1.5, bgcolor: tokens.bg.panelAlt }}>
                <Box sx={{ display: 'flex', alignItems: 'center', mb: 1.5 }}>
                  <Box sx={{ width: 8, height: 8, borderRadius: '50%', bgcolor: tokens.phase[c.phase] ?? tokens.signal.cyan, mr: 1 }} />
                  <Typography sx={{ fontFamily: mono, fontSize: '0.62rem', letterSpacing: '0.12em', textTransform: 'uppercase', color: tokens.ink.text, fontWeight: 700 }}>
                    {PHASE_LABEL[c.phase] ?? c.phase}
                  </Typography>
                  <Box sx={{ flex: 1 }} />
                  <Tooltip title="Reset this lane to defaults">
                    <IconButton size="small" onClick={() => resetPhase(c.phase)} sx={{ color: tokens.ink.faint }}>
                      <RestartAltIcon sx={{ fontSize: '0.95rem' }} />
                    </IconButton>
                  </Tooltip>
                </Box>
                <Box sx={{ display: 'grid', gridTemplateColumns: { xs: '1fr', sm: '1fr 1fr' }, gap: 1.5 }}>
                  <TextField
                    size="small"
                    label="Model"
                    value={f.model}
                    onChange={(e) => updateField(c.phase, 'model', e.target.value)}
                    placeholder={c.default_model || '(none)'}
                    helperText={`default: ${c.default_model || '—'}`}
                    sx={fieldSx}
                  />
                  <TextField
                    size="small"
                    label="Harness cmd"
                    value={f.harness_cmd}
                    onChange={(e) => updateField(c.phase, 'harness_cmd', e.target.value)}
                    placeholder={c.default_harness_cmd || '(none)'}
                    helperText={`default: ${c.default_harness_cmd || '—'}`}
                    sx={fieldSx}
                  />
                  <TextField
                    size="small"
                    label="Max retries"
                    type="number"
                    value={f.max_retries}
                    onChange={(e) => updateField(c.phase, 'max_retries', e.target.value)}
                    placeholder={String(c.default_max_retries)}
                    helperText={`default: ${c.default_max_retries}`}
                    sx={fieldSx}
                  />
                  <TextField
                    size="small"
                    label="Timeout (sec)"
                    type="number"
                    value={f.timeout_sec}
                    onChange={(e) => updateField(c.phase, 'timeout_sec', e.target.value)}
                    placeholder={String(c.default_timeout_sec)}
                    helperText={`default: ${c.default_timeout_sec}`}
                    sx={fieldSx}
                  />
                </Box>
              </Box>
            );
          })}
          {loading && (
            <Typography sx={{ fontFamily: mono, fontSize: '0.7rem', color: tokens.ink.faint }}>loading…</Typography>
          )}
        </Stack>
      </DialogContent>

      <DialogActions sx={{ borderColor: tokens.border.hair, p: 1.5 }}>
        <Button onClick={onClose} sx={{ color: tokens.ink.dim }}>Cancel</Button>
        <Button
          onClick={handleSave}
          variant="outlined"
          disabled={saving || loading}
          sx={{ borderColor: tokens.signal.cyan, color: tokens.signal.cyan, '&:hover': { borderColor: tokens.signal.cyan, bgcolor: 'rgba(79,209,197,0.08)' } }}
        >
          {saving ? 'Saving…' : 'Save'}
        </Button>
      </DialogActions>
    </Dialog>
  );
};

const fieldSx = {
  '& .MuiInputBase-input': { fontFamily: mono, fontSize: '0.78rem', color: tokens.ink.text },
  '& .MuiInputLabel-root': { fontFamily: sans, fontSize: '0.72rem', color: tokens.ink.dim },
  '& .MuiFormHelperText-root': { fontFamily: mono, fontSize: '0.6rem', color: tokens.ink.faint },
  '& .MuiOutlinedInput-root': {
    bgcolor: tokens.bg.inset,
    '& fieldset': { borderColor: tokens.border.hair },
    '&:hover fieldset': { borderColor: tokens.border.strong },
    '&.Mui-focused fieldset': { borderColor: tokens.signal.cyan },
  },
};