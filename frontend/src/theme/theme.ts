import { createTheme } from '@mui/material/styles';

/**
 * KanbanAI instrument-console tokens.
 *
 * Deep slate-ink base (not pure black, not the Material #121212 default),
 * hairline borders, a desaturated cyan-teal system signal paired with warm
 * amber for "running". Phase lamps trace a cool → warm → cool journey so the
 * pipeline reads as a progression rather than a row of random colors.
 */
export const tokens = {
  bg: {
    base: '#0E151C',
    panel: '#161F2A',
    panelAlt: '#1B2532',
    inset: '#0B1118',
  },
  border: {
    hair: '#26323F',
    strong: '#34465A',
  },
  ink: {
    text: '#DCE4EE',
    dim: '#8B9AAD',
    faint: '#5B6B7E',
  },
  signal: {
    cyan: '#4FD1C5',
    amber: '#F0A93B',
    sage: '#7FB98A',
    coral: '#E8696B',
    violet: '#9B8AE8',
  },
  phase: {
    planning: '#9B8AE8',
    todo: '#5B9BE8',
    doing: '#F0A93B',
    validating: '#E8C46B',
    testing: '#E89B6B',
    done: '#7FB98A',
  } as Record<string, string>,
  status: {
    pending: '#F0A93B',
    in_progress: '#4FD1C5',
    completed: '#7FB98A',
    failed: '#E8696B',
    cancelled: '#5B6B7E',
    paused: '#E8C46B',
  } as Record<string, string>,
};

const mono = '"JetBrains Mono", ui-monospace, monospace';
const sans = '"Space Grotesk", system-ui, sans-serif';

const theme = createTheme({
  palette: {
    mode: 'dark',
    primary: { main: tokens.signal.cyan },
    secondary: { main: tokens.signal.amber },
    error: { main: tokens.signal.coral },
    success: { main: tokens.signal.sage },
    warning: { main: tokens.signal.amber },
    background: {
      default: tokens.bg.base,
      paper: tokens.bg.panel,
    },
    text: {
      primary: tokens.ink.text,
      secondary: tokens.ink.dim,
      disabled: tokens.ink.faint,
    },
    divider: tokens.border.hair,
  },
  typography: {
    fontFamily: sans,
    h5: { fontFamily: sans, fontWeight: 700, letterSpacing: '-0.01em' },
    h6: { fontFamily: sans, fontWeight: 600, letterSpacing: '-0.01em' },
    subtitle1: { fontFamily: sans, fontWeight: 600 },
    subtitle2: { fontFamily: sans, fontWeight: 600 },
    body1: { fontFamily: sans },
    body2: { fontFamily: sans, fontSize: '0.875rem' },
    button: { fontFamily: sans, fontWeight: 600, letterSpacing: '0.01em' },
    caption: { fontFamily: mono, fontSize: '0.72rem', letterSpacing: '0.02em' },
    overline: {
      fontFamily: mono,
      fontSize: '0.68rem',
      letterSpacing: '0.14em',
      fontWeight: 500,
    },
  },
  shape: { borderRadius: 4 },
  components: {
    MuiCssBaseline: {
      styleOverrides: {
        body: { fontFeatureSettings: '"ss01","cv01"' },
      },
    },
    MuiPaper: {
      defaultProps: { elevation: 0 },
      styleOverrides: {
        root: {
          backgroundImage: 'none',
          backgroundColor: tokens.bg.panel,
        },
      },
    },
    MuiCard: {
      defaultProps: { elevation: 0 },
      styleOverrides: {
        root: {
          backgroundColor: tokens.bg.panel,
          border: `1px solid ${tokens.border.hair}`,
          borderRadius: 4,
        },
      },
    },
    MuiDivider: {
      styleOverrides: { root: { borderColor: tokens.border.hair } },
    },
    MuiChip: {
      styleOverrides: {
        root: { fontFamily: mono, borderRadius: 3 },
      },
    },
    MuiButton: {
      styleOverrides: {
        root: { textTransform: 'none', borderRadius: 4 },
        outlined: {
          borderColor: tokens.border.strong,
          '&:hover': { borderColor: tokens.signal.cyan, backgroundColor: 'rgba(79,209,197,0.06)' },
        },
      },
    },
    MuiIconButton: {
      styleOverrides: {
        root: { color: tokens.ink.dim, '&:hover': { color: tokens.ink.text, backgroundColor: 'rgba(255,255,255,0.04)' } },
      },
    },
    MuiTextField: {
      defaultProps: {
        variant: 'outlined',
        size: 'small',
      },
      styleOverrides: {
        root: { '& .MuiOutlinedInput-root': { fontFamily: sans } },
      },
    },
    MuiOutlinedInput: {
      styleOverrides: {
        notchedOutline: { borderColor: tokens.border.hair },
        root: {
          backgroundColor: tokens.bg.inset,
          '&:hover .MuiOutlinedInput-notchedOutline': { borderColor: tokens.border.strong },
          '&.Mui-focused .MuiOutlinedInput-notchedOutline': { borderColor: tokens.signal.cyan, borderWidth: 1 },
        },
      },
    },
    MuiInputLabel: {
      styleOverrides: { root: { fontFamily: mono, fontSize: '0.72rem', letterSpacing: '0.04em', color: tokens.ink.dim } },
    },
    MuiDialog: {
      styleOverrides: {
        paper: {
          backgroundColor: tokens.bg.panel,
          border: `1px solid ${tokens.border.hair}`,
          borderRadius: 4,
        },
      },
    },
    MuiDialogTitle: {
      styleOverrides: { root: { fontFamily: sans, fontWeight: 600, fontSize: '1rem' } },
    },
    MuiDrawer: {
      styleOverrides: { paper: { backgroundColor: tokens.bg.panel, borderLeft: `1px solid ${tokens.border.hair}` } },
    },
    MuiAccordion: {
      defaultProps: { elevation: 0 },
      styleOverrides: {
        root: {
          backgroundColor: 'transparent',
          borderBottom: `1px solid ${tokens.border.hair}`,
          '&:before': { display: 'none' },
          '&.Mui-expanded': { margin: 0 },
        },
      },
    },
    MuiAccordionSummary: {
      styleOverrides: { root: { minHeight: 44, padding: '0 4px' } },
    },
    MuiTooltip: {
      styleOverrides: {
        tooltip: {
          backgroundColor: tokens.bg.panelAlt,
          border: `1px solid ${tokens.border.hair}`,
          fontFamily: mono,
          fontSize: '0.72rem',
          padding: '4px 8px',
        },
      },
    },
    MuiStepLabel: {
      styleOverrides: { label: { fontFamily: mono, fontSize: '0.7rem' } },
    },
  },
});

export default theme;