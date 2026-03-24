import { useState, useCallback } from 'react';
import { Box, Card, CardContent, Collapse, Typography } from '@mui/material';
import { QueueStatusTable } from '../components/queues/QueueStatusTable';
import { QueueDetailPanel } from '../components/queues/QueueDetailPanel';
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
  const [expandedQueue, setExpandedQueue] = useState<string | null>(null);

  const handleToggleExpand = useCallback((queueKey: string) => {
    setExpandedQueue((prev) => (prev === queueKey ? null : queueKey));
  }, []);

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

          {/* Expandable detail panels below the table */}
          {(queues ?? []).map((q) => {
            const key = `${q.site}:${q.queue}`;
            const isExpanded = expandedQueue === key;
            return (
              <Box key={key}>
                <Typography
                  variant="caption"
                  sx={{
                    cursor: 'pointer',
                    color: 'primary.main',
                    display: 'block',
                    mt: 0.5,
                    '&:hover': { textDecoration: 'underline' },
                  }}
                  onClick={() => handleToggleExpand(key)}
                >
                  {isExpanded ? '▾' : '▸'} {q.queue}
                </Typography>
                <Collapse in={isExpanded}>
                  <Box sx={{ mt: 1, mb: 2 }}>
                    <QueueDetailPanel queue={q} />
                  </Box>
                </Collapse>
              </Box>
            );
          })}
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
