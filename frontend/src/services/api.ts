import { Task } from '../types/task';

const API_BASE = import.meta.env.VITE_API_BASE_URL || '';

async function request<T>(path: string, options?: RequestInit): Promise<T> {
  const res = await fetch(`${API_BASE}${path}`, {
    headers: { 'Content-Type': 'application/json' },
    ...options,
  });
  if (!res.ok) {
    throw new Error(`API error: ${res.status} ${res.statusText}`);
  }
  const json = await res.json();
  return json.data as T;
}

export const api = {
  createTask: (data: { title: string; description: string; priority: number }) =>
    request<Task>('/api/v1/tasks', {
      method: 'POST',
      body: JSON.stringify(data),
    }),

  listTasks: () => request<Task[]>('/api/v1/tasks'),

  getTask: (id: string) => request<Task>(`/api/v1/tasks/${id}`),

  updateTask: (id: string, data: { title: string; description: string; priority: number; version: number }) =>
    request<Task>(`/api/v1/tasks/${id}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    }),

  deleteTask: (id: string) =>
    request<void>(`/api/v1/tasks/${id}`, { method: 'DELETE' }),

  retryTask: (id: string) =>
    request<void>(`/api/v1/tasks/${id}/retry`, { method: 'POST' }),
};
