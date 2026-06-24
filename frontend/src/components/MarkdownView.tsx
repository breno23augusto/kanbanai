import React from 'react';
import ReactMarkdown from 'react-markdown';
import remarkGfm from 'remark-gfm';
import { Box, Typography, Checkbox } from '@mui/material';
import { tokens } from '../theme/theme';

const mono = '"JetBrains Mono", ui-monospace, monospace';
const sans = '"Space Grotesk", system-ui, sans-serif';

/**
 * MarkdownView
 *
 * Renders phase output (harness-produced markdown) with GitHub-flavored
 * markdown: headings, lists, tables, fenced code, and task-list checkboxes
 * (`- [ ]` / `- [x]`). The task-list checkboxes are the primary way subtasks
 * and their status surface in the UI — the planning/todo phases emit them and
 * they render here as read-only checks with sage/coral accents.
 *
 * Styling is hand-rolled against the KanbanAI token system so it matches the
 * instrument-console aesthetic instead of pulling in a generic markdown CSS
 * bundle.
 */
export const MarkdownView: React.FC<{ source: string }> = ({ source }) => {
  return (
    <Box
      sx={{
        // root markdown container
        fontFamily: sans,
        fontSize: '0.82rem',
        lineHeight: 1.65,
        color: tokens.ink.text,
        bgcolor: tokens.bg.inset,
        border: `1px solid ${tokens.border.hair}`,
        borderRadius: 1,
        p: 1.5,
        maxHeight: 520,
        overflowY: 'auto',
        '& > :first-child': { mt: 0 },
        '& > :last-child': { mb: 0 },
        // block spacing
        '& p': { my: 0.75 },
        '& h1, & h2, & h3, & h4': {
          fontFamily: sans,
          fontWeight: 700,
          color: tokens.ink.text,
          lineHeight: 1.25,
          mt: 1.5,
          mb: 0.5,
        },
        '& h1': { fontSize: '1.05rem' },
        '& h2': { fontSize: '0.96rem' },
        '& h3': { fontSize: '0.88rem' },
        '& h4': { fontSize: '0.82rem', color: tokens.ink.dim },
        '& ul, & ol': { pl: 3, my: 0.75 },
        '& li': { my: 0.25 },
        '& li > p': { my: 0.25 },
        // task list — remove default bullet, align the checkbox
        '& ul:has(> li > input[type="checkbox"])': { listStyle: 'none', pl: 1 },
        '& li:has(> input[type="checkbox"])': {
          display: 'flex',
          alignItems: 'flex-start',
          gap: 0.5,
          ml: -1,
        },
        '& a': { color: tokens.signal.cyan, textDecoration: 'none', '&:hover': { textDecoration: 'underline' } },
        '& strong': { color: tokens.ink.text, fontWeight: 700 },
        '& em': { color: tokens.ink.dim },
        '& blockquote': {
          borderLeft: `2px solid ${tokens.border.strong}`,
          pl: 1.5,
          my: 1,
          color: tokens.ink.dim,
        },
        '& hr': { border: 'none', borderTop: `1px solid ${tokens.border.hair}`, my: 1.25 },
        // inline code
        '& code': {
          fontFamily: mono,
          fontSize: '0.78rem',
          color: tokens.signal.cyan,
          bgcolor: tokens.bg.panelAlt,
          px: 0.4,
          borderRadius: 0.5,
          py: 0.1,
        },
        // fenced code blocks
        '& pre': {
          fontFamily: mono,
          fontSize: '0.76rem',
          lineHeight: 1.55,
          color: tokens.ink.dim,
          bgcolor: tokens.bg.base,
          border: `1px solid ${tokens.border.hair}`,
          borderRadius: 1,
          p: 1.25,
          overflowX: 'auto',
          my: 1,
        },
        '& pre code': {
          bgcolor: 'transparent',
          color: tokens.ink.dim,
          px: 0,
          py: 0,
          fontSize: 'inherit',
        },
        // tables
        '& table': {
          borderCollapse: 'collapse',
          width: '100%',
          my: 1,
          fontSize: '0.78rem',
        },
        '& th, & td': {
          border: `1px solid ${tokens.border.hair}`,
          px: 0.75,
          py: 0.5,
          textAlign: 'left',
          verticalAlign: 'top',
        },
        '& th': {
          fontFamily: mono,
          fontSize: '0.66rem',
          letterSpacing: '0.06em',
          textTransform: 'uppercase',
          color: tokens.ink.dim,
          bgcolor: tokens.bg.panelAlt,
        },
        '& td code': { fontSize: '0.74rem' },
      }}
    >
      <ReactMarkdown
        remarkPlugins={[remarkGfm]}
        components={{
          // Render GFM task-list checkboxes as MUI Checkboxes so they pick up
          // the theme and read clearly as subtask status indicators. They are
          // read-only — the source of truth is the harness output.
          input: ({ node, ...props }) => {
            const checked = props.checked ?? props.defaultChecked ?? false;
            return (
              <Checkbox
                checked={checked}
                size="small"
                readOnly
                disableRipple
                sx={{
                  p: 0.25,
                  mt: -0.25,
                  color: checked ? tokens.signal.sage : tokens.border.strong,
                  '&.Mui-checked': { color: tokens.signal.sage },
                  '&.Mui-disabled': { color: checked ? tokens.signal.sage : tokens.border.strong },
                }}
              />
            );
          },
          a: ({ node, ...props }) => <a {...props} target="_blank" rel="noreferrer noopener" />,
        }}
      >
        {source}
      </ReactMarkdown>
    </Box>
  );
};

