import { useState, useCallback, useEffect, useMemo } from 'react';
import {
  Badge,
  Box,
  Divider,
  Drawer,
  IconButton,
  Tab,
  Tabs,
  Typography,
} from '@mui/material';
import CloseIcon from '@mui/icons-material/Close';
import CircleIcon from '@mui/icons-material/Circle';
import type { WorkflowOp, OpResult, ArtifactSummary } from '../../api/types';
import { StatusChip } from '../common/StatusChip';
import { RetryOpButton } from './RetryOpButton';
import { decodeRuntimeEvent, useGetRecentRuntimeEventsQuery } from '../../api/runtimeEventsApi';
import { KindIcon, connectionColor, type ConnectionState } from './op-detail/helpers';
import { OpInputTab } from './op-detail/OpInputTab';
import { OpDepsTab } from './op-detail/OpDepsTab';
import { OpResultTab } from './op-detail/OpResultTab';
import { OpArtifactsTab } from './op-detail/OpArtifactsTab';
import { OpRuntimeTab } from './op-detail/OpRuntimeTab';
import { OpScriptTab } from './op-detail/OpScriptTab';
import { OpLogsTab } from './op-detail/OpLogsTab';

interface OpDetailDrawerProps {
  op: WorkflowOp | null;
  result: OpResult | null;
  artifacts?: ArtifactSummary[];
  artifactBodies?: Record<string, string>;
  scriptSource?: string | null;
  scriptLoading?: boolean;
  site?: string;
  scriptPath?: string;
  open: boolean;
  onClose: () => void;
  onRetry?: () => void;
  retryLoading?: boolean;
  onBrowseArtifacts?: (opId: string) => void;
}

type TabId = 'input' | 'deps' | 'result' | 'artifacts' | 'runtime' | 'script' | 'logs';

