import {
  Box,
  Chip,
  LinearProgress,
  Skeleton,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Typography,
} from '@mui/material';
import type { QueueStatus } from '../../api/types';

interface QueueStatusTableProps {
  queues: QueueStatus[];
  loading: boolean;
}

function utilizationColor(inFlight: number, maxInFlight: number): 'success' | 'warning' | 'error' {
  if (maxInFlight <= 0) return 'success';
  const pct = (inFlight / maxInFlight) * 100;
  if (pct >= 90) return 'error';
  if (pct >= 50) return 'warning';
  return 'success';
}

function SkeletonRows() {
  return (
    <>
      {Array.from({ length: 5 }).map((_, i) => (
        <TableRow key={i}>
          <TableCell><Skeleton width={160} /></TableCell>
          <TableCell><Skeleton width={80} /></TableCell>
          <TableCell><Skeleton width={140} /></TableCell>
          <TableCell><Skeleton width={40} /></TableCell>
          <TableCell><Skeleton width={40} /></TableCell>
          <TableCell><Skeleton width={40} /></TableCell>
          <TableCell><Skeleton width={40} /></TableCell>
        </TableRow>
      ))}
    </>
  );
}

export function QueueStatusTable({ queues, loading }: QueueStatusTableProps) {
  return (
    <TableContainer>
      <Table size="small">
        <TableHead>
          <TableRow>
            <TableCell>Queue Key</TableCell>
            <TableCell>Site</TableCell>
            <TableCell>In-Flight</TableCell>
            <TableCell align="right">Max In-Flight</TableCell>
            <TableCell align="right">Tokens</TableCell>
            <TableCell align="right">Rate/sec</TableCell>
            <TableCell align="right">Burst</TableCell>
          </TableRow>
        </TableHead>
        <TableBody>
          {loading && <SkeletonRows />}
          {!loading && queues.length === 0 && (
            <TableRow>
              <TableCell colSpan={7} align="center">
                <Typography variant="body2" color="text.disabled" sx={{ py: 4 }}>
                  No queues found
                </Typography>
              </TableCell>
            </TableRow>
          )}
          {!loading &&
            queues.map((q) => {
              const pct = q.maxInFlight > 0 ? (q.inFlight / q.maxInFlight) * 100 : 0;
              const color = utilizationColor(q.inFlight, q.maxInFlight);
              return (
                <TableRow key={`${q.site}:${q.queue}`}>
                  <TableCell>
                    <Typography
                      variant="body2"
                      sx={{ fontFamily: 'monospace', fontSize: '0.8rem' }}
                    >
                      {q.queue}
                    </Typography>
                  </TableCell>
                  <TableCell>
                    <Chip label={q.site} size="small" variant="outlined" />
                  </TableCell>
                  <TableCell>
                    <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, minWidth: 160 }}>
                      <LinearProgress
                        variant="determinate"
                        value={Math.min(pct, 100)}
                        color={color}
                        sx={{ flexGrow: 1, height: 6, borderRadius: 1 }}
                      />
                      <Typography variant="caption" color="text.secondary" noWrap>
                        {q.inFlight}/{q.maxInFlight}
                      </Typography>
                    </Box>
                  </TableCell>
                  <TableCell align="right">
                    {q.maxInFlight === 1 ? (
                      <Typography component="span" variant="body2" color="text.disabled">
                        1 (default)
                      </Typography>
                    ) : (
                      q.maxInFlight
                    )}
                  </TableCell>
                  <TableCell align="right">
                    {q.tokens != null ? q.tokens : (
                      <Typography component="span" variant="body2" color="text.disabled">
                        none
                      </Typography>
                    )}
                  </TableCell>
                  <TableCell align="right">
                    {q.ratePerSecond != null ? q.ratePerSecond : (
                      <Typography component="span" variant="body2" color="text.disabled">
                        none
                      </Typography>
                    )}
                  </TableCell>
                  <TableCell align="right">
                    {q.burst != null ? q.burst : (
                      <Typography component="span" variant="body2" color="text.disabled">
                        none
                      </Typography>
                    )}
                  </TableCell>
                </TableRow>
              );
            })}
        </TableBody>
      </Table>
    </TableContainer>
  );
}