/**
 * Extract GFM task-list items from a markdown string. Returns a list of
 * { label, done } for every `- [ ]` / `- [x]` / `* [ ]` / `* [x]` line
 * (leading whitespace tolerant). Used to power the subtask status summary
 * shown above the rendered output.
 */
export interface SubtaskItem {
  label: string;
  done: boolean;
}

export function extractSubtasks(source: string): SubtaskItem[] {
  const items: SubtaskItem[] = [];
  const re = /^\s*[-*+]\s+\[([ xX])\]\s+(.+?)\s*$/;
  for (const line of source.split(/\r?\n/)) {
    const m = line.match(re);
    if (!m) continue;
    items.push({ done: m[1].toLowerCase() === 'x', label: m[2] });
  }
  return items;
}

/**
 * SubtaskSummary — compact status line showing how many of the embedded
 * subtasks (GFM task-list items) are done vs. total. Only renders when the
 * phase output actually contains task-list items.
 */
export const SubtaskSummary: React.FC<{ source: string }> = ({ source }) => {
  const items = extractSubtasks(source);
  if (items.length === 0) return null;
  const done = items.filter((i) => i.done).length;
  const total = items.length;
  const allDone = done === total;
  const noneDone = done === 0;
  const color = allDone ? tokens.signal.sage : noneDone ? tokens.signal.amber : tokens.signal.cyan;

  return (
    <Box
      sx={{
        display: 'flex',
        alignItems: 'center',
        gap: 1,
        mb: 1,
        px: 1,
        py: 0.5,
        border: `1px solid ${tokens.border.hair}`,
        borderLeft: `2px solid ${color}`,
        borderRadius: 1,
        bgcolor: tokens.bg.inset,
      }}
    >
      <Box
        component="span"
        sx={{
          width: 7,
          height: 7,
          borderRadius: '50%',
          bgcolor: color,
          flexShrink: 0,
        }}
      />
      <Typography
        sx={{ fontFamily: mono, fontSize: '0.64rem', letterSpacing: '0.08em', textTransform: 'uppercase', color: tokens.ink.dim }}
      >
        subtasks
      </Typography>
      <Typography sx={{ fontFamily: mono, fontSize: '0.72rem', color }}>
        {done}/{total} done
      </Typography>
      {/* per-item dots for a quick at-a-glance read */}
      <Box sx={{ display: 'flex', gap: 0.25, flexWrap: 'wrap', ml: 'auto' }}>
        {items.map((it, idx) => (
          <Box
            key={idx}
            title={it.label}
            sx={{
              width: 8,
              height: 8,
              borderRadius: 0.5,
              bgcolor: it.done ? tokens.signal.sage : tokens.border.strong,
            }}
          />
        ))}
      </Box>
    </Box>
  );
};