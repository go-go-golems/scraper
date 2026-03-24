import { Box, Card, CardContent, Typography, Tooltip } from '@mui/material';
import type { OpStatus } from '../../api/types';

interface OpStatusBreakdownProps {
  counts: Partial<Record<OpStatus, number>>;
}

const statusConfig: { key: OpStatus; label: string; color: string }[] = [
  { key: 'pending', label: 'Pending', color: '#bdbdbd' },
  { key: 'ready', label: 'Ready', color: '#42a5f5' },
  { key: 'running', label: 'Running', color: '#1976d2' },
  { key: 'succeeded', label: 'Succeeded', color: '#66bb6a' },
  { key: 'failed', label: 'Failed', color: '#ef5350' },
  { key: 'canceled', label: 'Canceled', color: '#ffa726' },
];

export function OpStatusBreakdown({ counts }: OpStatusBreakdownProps) {
  const total = Object.values(counts).reduce((a, b) => a + (b ?? 0), 0);
  if (total === 0) return null;

  return (
    <Card>
      <CardContent>
        <Typography variant="body2" color="text.secondary" gutterBottom>
          Op Status Breakdown
        </Typography>
        <Box sx={{ display: 'flex', height: 24, borderRadius: 1, overflow: 'hidden', mt: 1 }}>
          {statusConfig.map(({ key, label, color }) => {
            const count = counts[key] ?? 0;
            if (count === 0) return null;
            const pct = (count / total) * 100;
            return (
              <Tooltip key={key} title={`${label}: ${count} (${pct.toFixed(1)}%)`}>
                <Box sx={{ width: `${pct}%`, bgcolor: color, minWidth: count > 0 ? 4 : 0 }} />
              </Tooltip>
            );
          })}
        </Box>
        <Box sx={{ display: 'flex', gap: 2, mt: 1.5, flexWrap: 'wrap' }}>
          {statusConfig.map(({ key, label, color }) => {
            const count = counts[key] ?? 0;
            if (count === 0) return null;
            return (
              <Box key={key} sx={{ display: 'flex', alignItems: 'center', gap: 0.5 }}>
                <Box sx={{ width: 10, height: 10, borderRadius: '50%', bgcolor: color }} />
                <Typography variant="caption">
                  {label}: {count}
                </Typography>
              </Box>
            );
          })}
        </Box>
      </CardContent>
    </Card>
  );
}
