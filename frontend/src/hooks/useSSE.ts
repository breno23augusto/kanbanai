import { useEffect, useRef, useCallback } from 'react';
import { SSEEvent } from '../types/event';

const SSE_URL = import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080';

export function useSSE(onEvent: (event: SSEEvent) => void) {
  const eventSourceRef = useRef<EventSource | null>(null);
  const onEventRef = useRef(onEvent);
  onEventRef.current = onEvent;

  const connect = useCallback(() => {
    if (eventSourceRef.current) {
      eventSourceRef.current.close();
    }

    const source = new EventSource(`${SSE_URL}/api/v1/events`);

    source.addEventListener('message', (e) => {
      try {
        const data = JSON.parse(e.data);
        onEventRef.current({ type: e.type || 'message', data });
      } catch (err) {
        console.error('SSE parse error:', err);
      }
    });

    source.onerror = () => {
      console.warn('SSE connection error, reconnecting...');
      source.close();
      setTimeout(connect, 3000);
    };

    eventSourceRef.current = source;
  }, []);

  useEffect(() => {
    connect();
    return () => {
      if (eventSourceRef.current) {
        eventSourceRef.current.close();
      }
    };
  }, [connect]);
}
