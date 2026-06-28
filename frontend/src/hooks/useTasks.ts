import { useState, useEffect, useCallback } from 'react';
import { Task } from '../types/task';
import { api } from '../services/api';

export function useTasks() {
  const [tasks, setTasks] = useState<Task[]>([]);
  const [loading, setLoading] = useState(true);

  const loadTasks = useCallback(async () => {
    try {
      const data = await api.listTasks();
      setTasks(data);
    } catch (err) {
      console.error('Failed to load tasks:', err);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    loadTasks();
  }, [loadTasks]);

  const createTask = async (title: string, description: string, priority: number, workspace: string) => {
    const task = await api.createTask({ title, description, priority, workspace: workspace || undefined });
    setTasks((prev) => [...prev, task]);
    return task;
  };

  const updateTask = async (id: string, data: { title: string; description: string; priority: number; workspace?: string; version: number }) => {
    const task = await api.updateTask(id, data);
    setTasks((prev) => prev.map((t) => (t.id === id ? task : t)));
    return task;
  };

  const deleteTask = async (id: string) => {
    await api.deleteTask(id);
    setTasks((prev) => prev.filter((t) => t.id !== id));
  };

  return { tasks, loading, createTask, updateTask, deleteTask, reload: loadTasks };
}