export function OpDetailDrawer({
  op,
  result,
  artifacts = [],
  artifactBodies = {},
  scriptSource,
  scriptLoading,
  site = '',
  scriptPath = '',
  open,
  onClose,
  onRetry,
  retryLoading,
  onBrowseArtifacts,
}: OpDetailDrawerProps) {
  const [activeTab, setActiveTab] = useState<TabId>('input');
  const [selectedArtifactId, setSelectedArtifactId] = useState<string | null>(null);
  const selectedSpec = op?.op;
  const runtimeTabActive = open && activeTab === 'runtime' && Boolean(selectedSpec);
  const {
    data: rawOpRuntimeEvents = [],
    isLoading: opRuntimeEventsLoading,
    isError: opRuntimeEventsError,
    isSuccess: opRuntimeEventsSuccess,
  } = useGetRecentRuntimeEventsQuery(
    {
      workflowId: selectedSpec?.WorkflowID,
      opId: selectedSpec?.ID,
      limit: 40,
    },
    { skip: !runtimeTabActive },
  );
  const opRuntimeEvents = useMemo(
    () =>
      rawOpRuntimeEvents
        .map((event) => {
          try {
            return decodeRuntimeEvent(event);
          } catch {
            return null;
          }
        })
        .filter((event): event is NonNullable<typeof event> => event !== null),
    [rawOpRuntimeEvents],
  );

  const runtimeConnectionState: ConnectionState =
    !runtimeTabActive ? 'closed' :
    opRuntimeEventsLoading ? 'connecting' :
    opRuntimeEventsError ? 'error' :
    opRuntimeEventsSuccess ? 'live' : 'closed';

  const handleTabChange = useCallback((_: unknown, value: TabId) => {
    setActiveTab(value);
  }, []);

  useEffect(() => {
    setSelectedArtifactId(null);
    setActiveTab('input');
  }, [op?.op.ID]);

  if (!op || !selectedSpec) return null;

  const spec = selectedSpec;
  const safeArtifacts = artifacts ?? [];
  const logArtifact = safeArtifacts.find((a) => a.kind === 'execution-log');
  const nonLogArtifacts = safeArtifacts.filter((a) => a.kind !== 'execution-log');
  const logEntries: { timestamp: string; message: string }[] = [];
  if (logArtifact && artifactBodies[logArtifact.id]) {
    try {
      const parsed = JSON.parse(artifactBodies[logArtifact.id]);
      if (Array.isArray(parsed)) logEntries.push(...parsed);
    } catch { /* ignore */ }
  }

  return (
    <Drawer
      anchor="right"
      open={open}
      onClose={onClose}
      PaperProps={{ sx: { width: 500, display: 'flex', flexDirection: 'column' } }}
    >
      {/* Header */}
      <Box sx={{ p: 2.5, pb: 0 }}>
        <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start' }}>
          <Box>
            <Typography
              variant="subtitle1"
              sx={{ fontFamily: 'monospace', fontSize: '0.85rem', fontWeight: 600 }}
            >
              {spec.ID}
            </Typography>
            <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mt: 0.5 }}>
              <Box sx={{ display: 'flex', alignItems: 'center' }}>
                <KindIcon kind={spec.Kind} />
                <Typography variant="body2">{spec.Kind}</Typography>
              </Box>
              <StatusChip status={op.status} />
              {op.status === 'failed' && onRetry && (
                <RetryOpButton
                  workflowId={spec.WorkflowID}
                  opId={spec.ID}
                  disabled={false}
                  onRetry={onRetry}
                  loading={retryLoading ?? false}
                />
              )}
            </Box>
            {spec.Metadata?.script && (
              <Typography variant="caption" color="text.secondary" sx={{ mt: 0.5, display: 'block' }}>
                Script: {spec.Metadata.script}
              </Typography>
            )}
            <Typography variant="caption" color="text.secondary">
              Queue: {spec.Queue}
            </Typography>
          </Box>
          <IconButton onClick={onClose} size="small">
            <CloseIcon fontSize="small" />
          </IconButton>
        </Box>

        <Tabs
          value={activeTab}
          onChange={handleTabChange}
          variant="scrollable"
          scrollButtons="auto"
          sx={{ mt: 1.5, minHeight: 36 }}
        >
          <Tab label="Input" value="input" sx={{ minHeight: 36, py: 0 }} />
          <Tab label="Deps" value="deps" sx={{ minHeight: 36, py: 0 }} />
          <Tab label="Result" value="result" sx={{ minHeight: 36, py: 0 }} />
          <Tab
            label={
              <Badge badgeContent={nonLogArtifacts.length} color="primary" max={99}>
                <Box sx={{ pr: nonLogArtifacts.length > 0 ? 1.5 : 0 }}>Artifacts</Box>
              </Badge>
            }
            value="artifacts"
            sx={{ minHeight: 36, py: 0 }}
          />
          <Tab
            label={
              <Badge badgeContent={opRuntimeEvents.length} color="primary" max={99}>
                <Box sx={{ pr: opRuntimeEvents.length > 0 ? 1.5 : 0, display: 'flex', alignItems: 'center', gap: 0.75 }}>
                  Runtime
                  <CircleIcon sx={{ fontSize: 9, color: `${connectionColor(runtimeConnectionState)}.main` }} />
                </Box>
              </Badge>
            }
            value="runtime"
            sx={{ minHeight: 36, py: 0 }}
          />
          {spec.Metadata?.script && (
            <Tab label="Script" value="script" sx={{ minHeight: 36, py: 0 }} />
          )}
          <Tab label="Logs" value="logs" sx={{ minHeight: 36, py: 0 }} />
        </Tabs>
      </Box>

      <Divider />

      {/* Tab content */}
      <Box sx={{ flexGrow: 1, overflow: 'auto', p: 2.5, pt: 1.5 }}>
        {activeTab === 'input' && (
          <OpInputTab input={spec.Input} />
        )}

        {activeTab === 'deps' && (
          <OpDepsTab dependsOn={spec.DependsOn} />
        )}

        {activeTab === 'result' && (
          <OpResultTab result={result} op={op} onBrowseArtifacts={onBrowseArtifacts} />
        )}

        {activeTab === 'artifacts' && (
          <OpArtifactsTab
            artifacts={nonLogArtifacts}
            artifactBodies={artifactBodies}
            selectedArtifactId={selectedArtifactId}
            onSelectArtifact={setSelectedArtifactId}
          />
        )}

        {activeTab === 'runtime' && (
          <OpRuntimeTab
            events={opRuntimeEvents}
            loading={opRuntimeEventsLoading}
            connectionState={runtimeConnectionState}
          />
        )}

        {activeTab === 'script' && spec.Metadata?.script && (
          <OpScriptTab
            site={site}
            scriptPath={scriptPath || spec.Metadata.script}
            source={scriptSource ?? null}
            loading={scriptLoading ?? false}
          />
        )}

        {activeTab === 'logs' && (
          <OpLogsTab entries={logEntries} />
        )}
      </Box>
    </Drawer>
  );
}
