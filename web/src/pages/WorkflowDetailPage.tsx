import { useState, useCallback, useEffect } from 'react';
import { Box, Card, CardContent, Typography } from '@mui/material';
import { WorkflowHeader } from '../components/workflows/WorkflowHeader';
import { WorkflowProgressBar } from '../components/workflows/WorkflowProgressBar';
import { OpTable } from '../components/workflows/OpTable';
import { OpDetailDrawer } from '../components/workflows/OpDetailDrawer';
import { CancelWorkflowButton } from '../components/workflows/CancelWorkflowButton';
import {
  useGetWorkflowQuery,
  useGetWorkflowOpsQuery,
  useGetOpResultQuery,
  useGetOpArtifactsQuery,
  useRetryOpMutation,
  useCancelWorkflowMutation,
} from '../api/workflowApi';
import { useGetScriptQuery } from '../api/catalogApi';

interface WorkflowDetailPageProps {
  workflowId: string;
}

export function WorkflowDetailPage({ workflowId }: WorkflowDetailPageProps) {
  const [selectedOpId, setSelectedOpId] = useState<string | null>(null);
  const [drawerOpen, setDrawerOpen] = useState(false);
  const [artifactBodies, setArtifactBodies] = useState<Record<string, string>>({});

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

  const { data: artifacts } = useGetOpArtifactsQuery(
    { wfId: workflowId, opId: selectedOpId ?? '' },
    { skip: !selectedOpId },
  );

  // Fetch script source for the selected op
  const scriptPath = selectedOp?.op.Metadata?.script;
  const siteName = selectedOp?.op.Site;
  const { data: scriptData, isLoading: scriptLoading } = useGetScriptQuery(
    { site: siteName ?? '', path: scriptPath ?? '' },
    { skip: !siteName || !scriptPath },
  );

  const [retryOp, { isLoading: retryLoading }] = useRetryOpMutation();
  const [cancelWorkflow, { isLoading: cancelLoading }] = useCancelWorkflowMutation();

  // Fetch artifact bodies on demand
  useEffect(() => {
    if (!artifacts || artifacts.length === 0) return;
    for (const a of artifacts) {
      if (artifactBodies[a.id]) continue;
      // Only fetch text-based artifacts inline
      if (
        a.contentType.startsWith('text/') ||
        a.contentType === 'application/json' ||
        a.kind === 'execution-log'
      ) {
        fetch(`/api/v1/artifacts/${a.id}`)
          .then((r) => r.text())
          .then((body) => {
            setArtifactBodies((prev) => ({ ...prev, [a.id]: body }));
          })
          .catch(() => {});
      }
    }
  }, [artifacts, artifactBodies]);

  const handleSelectOp = useCallback((id: string) => {
    setSelectedOpId(id);
    setDrawerOpen(true);
    setArtifactBodies({});
  }, []);

  const handleCloseDrawer = useCallback(() => {
    setDrawerOpen(false);
  }, []);

  const handleRetryOp = useCallback(() => {
    if (!selectedOpId) return;
    retryOp({ wfId: workflowId, opId: selectedOpId });
  }, [workflowId, selectedOpId, retryOp]);

  const handleCancelWorkflow = useCallback(() => {
    cancelWorkflow(workflowId);
  }, [workflowId, cancelWorkflow]);

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
      <Box sx={{ display: 'flex', alignItems: 'flex-start', gap: 2 }}>
        <Box sx={{ flexGrow: 1 }}>
          <WorkflowHeader workflow={workflow} />
        </Box>
        <CancelWorkflowButton
          workflowId={workflowId}
          status={workflow.workflow.Status}
          onCancel={handleCancelWorkflow}
          loading={cancelLoading}
        />
      </Box>

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
        artifacts={artifacts}
        artifactBodies={artifactBodies}
        scriptSource={scriptData?.source}
        scriptLoading={scriptLoading}
        open={drawerOpen}
        onClose={handleCloseDrawer}
        onRetry={handleRetryOp}
        retryLoading={retryLoading}
      />
    </Box>
  );
}
