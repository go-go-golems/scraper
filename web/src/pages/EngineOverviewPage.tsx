import { Grid, Box } from '@mui/material';
import { StatCardRow } from '../components/overview/StatCardRow';
import { OpStatusBreakdown } from '../components/overview/OpStatusBreakdown';
import { QueueHealthPreview } from '../components/overview/QueueHealthPreview';
import { useGetEngineStatusQuery } from '../api/engineApi';
import { useListQueuesQuery } from '../api/queueApi';

export function EngineOverviewPage() {
  const { data: status, isLoading: statusLoading } = useGetEngineStatusQuery(undefined, {
    pollingInterval: 5000,
  });
  const { data: queues } = useListQueuesQuery(undefined, {
    pollingInterval: 5000,
  });

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 3 }}>
      <StatCardRow status={status} loading={statusLoading} />
      <Grid container spacing={3}>
        <Grid size={{ xs: 12, md: 7 }}>
          {status && <OpStatusBreakdown counts={status.OpCounts} />}
        </Grid>
        <Grid size={{ xs: 12, md: 5 }}>
          <QueueHealthPreview queues={queues ?? []} />
        </Grid>
      </Grid>
    </Box>
  );
}
