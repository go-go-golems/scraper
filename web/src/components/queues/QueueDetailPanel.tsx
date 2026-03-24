import { Box, Card, CardContent, Typography } from '@mui/material';
import type { QueueStatus } from '../../api/types';
import { TokenBucketGauge } from './TokenBucketGauge';

interface QueueDetailPanelProps {
  queue: QueueStatus;
}

export function QueueDetailPanel({ queue }: QueueDetailPanelProps) {
  const hasRateLimit =
    queue.burst !== undefined && queue.burst > 0 && queue.ratePerSecond !== undefined;

  const policyParts: string[] = [`Max ${queue.maxInFlight} in-flight`];
  if (hasRateLimit) {
    policyParts.push(
      `token bucket at ${queue.ratePerSecond}/sec burst ${queue.burst}`,
    );
  }

  return (
    <Card variant="outlined">
      <CardContent>
        <Typography variant="subtitle2" gutterBottom>
          {queue.queue}
        </Typography>
        <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
          {policyParts.join(', ')}
        </Typography>

        {hasRateLimit && (
          <Box sx={{ mb: 2 }}>
            <TokenBucketGauge
              tokens={queue.tokens ?? 0}
              burst={queue.burst!}
              ratePerSecond={queue.ratePerSecond!}
            />
          </Box>
        )}

        <Typography variant="caption" color="text.secondary" component="div">
          Op status breakdown
        </Typography>
        <Box sx={{ display: 'flex', gap: 2, mt: 0.5 }}>
          <Typography variant="body2">Pending: {queue.pending}</Typography>
          <Typography variant="body2">Ready: {queue.ready}</Typography>
          <Typography variant="body2">Running: {queue.running}</Typography>
          <Typography variant="body2">Succeeded: {queue.succeeded}</Typography>
          <Typography variant="body2">Failed: {queue.failed}</Typography>
        </Box>
      </CardContent>
    </Card>
  );
}
