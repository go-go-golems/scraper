import { useState, useCallback, useEffect, useMemo } from 'react';
import {
  Badge,
  Box,
  Chip,
  Divider,
  Drawer,
  IconButton,
  List,
  ListItem,
  ListItemText,
  Stack,
  Tab,
  Tabs,
  Typography,
} from '@mui/material';
import CloseIcon from '@mui/icons-material/Close';
import CodeIcon from '@mui/icons-material/Code';
import HttpIcon from '@mui/icons-material/Http';
import CircleIcon from '@mui/icons-material/Circle';
import type { WorkflowOp, OpResult, ArtifactSummary } from '../../api/types';
import { StatusChip } from '../common/StatusChip';
import { JsonViewer } from '../common/JsonViewer';
import { ArtifactList } from '../artifacts/ArtifactList';
import { ArtifactPreview } from '../artifacts/ArtifactPreview';
import { OpExecutionLog } from '../logs/OpExecutionLog';
import { ScriptTab } from '../scripts/ScriptTab';
import { RetryOpButton } from './RetryOpButton';
import { RuntimeEventTable } from './RuntimeEventTable';
import { decodeRuntimeEvent, useGetRecentRuntimeEventsQuery } from '../../api/runtimeEventsApi';

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
}

type TabId = 'input' | 'deps' | 'result' | 'artifacts' | 'runtime' | 'script' | 'logs';

function SectionTitle({ children }: { children: React.ReactNode }) {
  return (
    <Typography variant="overline" color="text.secondary" sx={{ mt: 2, mb: 0.5, display: 'block' }}>
      {children}
    </Typography>
  );
}

function KindIcon({ kind }: { kind: string }) {
  if (kind === 'js') return <CodeIcon fontSize="small" sx={{ mr: 0.5 }} />;
  if (kind === 'http' || kind === 'http/fetch') return <HttpIcon fontSize="small" sx={{ mr: 0.5 }} />;
  return null;
}

type ConnectionState = 'connecting' | 'live' | 'error' | 'closed';

function connectionColor(state: ConnectionState): 'disabled' | 'success' | 'warning' | 'error' {
  switch (state) {
    case 'live':
      return 'success';
    case 'connecting':
      return 'warning';
    case 'error':
      return 'error';
    default:
      return 'disabled';
  }
}

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

  const selectedArtifact = selectedArtifactId
    ? nonLogArtifacts.find((a) => a.id === selectedArtifactId) ?? null
    : null;

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
          <JsonViewer data={spec.Input} maxHeight={500} />
        )}

        {activeTab === 'deps' && (
          <>
            {spec.DependsOn.length === 0 ? (
              <Typography variant="body2" color="text.secondary">No dependencies</Typography>
            ) : (
              <List dense disablePadding>
                {spec.DependsOn.map((dep) => (
                  <ListItem key={dep.OpID} disablePadding sx={{ py: 0.25 }}>
                    <ListItemText
                      primary={
                        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                          <Typography variant="body2" sx={{ fontFamily: 'monospace', fontSize: '0.8rem' }}>
                            {dep.OpID}
                          </Typography>
                          {dep.Required && (
                            <Chip label="required" size="small" color="primary" variant="outlined" />
                          )}
                        </Box>
                      }
                    />
                  </ListItem>
                ))}
              </List>
            )}
          </>
        )}

        {activeTab === 'result' && (
          <>
            {result ? (
              <>
                {result.Data !== undefined && result.Data !== null && (
                  <Box sx={{ mb: 1.5 }}>
                    <SectionTitle>Data</SectionTitle>
                    <JsonViewer data={result.Data} maxHeight={250} />
                  </Box>
                )}
                {result.Error && (
                  <Box sx={{ mb: 1.5 }}>
                    <SectionTitle>Error</SectionTitle>
                    <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mb: 0.5 }}>
                      <Typography variant="body2" fontWeight={600}>{result.Error.Code}</Typography>
                      <Chip
                        label={result.Error.Retryable ? 'retryable' : 'non-retryable'}
                        size="small"
                        color={result.Error.Retryable ? 'warning' : 'error'}
                        variant="outlined"
                      />
                    </Box>
                    <Typography variant="body2">{result.Error.Message}</Typography>
                  </Box>
                )}
                <Box sx={{ display: 'flex', gap: 2 }}>
                  <Typography variant="caption" color="text.secondary">
                    Artifacts: {result.Artifacts.length}
                  </Typography>
                  <Typography variant="caption" color="text.secondary">
                    Emitted: {result.EmittedIDs.length}
                  </Typography>
                </Box>
                <SectionTitle>Retry</SectionTitle>
                <Typography variant="caption">
                  Attempt: {spec.RetryState.Attempt}/{spec.Retry.MaxAttempts}
                </Typography>
                {spec.RetryState.LastError && (
                  <Typography variant="caption" color="error" display="block">
                    Last error: {spec.RetryState.LastError}
                  </Typography>
                )}
                {op.lease && (
                  <>
                    <SectionTitle>Lease</SectionTitle>
                    <Typography variant="caption" display="block">
                      Worker: {op.lease.WorkerID}
                    </Typography>
                    <Typography variant="caption" display="block">
                      Acquired: {new Date(op.lease.AcquiredAt).toLocaleString()}
                    </Typography>
                    <Typography variant="caption" display="block">
                      Expires: {new Date(op.lease.ExpiresAt).toLocaleString()}
                    </Typography>
                  </>
                )}
              </>
            ) : (
              <Typography variant="body2" color="text.secondary">No result yet</Typography>
            )}
          </>
        )}

        {activeTab === 'artifacts' && (
          <>
            {nonLogArtifacts.length === 0 ? (
              <Typography variant="body2" color="text.secondary">No artifacts</Typography>
            ) : (
              <>
                <ArtifactList
                  artifacts={nonLogArtifacts}
                  selectedId={selectedArtifactId}
                  onSelect={setSelectedArtifactId}
                />
                {selectedArtifact && artifactBodies[selectedArtifact.id] && (
                  <Box sx={{ mt: 2 }}>
                    <ArtifactPreview
                      content={artifactBodies[selectedArtifact.id]}
                      contentType={selectedArtifact.contentType}
                      name={selectedArtifact.name}
                    />
                  </Box>
                )}
              </>
            )}
          </>
        )}

        {activeTab === 'runtime' && (
          <>
            <Stack direction="row" spacing={1} sx={{ mb: 1.5 }} flexWrap="wrap" useFlexGap>
              <Chip label={`Stream: ${runtimeConnectionState}`} size="small" variant="outlined" />
              <Chip label={`${opRuntimeEvents.length} events`} size="small" variant="outlined" />
            </Stack>
            <RuntimeEventTable
              events={opRuntimeEvents}
              loading={opRuntimeEventsLoading}
              dense
              emptyMessage="No runtime events for this op yet."
            />
          </>
        )}

        {activeTab === 'script' && spec.Metadata?.script && (
          <ScriptTab
            site={site}
            scriptPath={scriptPath || spec.Metadata.script}
            source={scriptSource ?? null}
            loading={scriptLoading ?? false}
            error={null}
          />
        )}

        {activeTab === 'logs' && (
          <OpExecutionLog entries={logEntries} loading={false} />
        )}
      </Box>
    </Drawer>
  );
}
