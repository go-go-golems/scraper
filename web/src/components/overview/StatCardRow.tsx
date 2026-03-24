import { Grid } from '@mui/material';
import { StatCard } from './StatCard';
import type { EngineStatus } from '../../api/types';

interface StatCardRowProps {
  status?: EngineStatus;
  loading?: boolean;
}

export function StatCardRow({ status, loading }: StatCardRowProps) {
  if (loading || !status) {
    return (
      <Grid container spacing={2}>
        {[0, 1, 2, 3].map((i) => (
          <Grid size={{ xs: 12, sm: 6, md: 3 }} key={i}>
            <StatCard title="" value={0} loading />
          </Grid>
        ))}
      </Grid>
    );
  }

  const ops = status.OpCounts;
  const totalOps = Object.values(ops).reduce((a, b) => a + b, 0);

  return (
    <Grid container spacing={2}>
      <Grid size={{ xs: 12, sm: 6, md: 3 }}>
        <StatCard
          title="Workflows"
          value={status.WorkflowCount}
          breakdown={[
            { label: 'running', value: ops.running > 0 ? Math.min(status.WorkflowCount, ops.running) : 0, color: 'info' },
          ]}
        />
      </Grid>
      <Grid size={{ xs: 12, sm: 6, md: 3 }}>
        <StatCard
          title="Operations"
          value={totalOps}
          breakdown={[
            { label: 'ready', value: ops.ready ?? 0, color: 'info' },
            { label: 'running', value: ops.running ?? 0, color: 'primary' },
            { label: 'failed', value: ops.failed ?? 0, color: 'error' },
          ]}
        />
      </Grid>
      <Grid size={{ xs: 12, sm: 6, md: 3 }}>
        <StatCard
          title="Leases"
          value={status.ActiveLeases}
          breakdown={[
            { label: 'expired', value: status.ExpiredLeases, color: status.ExpiredLeases > 0 ? 'warning' : 'default' },
          ]}
        />
      </Grid>
      <Grid size={{ xs: 12, sm: 6, md: 3 }}>
        <StatCard
          title="Artifacts"
          value={status.ArtifactCount}
        />
      </Grid>
    </Grid>
  );
}
