import { Box, Chip, Divider, List, ListItem, ListItemText, Stack, Typography } from '@mui/material';
import type { RuntimeEventV1 } from '../../pb/proto/scraper/runtime/v1/events_pb';
import { RuntimeEventSeverity, RuntimeEventSource } from '../../pb/proto/scraper/runtime/v1/events_pb';

interface RuntimeEventListProps {
  events: RuntimeEventV1[];
  loading?: boolean;
}

function formatTimestamp(event: RuntimeEventV1): string {
  if (!event.occurredAt) return 'Pending timestamp';
  const millis = Number(event.occurredAt.seconds) * 1000 + Math.floor(event.occurredAt.nanos / 1_000_000);
  return new Date(millis).toLocaleTimeString();
}

function severityColor(severity: RuntimeEventSeverity): 'default' | 'info' | 'warning' | 'error' {
  switch (severity) {
    case RuntimeEventSeverity.INFO:
      return 'info';
    case RuntimeEventSeverity.WARN:
      return 'warning';
    case RuntimeEventSeverity.ERROR:
      return 'error';
    default:
      return 'default';
  }
}

function payloadSummary(event: RuntimeEventV1): string | null {
  if (!event.payload || typeof event.payload !== 'object') return null;
  if (typeof event.payload.durationMillis === 'number') {
    return `${event.payload.durationMillis} ms`;
  }
  if (typeof event.payload.submittedCount === 'number') {
    return `${event.payload.submittedCount} ops submitted`;
  }
  if (typeof event.payload.artifactCount === 'number') {
    return `${event.payload.artifactCount} artifacts`;
  }
  if (typeof event.payload.error === 'string') {
    return event.payload.error;
  }
  return null;
}

export function RuntimeEventList({ events, loading = false }: RuntimeEventListProps) {
  if (loading && events.length === 0) {
    return <Typography color="text.secondary">Loading runtime events...</Typography>;
  }

  if (events.length === 0) {
    return <Typography color="text.secondary">No runtime events yet.</Typography>;
  }

  return (
    <List disablePadding>
      {events.map((event, index) => (
        <Box key={event.id || `${event.kind}-${index}`}>
          <ListItem alignItems="flex-start" sx={{ px: 0, py: 1.25 }}>
            <ListItemText
              primary={
                <Stack direction="row" spacing={1} alignItems="center" flexWrap="wrap" useFlexGap>
                  <Chip label={RuntimeEventSource[event.source]} size="small" variant="outlined" />
                  <Chip label={RuntimeEventSeverity[event.severity]} size="small" color={severityColor(event.severity)} />
                  <Typography variant="caption" color="text.secondary">
                    {formatTimestamp(event)}
                  </Typography>
                </Stack>
              }
              secondary={
                <Stack spacing={0.5} sx={{ mt: 0.75 }}>
                  <Typography variant="body2" color="text.primary">
                    {event.message || 'Runtime event'}
                  </Typography>
                  <Stack direction="row" spacing={1} flexWrap="wrap" useFlexGap>
                    {event.opId && (
                      <Typography variant="caption" color="text.secondary">
                        Op: {event.opId}
                      </Typography>
                    )}
                    {event.workerId && (
                      <Typography variant="caption" color="text.secondary">
                        Worker: {event.workerId}
                      </Typography>
                    )}
                    {payloadSummary(event) && (
                      <Typography variant="caption" color="text.secondary">
                        {payloadSummary(event)}
                      </Typography>
                    )}
                  </Stack>
                </Stack>
              }
            />
          </ListItem>
          {index < events.length - 1 && <Divider />}
        </Box>
      ))}
    </List>
  );
}
