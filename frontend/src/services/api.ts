import { Task, TaskDetail } from '../types/task';

const API_BASE = import.meta.env.VITE_API_BASE_URL || '';

async function request<T>(path: string, options?: RequestInit): Promise<T> {
  const res = await fetch(`${API_BASE}${path}`, {
    headers: { 'Content-Type': 'application/json' },
    ...options,
  });
  if (!res.ok) {
    throw new Error(`API error: ${res.status} ${res.statusText}`);
  }
  // 204 No Content (e.g. DELETE) — no body to parse.
  if (res.status === 204) {
    return undefined as T;
  }
  const json = await res.json();
  return json.data as T;
}

const tasksPath = (id: string) => `/api/v1/tasks/${id}`;

export const api = {
  createTask: (data: { title: string; description: string; priority: number }) =>
    request<Task>('/api/v1/tasks', {
      method: 'POST',
      body: JSON.stringify(data),
    }),

  // The backend returns { "data": null } when there are no tasks; coalesce to
  // an empty array so callers can always .filter/.map safely.
  listTasks: async (): Promise<Task[]> => {
    const data = await request<Task[] | null>('/api/v1/tasks');
    return data ?? [];
  },

  getTask: (id: string) => request<Task>(tasksPath(id)),

  // Full detail including phase outputs (what the harness produced per phase).
  getTaskDetail: (id: string) => request<TaskDetail>(tasksPath(id)),

  updateTask: (id: string, data: { title: string; description: string; priority: number; version: number }) =>
    request<Task>(tasksPath(id), {
      method: 'PUT',
      body: JSON.stringify(data),
    }),

  deleteTask: (id: string) =>
    request<void>(tasksPath(id), { method: 'DELETE' }),

  retryTask: (id: string) =>
    request<void>(tasksPath(id) + '/retry', { method: 'POST' }),
};