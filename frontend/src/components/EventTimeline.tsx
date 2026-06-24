import React from 'react';
import { Box, Typography, Timeline, TimelineItem, TimelineSeparator, TimelineDot, TimelineConnector, TimelineContent } from '@mui/lab';

interface EventTimelineProps {
  events: Array<{
    id: string;
    event_type: string;
    message: string;
    created_at: string;
  }>;
}

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
        events.map((event) => (
          <Box key={event.id} sx={{ mb: 1.5, pl: 2, borderLeft: '2px solid', borderColor: 'divider' }}>
            <Typography variant="caption" color="text.secondary">
              {new Date(event.created_at).toLocaleTimeString()}
            </Typography>
            <Typography variant="body2">{event.event_type}</Typography>
            {event.message && (
              <Typography variant="caption" color="text.secondary">
                {event.message}
              </Typography>
            )}
          </Box>
        ))
      )}
    </Box>
  );
};
