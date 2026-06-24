export interface Task {
  id: string;
  title: string;
  description: string;
  current_phase: Phase;
  status: Status;
  priority: number;
  version: number;
  created_at: string;
  updated_at: string;
}

export type Phase = 'planning' | 'todo' | 'doing' | 'validating' | 'testing' | 'done';

export type Status = 'pending' | 'in_progress' | 'completed' | 'failed' | 'cancelled';

export const PHASE_ORDER: Phase[] = [
  'planning',
  'todo',
  'doing',
  'validating',
  'testing',
  'done',
];

export const PHASE_LABELS: Record<Phase, string> = {
  planning: 'Planning',
  todo: 'Todo',
  doing: 'Doing',
  validating: 'Validating',
  testing: 'Testing',
  done: 'Done',
};
