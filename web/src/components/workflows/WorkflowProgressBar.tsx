import { Box, Tooltip, Typography } from '@mui/material';
import type { WorkflowStats } from '../../api/types';

interface WorkflowProgressBarProps {
  stats: WorkflowStats;
}

const segments: { key: keyof WorkflowStats; label: string; color: string }[] = [
  { key: 'Succeeded', label: 'Succeeded', color: '#66bb6a' },
  { key: 'Failed', label: 'Failed', color: '#ef5350' },
  { key: 'Running', label: 'Running', color: '#1976d2' },
  { key: 'Ready', label: 'Ready', color: '#42a5f5' },
  { key: 'Pending', label: 'Pending', color: '#bdbdbd' },
  { key: 'Canceled', label: 'Canceled', color: '#ffa726' },
];

export function WorkflowProgressBar({ stats }: WorkflowProgressBarProps) {
  const total = stats.Total || 1;
  const done = stats.Succeeded + stats.Failed + stats.Canceled;

  return (
    <Box>
      <Box sx={{ display: 'flex', justifyContent: 'space-between', mb: 0.5 }}>
        <Typography variant="body2" color="text.secondary">
          Progress
        </Typography>
        <Typography variant="body2" color="text.secondary">
          {done}/{stats.Total} ops
        </Typography>
      </Box>

      <Box sx={{ display: 'flex', height: 20, borderRadius: 1, overflow: 'hidden' }}>
        {segments.map(({ key, label, color }) => {
          const count = stats[key] as number;
          if (count === 0) return null;
          const pct = (count / total) * 100;
          return (
            <Tooltip key={key} title={`${label}: ${count} (${pct.toFixed(1)}%)`}>
              <Box sx={{ width: `${pct}%`, bgcolor: color, minWidth: count > 0 ? 2 : 0 }} />
            </Tooltip>
          );
        })}
      </Box>

      <Box sx={{ display: 'flex', gap: 2, mt: 1, flexWrap: 'wrap' }}>
        {segments.map(({ key, label, color }) => {
          const count = stats[key] as number;
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
    </Box>
  );
}
