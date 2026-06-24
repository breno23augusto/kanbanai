import React from 'react';
import {
  Timeline,
  TimelineItem,
  TimelineSeparator,
  TimelineDot,
  TimelineConnector,
  TimelineContent,
} from '@mui/lab';
import { Box, Typography, Chip } from '@mui/material';

interface TimelineEvent {
  id: string;
  event_type: string;
  message: string;
  created_at: string;
}

interface EventTimelineProps {
  events: TimelineEvent[];
}

const eventTone = (eventType: string): 'success' | 'info' | 'warning' | 'error' | 'grey' => {
  if (eventType.endsWith('.completed') || eventType === 'task.created') return 'success';
  if (eventType.endsWith('.started')) return 'info';
  if (eventType.endsWith('.retry')) return 'warning';
  if (eventType.endsWith('.failed') || eventType === 'system.error') return 'error';
  return 'grey';
};

export const EventTimeline: React.FC<EventTimelineProps> = ({ events }) => {
  return (
    <Box>
      <Typography variant="subtitle2" sx={{ mb: 2 }}>
        Event Timeline
      </Typography>
      {events.length === 0 ? (
        <Typography variant="body2" color="text.secondary">
          No events yet
        </Typography>
      ) : (
        <Timeline sx={{ p: 0, m: 0 }}>
          {events.map((event) => (
            <TimelineItem key={event.id}>
              <TimelineSeparator>
                <TimelineDot color={eventTone(event.event_type)} sx={{ my: 0.5 }} />
                <TimelineConnector />
              </TimelineSeparator>
              <TimelineContent sx={{ py: 0.5, px: 2 }}>
                <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, flexWrap: 'wrap' }}>
                  <Chip
                    label={event.event_type}
                    size="small"
                    variant="outlined"
                    sx={{ fontSize: '0.7rem', height: 20 }}
                  />
                  <Typography variant="caption" color="text.secondary">
                    {new Date(event.created_at).toLocaleTimeString()}
                  </Typography>
                </Box>
                {event.message && (
                  <Typography variant="body2" color="text.secondary" sx={{ mt: 0.5 }}>
                    {event.message}
                  </Typography>
                )}
              </TimelineContent>
            </TimelineItem>
          ))}
        </Timeline>
      )}
    </Box>
  );
};