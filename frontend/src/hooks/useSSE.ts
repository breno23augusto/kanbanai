import { useEffect, useRef, useCallback } from 'react';
import { SSEEvent } from '../types/event';

const SSE_BASE = import.meta.env.VITE_API_BASE_URL || '';

// Named SSE event types the frontend reacts to. The backend (sse_handler.go)
// emits each event with `event: <type>` via c.SSEvent, so EventSource only
// delivers them through addEventListener(type) — the generic `message` listener
// never fires. We register a listener for every type the UI cares about and
// forward the parsed payload to the consumer.
const EVENT_TYPES = [
  'task.created',
  'task.updated',
  'task.deleted',
  'task.status_changed',
  'task.paused',
  'task.resumed',
  'lane.transition.started',
  'lane.transition.completed',
  'lane.transition.failed',
  'lane.reopened',
  'subtask.created',
  'subtask.updated',
  'phase.planning.started',
  'phase.planning.completed',
  'phase.planning.failed',
  'phase.planning.retry',
  'phase.todo.started',
  'phase.todo.completed',
  'phase.todo.failed',
  'phase.todo.retry',
  'phase.doing.started',
  'phase.doing.completed',
  'phase.doing.failed',
  'phase.doing.retry',
  'phase.validating.started',
  'phase.validating.completed',
  'phase.validating.failed',
  'phase.validating.retry',
  'phase.testing.started',
  'phase.testing.completed',
  'phase.testing.failed',
  'phase.testing.retry',
  'phase.done.reached',
  'harness.error.occurred',
  'phase_configs.updated',
] as const;

export function useSSE(onEvent: (event: SSEEvent) => void) {
  const eventSourceRef = useRef<EventSource | null>(null);
  const reconnectRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const onEventRef = useRef(onEvent);
  onEventRef.current = onEvent;

  const connect = useCallback(() => {
    if (eventSourceRef.current) {
      eventSourceRef.current.close();
    }

    const source = new EventSource(`${SSE_BASE}/api/v1/events`);
    eventSourceRef.current = source;

    const handle = (type: string) => (e: MessageEvent) => {
      try {
        const data = e.data ? JSON.parse(e.data) : null;
        onEventRef.current({ type, data });
      } catch (err) {
        console.error('SSE parse error:', err);
      }
    };

    for (const type of EVENT_TYPES) {
      source.addEventListener(type, handle(type) as EventListener);
    }
    // Fallback for any unnamed/default message events.
    source.addEventListener('message', handle('message') as EventListener);

    source.onerror = () => {
      console.warn('SSE connection error, reconnecting in 3s...');
      source.close();
      reconnectRef.current = setTimeout(connect, 3000);
    };
  }, []);

  useEffect(() => {
    connect();
    return () => {
      if (reconnectRef.current) clearTimeout(reconnectRef.current);
      if (eventSourceRef.current) {
        eventSourceRef.current.close();
      }
    };
  }, [connect]);
}