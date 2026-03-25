import { Box, Chip, Divider, List, ListItem, ListItemText, Stack, Typography } from '@mui/material';
import type { RuntimeEventV1 } from '../../pb/proto/scraper/runtime/v1/events_pb';
import {
  RuntimeEventKind,
  RuntimeEventSeverity,
  RuntimeEventSource,
} from '../../pb/proto/scraper/runtime/v1/events_pb';

interface RuntimeEventListProps {
  events: RuntimeEventV1[];
  loading?: boolean;
  emptyMessage?: string;
  onWorkflowClick?: (workflowId: string) => void;
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

function normalizeEnumLabel(raw: string | undefined): string {
  if (!raw) return 'Unknown';
  return raw
    .replace(/^RUNTIME_EVENT_(SOURCE|SEVERITY|KIND)_/, '')
    .toLowerCase()
    .split('_')
    .map((part) => part.charAt(0).toUpperCase() + part.slice(1))
    .join(' ');
}

function payloadSummary(event: RuntimeEventV1): string | null {
  if (!event.payload || typeof event.payload !== 'object') return null;
  if (typeof event.payload.errorMessage === 'string') {
    return event.payload.errorMessage;
  }
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

function payloadDetails(event: RuntimeEventV1): string[] {
  if (!event.payload || typeof event.payload !== 'object') return [];

  const details: string[] = [];
  if (typeof event.payload.attempt === 'number') {
    details.push(`Attempt ${event.payload.attempt}`);
  }
  if (typeof event.payload.errorCode === 'string') {
    details.push(`Code ${event.payload.errorCode}`);
  }
  if (typeof event.payload.retryable === 'boolean') {
    details.push(event.payload.retryable ? 'Retryable' : 'Non-retryable');
  }
  if (typeof event.payload.runnerKind === 'string') {
    details.push(`Runner ${event.payload.runnerKind}`);
  }
  if (typeof event.payload.emittedCount === 'number') {
    details.push(`${event.payload.emittedCount} emitted`);
  }
  if (typeof event.payload.recordWriteCount === 'number') {
    details.push(`${event.payload.recordWriteCount} records`);
  }
  if (typeof event.payload.statusCode === 'number') {
    details.push(`HTTP ${event.payload.statusCode}`);
  }
  if (typeof event.payload.commandPath === 'string') {
    details.push(event.payload.commandPath);
  }
  if (typeof event.payload.siteDbPath === 'string') {
    details.push(event.payload.siteDbPath);
  }
  if (typeof event.payload.workflowStatus === 'string') {
    details.push(`Workflow ${event.payload.workflowStatus}`);
  }
  if (typeof event.payload.path === 'string' && typeof event.payload.method === 'string') {
    details.push(`${event.payload.method} ${event.payload.path}`);
  }

  return details;
}

export function RuntimeEventList({
  events,
  loading = false,
  emptyMessage = 'No runtime events yet.',
  onWorkflowClick,
}: RuntimeEventListProps) {
  if (loading && events.length === 0) {
    return <Typography color="text.secondary">Loading runtime events...</Typography>;
  }

  if (events.length === 0) {
    return <Typography color="text.secondary">{emptyMessage}</Typography>;
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
                  <Chip label={normalizeEnumLabel(RuntimeEventKind[event.kind])} size="small" variant="outlined" />
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
                    {event.workflowId && (
                      <Chip
                        label={`Workflow: ${event.workflowId}`}
                        size="small"
                        variant="outlined"
                        onClick={onWorkflowClick ? () => onWorkflowClick(event.workflowId) : undefined}
                        clickable={Boolean(onWorkflowClick)}
                      />
                    )}
                    {event.site && (
                      <Typography variant="caption" color="text.secondary">
                        Site: {event.site}
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
                    {event.queue && (
                      <Typography variant="caption" color="text.secondary">
                        Queue: {event.queue}
                      </Typography>
                    )}
                    {event.requestId && (
                      <Typography variant="caption" color="text.secondary">
                        Request: {event.requestId}
                      </Typography>
                    )}
                    {event.artifactId && (
                      <Typography variant="caption" color="text.secondary">
                        Artifact: {event.artifactId}
                      </Typography>
                    )}
                    {payloadDetails(event).map((detail) => (
                      <Typography key={detail} variant="caption" color="text.secondary">
                        {detail}
                      </Typography>
                    ))}
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
