import React from 'react';
import { Box } from '@mui/material';

interface LampProps {
  color: string;
  size?: number;
  pulse?: boolean;
  ring?: boolean;
}

/** A signal lamp — the console vernacular for live/instrument state. */
export const Lamp: React.FC<LampProps> = ({ color, size = 8, pulse = false, ring = false }) => (
  <Box
    component="span"
    className={pulse ? 'kai-pulse' : undefined}
    sx={{
      display: 'inline-block',
      width: size,
      height: size,
      borderRadius: '50%',
      bgcolor: color,
      color, // feeds the pulse's currentColor
      flexShrink: 0,
      boxShadow: ring ? `0 0 0 2px ${color}33, 0 0 6px ${color}55` : undefined,
    }}
  />
);