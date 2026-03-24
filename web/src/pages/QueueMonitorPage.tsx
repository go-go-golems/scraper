import { Box, Card, CardContent, Typography } from '@mui/material';
import { QueueStatusTable } from '../components/queues/QueueStatusTable';
import { ThroughputChart } from '../components/queues/ThroughputChart';
import { useListQueuesQuery } from '../api/queueApi';
import type { ThroughputSeries } from '../components/queues/ThroughputChart';

// Static placeholder data until we wire real throughput metrics
const placeholderThroughput: ThroughputSeries[] = [
  {
    queueKey: 'site:hn:http',
    points: Array.from({ length: 15 }, (_, i) => ({
      time: `${String(14 - i).padStart(2, '0')}:00`,
      opsPerMin: Math.floor(10 + Math.random() * 12),
    })).reverse(),
  },
  {
    queueKey: 'site:hn:js',
    points: Array.from({ length: 15 }, (_, i) => ({
      time: `${String(14 - i).padStart(2, '0')}:00`,
      opsPerMin: Math.floor(4 + Math.random() * 8),
    })).reverse(),
  },
];

export function QueueMonitorPage() {
  const { data: queues, isLoading } = useListQueuesQuery(undefined, {
    pollingInterval: 5000,
  });

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 3 }}>
      <Typography variant="h5" component="h1">
        Queue Monitor
      </Typography>

      <Card>
        <CardContent>
          <Typography variant="body2" color="text.secondary" gutterBottom>
            Queue Status
          </Typography>
          <QueueStatusTable queues={queues ?? []} loading={isLoading} />
        </CardContent>
      </Card>

      <Card>
        <CardContent>
          <ThroughputChart data={placeholderThroughput} timeRange="15m" />
        </CardContent>
      </Card>
    </Box>
  );
}
