import { Card, CardContent, Typography, Box, LinearProgress } from '@mui/material';
import type { QueueStatus } from '../../api/types';

interface QueueHealthPreviewProps {
  queues: QueueStatus[];
  maxVisible?: number;
}

export function QueueHealthPreview({ queues, maxVisible = 6 }: QueueHealthPreviewProps) {
  const visible = queues.slice(0, maxVisible);

  return (
    <Card>
      <CardContent>
        <Typography variant="body2" color="text.secondary" gutterBottom>
          Queue Health
        </Typography>
        {visible.length === 0 && (
          <Typography variant="body2" color="text.disabled" sx={{ mt: 1 }}>
            No active queues
          </Typography>
        )}
        <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1.5, mt: 1 }}>
          {visible.map((q) => {
            const utilization = q.maxInFlight > 0 ? (q.inFlight / q.maxInFlight) * 100 : 0;
            const color = utilization >= 90 ? 'error' : utilization >= 50 ? 'warning' : 'primary';
            return (
              <Box key={`${q.site}:${q.queue}`}>
                <Box sx={{ display: 'flex', justifyContent: 'space-between', mb: 0.5 }}>
                  <Typography variant="caption" noWrap sx={{ maxWidth: '70%' }}>
                    {q.queue}
                  </Typography>
                  <Typography variant="caption" color="text.secondary">
                    {q.inFlight}/{q.maxInFlight}
                  </Typography>
                </Box>
                <LinearProgress
                  variant="determinate"
                  value={Math.min(utilization, 100)}
                  color={color}
                  sx={{ height: 6, borderRadius: 1 }}
                />
              </Box>
            );
          })}
        </Box>
      </CardContent>
    </Card>
  );
}
