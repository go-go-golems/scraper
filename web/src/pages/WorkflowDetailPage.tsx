import { useState, useCallback, useEffect, useMemo } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { Box, Card, CardContent, IconButton, Tab, Tabs, Typography } from '@mui/material';
import ArrowBackIcon from '@mui/icons-material/ArrowBack';
import { WorkflowHeader } from '../components/workflows/WorkflowHeader';
import { WorkflowProgressBar } from '../components/workflows/WorkflowProgressBar';
import { OpTable } from '../components/workflows/OpTable';
import { OpDetailDrawer } from '../components/workflows/OpDetailDrawer';
import { RuntimeEventTable } from '../components/workflows/RuntimeEventTable';
import { CancelWorkflowButton } from '../components/workflows/CancelWorkflowButton';
import { ArtifactsPanel } from '../components/artifacts/ArtifactsPanel';
import { ResultsPanel } from '../components/results/ResultsPanel';
import {
  useGetWorkflowQuery,
  useGetWorkflowOpsQuery,
  useGetOpResultQuery,
  useGetOpArtifactsQuery,
  useRetryOpMutation,
  useCancelWorkflowMutation,
} from '../api/workflowApi';
import { useGetScriptQuery } from '../api/catalogApi';
import { decodeRuntimeEvent, useGetRecentRuntimeEventsQuery } from '../api/runtimeEventsApi';
import { useToast } from '../components/common/ToastProvider';

