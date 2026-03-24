import { Box, Card, CardContent, Chip, Typography } from '@mui/material';
import type { WorkflowSummary } from '../../api/types';
import { StatusChip } from '../common/StatusChip';

interface WorkflowHeaderProps {
  workflow: WorkflowSummary;
}

function formatTimestamp(dateStr: string): string {
  return new Date(dateStr).toLocaleString();
}

export function WorkflowHeader({ workflow }: WorkflowHeaderProps) {
  const { workflow: wf, stats } = workflow;

  return (
    <Card>
      <CardContent>
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5, mb: 1.5 }}>
          <Typography
            variant="h6"
            sx={{ fontFamily: 'monospace', fontSize: '1rem' }}
          >
            {wf.ID}
          </Typography>
          <Chip label={wf.Site} size="small" variant="outlined" />
          <StatusChip status={wf.Status} />
        </Box>

        <Typography variant="body1" gutterBottom>
          {wf.Name}
        </Typography>

        <Box sx={{ display: 'flex', gap: 3, mt: 1 }}>
          <Typography variant="caption" color="text.secondary">
            Created: {formatTimestamp(wf.CreatedAt)}
          </Typography>
          <Typography variant="caption" color="text.secondary">
            Updated: {formatTimestamp(wf.UpdatedAt)}
          </Typography>
          <Typography variant="caption" color="text.secondary">
            Ops: {stats.Total}
          </Typography>
        </Box>
      </CardContent>
    </Card>
  );
}
