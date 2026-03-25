import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import {
  Box,
  Card,
  CardContent,
  Chip,
  MenuItem,
  Stack,
  TextField,
  Typography,
} from '@mui/material';
import { RuntimeEventList } from '../components/workflows/RuntimeEventList';
import {
  useRuntimeEventFeed,
  type RuntimeEventConnectionState,
} from '../features/runtime-events/runtimeEventFeed';
import {
  RuntimeEventSeverity,
  RuntimeEventSource,
} from '../pb/proto/scraper/runtime/v1/events_pb';

function connectionColor(state: RuntimeEventConnectionState): 'default' | 'success' | 'warning' | 'error' {
  switch (state) {
    case 'live':
      return 'success';
    case 'connecting':
      return 'warning';
    case 'error':
      return 'error';
    default:
      return 'default';
  }
}

function formatLastEventAt(lastEventAt: number | null): string {
  if (!lastEventAt) return 'No events received yet';
  return `Last event ${new Date(lastEventAt).toLocaleString()}`;
}

export function RuntimeEventsPage() {
  const navigate = useNavigate();
  const [workflowId, setWorkflowId] = useState('');
  const [opId, setOpId] = useState('');
  const [site, setSite] = useState('');
  const [workerId, setWorkerId] = useState('');
  const [severity, setSeverity] = useState<RuntimeEventSeverity | 'all'>('all');
  const [source, setSource] = useState<RuntimeEventSource | 'all'>('all');

  const { events, isLoadingHistory, connectionState, lastEventAt } = useRuntimeEventFeed({
    serverFilters: {
      workflowId: workflowId || undefined,
      opId: opId || undefined,
      site: site || undefined,
      workerId: workerId || undefined,
      limit: 100,
    },
    clientFilters: {
      severity,
      source,
    },
  });

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
      <Card>
        <CardContent>
          <Stack spacing={1.5}>
            <Box>
              <Typography variant="h5">Runtime Events</Typography>
              <Typography variant="body2" color="text.secondary">
                Global operator console for recent runtime history and live event streaming.
              </Typography>
            </Box>

            <Stack direction="row" spacing={1} useFlexGap flexWrap="wrap">
              <Chip label={`Stream: ${connectionState}`} color={connectionColor(connectionState)} />
              <Chip label={`${events.length} events`} variant="outlined" />
              <Chip label={formatLastEventAt(lastEventAt)} variant="outlined" />
            </Stack>

            <Box
              sx={{
                display: 'grid',
                gap: 1.5,
                gridTemplateColumns: {
                  xs: '1fr',
                  md: 'repeat(3, minmax(0, 1fr))',
                },
              }}
            >
              <TextField
                label="Workflow ID"
                value={workflowId}
                onChange={(event) => setWorkflowId(event.target.value)}
                size="small"
              />
              <TextField
                label="Op ID"
                value={opId}
                onChange={(event) => setOpId(event.target.value)}
                size="small"
              />
              <TextField
                label="Site"
                value={site}
                onChange={(event) => setSite(event.target.value)}
                size="small"
              />
              <TextField
                label="Worker ID"
                value={workerId}
                onChange={(event) => setWorkerId(event.target.value)}
                size="small"
              />
              <TextField
                select
                label="Severity"
                value={severity}
                onChange={(event) => setSeverity(event.target.value === 'all' ? 'all' : Number(event.target.value))}
                size="small"
              >
                <MenuItem value="all">All severities</MenuItem>
                <MenuItem value={RuntimeEventSeverity.DEBUG}>Debug</MenuItem>
                <MenuItem value={RuntimeEventSeverity.INFO}>Info</MenuItem>
                <MenuItem value={RuntimeEventSeverity.WARN}>Warn</MenuItem>
                <MenuItem value={RuntimeEventSeverity.ERROR}>Error</MenuItem>
              </TextField>
              <TextField
                select
                label="Source"
                value={source}
                onChange={(event) => setSource(event.target.value === 'all' ? 'all' : Number(event.target.value))}
                size="small"
              >
                <MenuItem value="all">All sources</MenuItem>
                <MenuItem value={RuntimeEventSource.SCHEDULER}>Scheduler</MenuItem>
                <MenuItem value={RuntimeEventSource.WORKER}>Worker</MenuItem>
                <MenuItem value={RuntimeEventSource.RUNNER}>Runner</MenuItem>
                <MenuItem value={RuntimeEventSource.SERVER}>Server</MenuItem>
                <MenuItem value={RuntimeEventSource.SUBMISSION}>Submission</MenuItem>
                <MenuItem value={RuntimeEventSource.REQUEST}>Request</MenuItem>
              </TextField>
            </Box>
          </Stack>
        </CardContent>
      </Card>

      <Card>
        <CardContent>
          <RuntimeEventList
            events={events}
            loading={isLoadingHistory}
            onWorkflowClick={(selectedWorkflowId) => navigate(`/workflows/${selectedWorkflowId}`)}
            emptyMessage="No runtime events matched the current filters."
          />
        </CardContent>
      </Card>
    </Box>
  );
}