export function WorkflowDetailPage() {
  const { workflowId } = useParams<{ workflowId: string }>();
  const navigate = useNavigate();
  const [selectedOpId, setSelectedOpId] = useState<string | null>(null);
  const [drawerOpen, setDrawerOpen] = useState(false);
  const [artifactBodies, setArtifactBodies] = useState<Record<string, string>>({});
  const [activeTab, setActiveTab] = useState<'ops' | 'artifacts' | 'results'>('ops');
  // Bridge: when navigating from OpResultTab → artifact/results browser, pre-fill the opId filter.
  const [artifactFilterOpId, setArtifactFilterOpId] = useState<string | null>(null);
  const [resultFilterOpId, setResultFilterOpId] = useState<string | null>(null);

  const { data: workflow, isLoading: workflowLoading } = useGetWorkflowQuery(workflowId!, {
    skip: !workflowId,
    pollingInterval: 3000,
  });

  const { data: ops } = useGetWorkflowOpsQuery(workflowId!, {
    skip: !workflowId,
    pollingInterval: 3000,
  });

  const selectedOp = ops?.find((o) => o.op.ID === selectedOpId) ?? null;

  const { data: opResult } = useGetOpResultQuery(
    { workflowId: workflowId!, opId: selectedOpId ?? '' },
    { skip: !workflowId || !selectedOpId },
  );

  const { data: artifacts } = useGetOpArtifactsQuery(
    { wfId: workflowId!, opId: selectedOpId ?? '' },
    { skip: !workflowId || !selectedOpId },
  );

  const scriptPath = selectedOp?.op.Metadata?.script;
  const siteName = selectedOp?.op.Site;
  const { data: scriptData, isLoading: scriptLoading } = useGetScriptQuery(
    { site: siteName ?? '', path: scriptPath ?? '' },
    { skip: !siteName || !scriptPath },
  );
  const { data: rawRuntimeEvents = [], isLoading: runtimeEventsLoading } = useGetRecentRuntimeEventsQuery(
    { workflowId, limit: 50 },
    { skip: !workflowId },
  );
  const runtimeEvents = useMemo(
    () =>
      rawRuntimeEvents
        .map((event) => {
          try {
            return decodeRuntimeEvent(event);
          } catch {
            return null;
          }
        })
        .filter((event): event is NonNullable<typeof event> => event !== null),
    [rawRuntimeEvents],
  );

  const [retryOp, { isLoading: retryLoading }] = useRetryOpMutation();
  const [cancelWorkflow, { isLoading: cancelLoading }] = useCancelWorkflowMutation();
  const { showToast } = useToast();

  useEffect(() => {
    if (!artifacts || artifacts.length === 0) return;
    for (const a of artifacts) {
      if (artifactBodies[a.id]) continue;
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

  const handleRetryOp = useCallback(async () => {
    if (!selectedOpId || !workflowId) return;
    try {
      await retryOp({ wfId: workflowId, opId: selectedOpId }).unwrap();
      showToast('Op retry initiated', 'success');
    } catch {
      showToast('Failed to retry op', 'error');
    }
  }, [workflowId, selectedOpId, retryOp, showToast]);

  const handleCancelWorkflow = useCallback(async () => {
    if (!workflowId) return;
    try {
      await cancelWorkflow(workflowId).unwrap();
      showToast('Workflow canceled', 'info');
    } catch {
      showToast('Failed to cancel workflow', 'error');
    }
  }, [workflowId, cancelWorkflow, showToast]);

  const handleBrowseArtifacts = useCallback((opId: string) => {
    setArtifactFilterOpId(opId);
    setActiveTab('artifacts');
  }, []);

  if (!workflowId) {
    return <Typography color="text.disabled">No workflow ID</Typography>;
  }

  if (workflowLoading) {
    return <Typography color="text.secondary">Loading workflow...</Typography>;
  }

  if (!workflow) {
    return <Typography color="text.disabled">Workflow not found</Typography>;
  }

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
      <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
        <IconButton onClick={() => navigate('/workflows')} size="small">
          <ArrowBackIcon />
        </IconButton>
        <Typography variant="body2" color="text.secondary">
          Back to Workflows
        </Typography>
      </Box>

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
        <CardContent>
          <Typography variant="subtitle2" sx={{ mb: 1.5 }}>
            Runtime Events
          </Typography>
          <RuntimeEventTable
            events={runtimeEvents}
            loading={runtimeEventsLoading}
            onWorkflowClick={(id) => navigate(`/workflows/${id}`)}
            emptyMessage="No runtime events for this workflow."
          />
        </CardContent>
      </Card>

      {/* Ops / Results / Artifacts tab */}
      <Card>
        <Box sx={{ borderBottom: 1, borderColor: 'divider', px: 1 }}>
          <Tabs
            value={activeTab}
            onChange={(_e, v) => {
              setActiveTab(v);
              if (v === 'ops') { setArtifactFilterOpId(null); setResultFilterOpId(null); }
            }}
            sx={{ minHeight: 40 }}
          >
            <Tab value="ops" label="Ops" sx={{ minHeight: 40 }} />
            <Tab value="results" label="Results" sx={{ minHeight: 40 }} />
            <Tab value="artifacts" label="Artifacts" sx={{ minHeight: 40 }} />
          </Tabs>
        </Box>

        {activeTab === 'ops' && (
          <CardContent sx={{ p: 0, '&:last-child': { pb: 0 } }}>
            <OpTable
              ops={ops ?? []}
              selectedOpId={selectedOpId}
              onSelectOp={handleSelectOp}
            />
          </CardContent>
        )}

        {activeTab === 'results' && (
          <CardContent>
            <ResultsPanel
              workflowId={workflowId}
              initialOpIdFilter={resultFilterOpId ?? undefined}
              onOpClick={(opId) => {
                setSelectedOpId(opId);
                setDrawerOpen(true);
              }}
            />
          </CardContent>
        )}

        {activeTab === 'artifacts' && (
          <CardContent>
            <ArtifactsPanel
              workflowId={workflowId}
              initialOpIdFilter={artifactFilterOpId ?? undefined}
              onOpClick={(opId) => {
                setSelectedOpId(opId);
                setDrawerOpen(true);
                setArtifactBodies({});
              }}
            />
          </CardContent>
        )}
      </Card>

      {/* OpDetailDrawer — shown when an op is selected, regardless of activeTab */}
      {selectedOpId && (
        <OpDetailDrawer
          op={selectedOp}
          result={opResult ?? null}
          artifacts={artifacts}
          artifactBodies={artifactBodies}
          scriptSource={scriptData?.source}
          scriptLoading={scriptLoading}
          site={siteName ?? ''}
          scriptPath={scriptPath ?? ''}
          open={drawerOpen}
          onClose={handleCloseDrawer}
          onRetry={handleRetryOp}
          retryLoading={retryLoading}
          onBrowseArtifacts={handleBrowseArtifacts}
        />
      )}
    </Box>
  );
}
