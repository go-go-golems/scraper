import { useState, useMemo } from 'react';
import { useNavigate } from 'react-router-dom';
import {
  Box,
  Card,
  CardContent,
  Chip,
  Collapse,
  IconButton,
  Stack,
  TextField,
  Typography,
} from '@mui/material';
import PauseIcon from '@mui/icons-material/Pause';
import PlayArrowIcon from '@mui/icons-material/PlayArrow';
import FilterListIcon from '@mui/icons-material/FilterList';
import { RuntimeEventTable } from '../components/workflows/RuntimeEventTable';
import { MultiSelectChipFilter, type MultiSelectOption } from '../components/common/MultiSelectChipFilter';
import { TimeRangeSelector, type TimeRange } from '../components/common/TimeRangeSelector';
import { decodeRuntimeEvent, useGetRecentRuntimeEventsQuery } from '../api/runtimeEventsApi';
import { runtimeEventOccurredAtMillis } from '../api/runtimeEventsApi';
import {
  RuntimeEventSeverity,
  RuntimeEventSource,
} from '../pb/proto/scraper/runtime/v1/events_pb';
import dayjs from 'dayjs';

type ConnectionState = 'connecting' | 'live' | 'error' | 'closed';

function connectionColor(state: ConnectionState): 'default' | 'success' | 'warning' | 'error' {
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

function formatLastEventAt(events: ReturnType<typeof runtimeEventOccurredAtMillis> | null): string {
  if (!events) return 'No events received yet';
  return `Last event ${new Date(events).toLocaleString()}`;
}

const severityOptions: MultiSelectOption[] = [
  { value: String(RuntimeEventSeverity.DEBUG), label: 'Debug', color: 'default' },
  { value: String(RuntimeEventSeverity.INFO), label: 'Info', color: 'info' },
  { value: String(RuntimeEventSeverity.WARN), label: 'Warn', color: 'warning' },
  { value: String(RuntimeEventSeverity.ERROR), label: 'Error', color: 'error' },
];

const sourceOptions: MultiSelectOption[] = [
  { value: String(RuntimeEventSource.SCHEDULER), label: 'Scheduler' },
  { value: String(RuntimeEventSource.WORKER), label: 'Worker' },
  { value: String(RuntimeEventSource.RUNNER), label: 'Runner' },
  { value: String(RuntimeEventSource.SERVER), label: 'Server' },
  { value: String(RuntimeEventSource.SUBMISSION), label: 'Submission' },
  { value: String(RuntimeEventSource.REQUEST), label: 'Request' },
];

export function RuntimeEventsPage() {
  const navigate = useNavigate();
  const [workflowId, setWorkflowId] = useState('');
  const [opId, setOpId] = useState('');
  const [site, setSite] = useState('');
  const [workerId, setWorkerId] = useState('');
  const [selectedSeverities, setSelectedSeverities] = useState<string[]>([]);
  const [selectedSources, setSelectedSources] = useState<string[]>([]);
  const [timeRange, setTimeRange] = useState<TimeRange>({ mode: 'live' });
  const [showAdvanced, setShowAdvanced] = useState(false);
  const [paused, setPaused] = useState(false);

  // Compute since/until from time range for server-side filtering
  const serverSince = useMemo(() => {
    if (timeRange.mode === 'relative' && timeRange.range) {
      const map: Record<string, [number, string]> = {
        '1h': [1, 'hour'],
        '6h': [6, 'hour'],
        '24h': [24, 'hour'],
        '7d': [7, 'day'],
      };
      const [amount, unit] = map[timeRange.range] ?? [1, 'hour'];
      return dayjs().subtract(amount, unit as dayjs.ManipulateType).toISOString();
    }
    if (timeRange.mode === 'absolute' && timeRange.from) {
      return timeRange.from;
    }
    return undefined;
  }, [timeRange]);

  const serverUntil = timeRange.mode === 'absolute' ? timeRange.to : undefined;

  // RTK Query — SSE managed by onCacheEntryAdded, pause via skip
  const { data: rawEvents = [], isLoading, isError, isSuccess } = useGetRecentRuntimeEventsQuery(
    {
      workflowId: workflowId || undefined,
      opId: opId || undefined,
      site: site || undefined,
      workerId: workerId || undefined,
      limit: 100,
      since: serverSince,
      until: serverUntil,
    },
    { skip: paused },
  );

  const events = useMemo(
    () =>
      rawEvents
        .map((event) => {
          try {
            return decodeRuntimeEvent(event);
          } catch {
            return null;
          }
        })
        .filter((event): event is NonNullable<typeof event> => event !== null)
        .sort((a, b) => runtimeEventOccurredAtMillis(b) - runtimeEventOccurredAtMillis(a)),
    [rawEvents],
  );

  const connectionState: ConnectionState =
    paused ? 'closed' :
    isLoading ? 'connecting' :
    isError ? 'error' :
    isSuccess ? 'live' : 'closed';

  const lastEventAt = useMemo(() => {
    if (events.length === 0) return null;
    return runtimeEventOccurredAtMillis(events[0]);
  }, [events]);

  // Client-side filtering by multi-select chips
  const filteredEvents = useMemo(() => {
    return events.filter((event) => {
      if (selectedSeverities.length > 0 && !selectedSeverities.includes(String(event.severity))) {
        return false;
      }
      if (selectedSources.length > 0 && !selectedSources.includes(String(event.source))) {
        return false;
      }
      return true;
    });
  }, [events, selectedSeverities, selectedSources]);

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

            <Stack direction="row" spacing={1} useFlexGap flexWrap="wrap" alignItems="center">
              <Chip label={`Stream: ${connectionState}`} color={connectionColor(connectionState)} />
              <Chip label={`${filteredEvents.length} events`} variant="outlined" />
              <Chip label={formatLastEventAt(lastEventAt)} variant="outlined" />
              <IconButton
                size="small"
                onClick={() => setPaused((p) => !p)}
                title={paused ? 'Resume stream' : 'Pause stream'}
              >
                {paused ? <PlayArrowIcon fontSize="small" /> : <PauseIcon fontSize="small" />}
              </IconButton>
            </Stack>

            <TimeRangeSelector value={timeRange} onChange={setTimeRange} />

            <Stack direction="row" spacing={2} useFlexGap flexWrap="wrap">
              <MultiSelectChipFilter
                label="Severity"
                options={severityOptions}
                selected={selectedSeverities}
                onChange={setSelectedSeverities}
              />
              <MultiSelectChipFilter
                label="Source"
                options={sourceOptions}
                selected={selectedSources}
                onChange={setSelectedSources}
              />
              <IconButton size="small" onClick={() => setShowAdvanced((v) => !v)} title="Advanced filters">
                <FilterListIcon fontSize="small" />
              </IconButton>
            </Stack>

            <Collapse in={showAdvanced}>
              <Box
                sx={{
                  display: 'grid',
                  gap: 1.5,
                  gridTemplateColumns: {
                    xs: '1fr',
                    md: 'repeat(2, minmax(0, 1fr))',
                  },
                  mt: 1,
                }}
              >
                <TextField
                  label="Workflow ID"
                  value={workflowId}
                  onChange={(e) => setWorkflowId(e.target.value)}
                  size="small"
                />
                <TextField
                  label="Op ID"
                  value={opId}
                  onChange={(e) => setOpId(e.target.value)}
                  size="small"
                />
                <TextField
                  label="Site"
                  value={site}
                  onChange={(e) => setSite(e.target.value)}
                  size="small"
                />
                <TextField
                  label="Worker ID"
                  value={workerId}
                  onChange={(e) => setWorkerId(e.target.value)}
                  size="small"
                />
              </Box>
            </Collapse>
          </Stack>
        </CardContent>
      </Card>

      <Card>
        <CardContent sx={{ p: 0, '&:last-child': { pb: 0 } }}>
          <RuntimeEventTable
            events={filteredEvents}
            loading={isLoading}
            showPagination
            onWorkflowClick={(id) => navigate(`/workflows/${id}`)}
            emptyMessage="No runtime events matched the current filters."
          />
        </CardContent>
      </Card>
    </Box>
  );
}
