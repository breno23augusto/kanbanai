import React from 'react';
import { Box, Typography, LinearProgress } from '@mui/material';

interface PhaseProgressProps {
  phase: string;
  progress: number;
  message?: string;
}

export const PhaseProgress: React.FC<PhaseProgressProps> = ({ phase, progress, message }) => {
  return (
    <Box sx={{ mb: 2 }}>
      <Typography variant="caption" color="text.secondary" sx={{ textTransform: 'capitalize' }}>
        {phase}
      </Typography>
      <LinearProgress variant="determinate" value={progress} sx={{ mt: 0.5 }} />
      {message && (
        <Typography variant="caption" color="text.secondary" sx={{ mt: 0.5, display: 'block' }}>
          {message}
        </Typography>
      )}
    </Box>
  );
};
