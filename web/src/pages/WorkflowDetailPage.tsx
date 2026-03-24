import { useState, useCallback } from 'react';
import { Box, Card, CardContent, Typography } from '@mui/material';
import { WorkflowHeader } from '../components/workflows/WorkflowHeader';
import { WorkflowProgressBar } from '../components/workflows/WorkflowProgressBar';
import { OpTable } from '../components/workflows/OpTable';
import { OpDetailDrawer } from '../components/workflows/OpDetailDrawer';
import {
  useGetWorkflowQuery,
  useGetWorkflowOpsQuery,
  useGetOpResultQuery,
} from '../api/workflowApi';

interface WorkflowDetailPageProps {
  workflowId: string;
}

export function WorkflowDetailPage({ workflowId }: WorkflowDetailPageProps) {
  const [selectedOpId, setSelectedOpId] = useState<string | null>(null);
  const [drawerOpen, setDrawerOpen] = useState(false);

  const { data: workflow, isLoading: workflowLoading } = useGetWorkflowQuery(workflowId, {
    pollingInterval: 3000,
  });

  const { data: ops } = useGetWorkflowOpsQuery(workflowId, {
    pollingInterval: 3000,
  });

  const selectedOp = ops?.find((o) => o.op.ID === selectedOpId) ?? null;

  const { data: opResult } = useGetOpResultQuery(
    { workflowId, opId: selectedOpId ?? '' },
    { skip: !selectedOpId },
  );

  const handleSelectOp = useCallback((id: string) => {
    setSelectedOpId(id);
    setDrawerOpen(true);
  }, []);

  const handleCloseDrawer = useCallback(() => {
    setDrawerOpen(false);
  }, []);

  if (workflowLoading) {
    return (
      <Typography variant="body2" color="text.secondary">
        Loading workflow...
      </Typography>
    );
  }

  if (!workflow) {
    return (
      <Typography variant="body2" color="text.disabled">
        Workflow not found
      </Typography>
    );
  }

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
      <WorkflowHeader workflow={workflow} />

      <Card>
        <CardContent>
          <WorkflowProgressBar stats={workflow.stats} />
        </CardContent>
      </Card>

      <Card>
        <CardContent sx={{ p: 0, '&:last-child': { pb: 0 } }}>
          <OpTable
            ops={ops ?? []}
            selectedOpId={selectedOpId}
            onSelectOp={handleSelectOp}
          />
        </CardContent>
      </Card>

      <OpDetailDrawer
        op={selectedOp}
        result={opResult ?? null}
        open={drawerOpen}
        onClose={handleCloseDrawer}
      />
    </Box>
  );
}
