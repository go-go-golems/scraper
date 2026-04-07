import { useState, useCallback } from 'react';
import {
  Box,
  Collapse,
  IconButton,
  KeyboardArrowDownIcon,
  KeyboardArrowUpIcon,
  Skeleton,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TablePagination,
  TableRow,
  Typography,
} from '@mui/material';
import type { RuntimeEventV1 } from '../../pb/proto/scraper/runtime/v1/events_pb';
import {
  RuntimeEventKind,
  RuntimeEventSeverity,
  RuntimeEventSource,
} from '../../pb/proto/scraper/runtime/v1/events_pb';
import { SeverityDotIndicator } from './SeverityDotIndicator';
import { JsonViewer } from '../common/JsonViewer';

// ── helpers ──────────────────────────────────────────────────────────

function formatTimestamp(event: RuntimeEventV1): string {
  if (!event.occurredAt) return '—';
  const millis =
    Number(event.occurredAt.seconds) * 1000 +
    Math.floor(event.occurredAt.nanos / 1_000_000);
  return new Date(millis).toLocaleTimeString();
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

function truncate(str: string, max: number): string {
  return str.length > max ? str.slice(0, max) + '…' : str;
}

function sortEvents(
  events: RuntimeEventV1[],
  field: SortField,
  dir: SortDir,
): RuntimeEventV1[] {
  return [...events].sort((a, b) => {
    let cmp = 0;
    switch (field) {
      case 'timestamp': {
        const aT =
          Number(a.occurredAt?.seconds ?? 0) * 1000 +
          Math.floor((a.occurredAt?.nanos ?? 0) / 1_000_000);
        const bT =
          Number(b.occurredAt?.seconds ?? 0) * 1000 +
          Math.floor((b.occurredAt?.nanos ?? 0) / 1_000_000);
        cmp = aT - bT;
        break;
      }
      case 'severity':
        cmp = a.severity - b.severity;
        break;
      case 'source':
        cmp = (RuntimeEventSource[a.source] ?? '').localeCompare(
          RuntimeEventSource[b.source] ?? '',
        );
        break;
      case 'kind':
        cmp = (RuntimeEventKind[a.kind] ?? '').localeCompare(
          RuntimeEventKind[b.kind] ?? '',
        );
        break;
    }
    return dir === 'desc' ? -cmp : cmp;
  });
}

// ── types ────────────────────────────────────────────────────────────

type SortField = 'timestamp' | 'severity' | 'source' | 'kind';
type SortDir = 'asc' | 'desc';

interface RuntimeEventTableProps {
  events: RuntimeEventV1[];
  loading?: boolean;
  dense?: boolean;
  expandable?: boolean;
  showPagination?: boolean;
  onWorkflowClick?: (workflowId: string) => void;
  onOpClick?: (opId: string) => void;
  emptyMessage?: string;
}

// ── detail row ───────────────────────────────────────────────────────

function EventDetailRow({
  event,
  colSpan,
  onWorkflowClick,
  onOpClick,
}: {
  event: RuntimeEventV1;
  colSpan: number;
  onWorkflowClick?: (id: string) => void;
  onOpClick?: (id: string) => void;
}) {
  return (
    <TableRow>
      <TableCell colSpan={colSpan} sx={{ py: 1.5, px: 2, bgcolor: 'grey.50' }}>
        <Typography variant="body2" sx={{ mb: 1 }}>
          {event.message || 'Runtime event'}
        </Typography>

        <Box sx={{ display: 'flex', gap: 1.5, flexWrap: 'wrap', mb: 1 }}>
          {event.opId && (
            <Typography
              variant="caption"
              sx={{ fontFamily: 'monospace', cursor: onOpClick ? 'pointer' : 'default', color: onOpClick ? 'primary.main' : 'text.secondary' }}
              onClick={onOpClick ? () => onOpClick(event.opId) : undefined}
            >
              Op: {event.opId}
            </Typography>
          )}
          {event.workflowId && (
            <Typography
              variant="caption"
              sx={{ fontFamily: 'monospace', cursor: onWorkflowClick ? 'pointer' : 'default', color: onWorkflowClick ? 'primary.main' : 'text.secondary' }}
              onClick={onWorkflowClick ? () => onWorkflowClick(event.workflowId) : undefined}
            >
              Workflow: {event.workflowId}
            </Typography>
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
        </Box>

        {event.payload &&
          typeof event.payload === 'object' &&
          Object.keys(event.payload as Record<string, unknown>).length > 0 && (
            <Box sx={{ maxWidth: 480 }}>
              <JsonViewer data={event.payload} maxHeight={200} />
            </Box>
          )}
      </TableCell>
    </TableRow>
  );
}

// ── skeleton ─────────────────────────────────────────────────────────

function SkeletonRows() {
  return (
    <>
      {Array.from({ length: 5 }).map((_, i) => (
        <TableRow key={i}>
          <TableCell><Skeleton width={80} /></TableCell>
          <TableCell><Skeleton width={60} /></TableCell>
          <TableCell><Skeleton width={70} /></TableCell>
          <TableCell><Skeleton width={100} /></TableCell>
          <TableCell><Skeleton width={180} /></TableCell>
        </TableRow>
      ))}
    </>
  );
}

// ── main component ───────────────────────────────────────────────────

export function RuntimeEventTable({
  events,
  loading = false,
  dense = true,
  expandable = true,
  showPagination = false,
  onWorkflowClick,
  onOpClick,
  emptyMessage = 'No runtime events.',
}: RuntimeEventTableProps) {
  const [expandedId, setExpandedId] = useState<string | null>(null);
  const [sortField, setSortField] = useState<SortField>('timestamp');
  const [sortDir, setSortDir] = useState<SortDir>('desc');
  const [page, setPage] = useState(0);
  const [rowsPerPage, setRowsPerPage] = useState(25);

  const handleSort = useCallback(
    (field: SortField) => {
      if (field === sortField) {
        setSortDir((d) => (d === 'asc' ? 'desc' : 'asc'));
      } else {
        setSortField(field);
        setSortDir('desc');
      }
    },
    [sortField],
  );

  const handleRowClick = useCallback(
    (id: string) => {
      if (!expandable) return;
      setExpandedId((prev) => (prev === id ? null : id));
    },
    [expandable],
  );

  const sorted = sortEvents(events, sortField, sortDir);
  const paginated = showPagination
    ? sorted.slice(page * rowsPerPage, page * rowsPerPage + rowsPerPage)
    : sorted;

  const colCount = expandable ? 6 : 5;

  const sortArrow = (field: SortField) => {
    if (field !== sortField) return '';
    return sortDir === 'asc' ? ' ↑' : ' ↓';
  };

  const sortableHeader = (field: SortField, label: string) => (
    <TableCell
      sx={{ cursor: 'pointer', userSelect: 'none', whiteSpace: 'nowrap' }}
      onClick={() => handleSort(field)}
    >
      {label}{sortArrow(field)}
    </TableCell>
  );

  return (
    <Box>
      <TableContainer>
        <Table size={dense ? 'small' : 'medium'}>
          <TableHead>
            <TableRow>
              {expandable && <TableCell sx={{ width: 40 }} />}
              {sortableHeader('timestamp', 'Time')}
              {sortableHeader('severity', 'Severity')}
              {sortableHeader('source', 'Source')}
              {sortableHeader('kind', 'Kind')}
              <TableCell>Message</TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {loading && <SkeletonRows />}

            {!loading && events.length === 0 && (
              <TableRow>
                <TableCell colSpan={colCount} align="center">
                  <Typography variant="body2" color="text.disabled" sx={{ py: 4 }}>
                    {emptyMessage}
                  </Typography>
                </TableCell>
              </TableRow>
            )}

            {!loading &&
              paginated.map((event, index) => {
                const id = event.id || `${event.kind}-${index}`;
                const isExpanded = expandedId === id;
                return (
                  <Box component="tbody" key={id}>
                    <TableRow
                      hover
                      sx={{ cursor: expandable ? 'pointer' : 'default' }}
                      onClick={() => handleRowClick(id)}
                    >
                      {expandable && (
                        <TableCell>
                          <IconButton size="small" disableRipple>
                            {isExpanded ? (
                              <KeyboardArrowDownIcon fontSize="small" />
                            ) : (
                              <KeyboardArrowUpIcon fontSize="small" />
                            )}
                          </IconButton>
                        </TableCell>
                      )}
                      <TableCell>
                        <Typography variant="caption" color="text.secondary" noWrap>
                          {formatTimestamp(event)}
                        </Typography>
                      </TableCell>
                      <TableCell>
                        <SeverityDotIndicator severity={event.severity} />
                      </TableCell>
                      <TableCell>
                        <Typography variant="caption">
                          {RuntimeEventSource[event.source] ?? 'Unknown'}
                        </Typography>
                      </TableCell>
                      <TableCell>
                        <Typography variant="caption">
                          {normalizeEnumLabel(RuntimeEventKind[event.kind])}
                        </Typography>
                      </TableCell>
                      <TableCell>
                        <Typography variant="caption" noWrap>
                          {truncate(event.message || '—', 80)}
                        </Typography>
                      </TableCell>
                    </TableRow>
                    {expandable && isExpanded && (
                      <EventDetailRow
                        event={event}
                        colSpan={colCount}
                        onWorkflowClick={onWorkflowClick}
                        onOpClick={onOpClick}
                      />
                    )}
                  </Box>
                );
              })}
          </TableBody>
        </Table>
      </TableContainer>

      {showPagination && (
        <TablePagination
          component="div"
          count={events.length}
          page={page}
          onPageChange={(_, p) => setPage(p)}
          rowsPerPage={rowsPerPage}
          onRowsPerPageChange={(e) => {
            setRowsPerPage(parseInt(e.target.value, 10));
            setPage(0);
          }}
          rowsPerPageOptions={[25, 50, 100]}
          size="small"
        />
      )}
    </Box>
  );
}
