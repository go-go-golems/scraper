import {
  Box,
  Chip,
  Divider,
  Drawer,
  IconButton,
  List,
  ListItem,
  ListItemText,
  Typography,
} from '@mui/material';
import CloseIcon from '@mui/icons-material/Close';
import CodeIcon from '@mui/icons-material/Code';
import HttpIcon from '@mui/icons-material/Http';
import type { WorkflowOp, OpResult } from '../../api/types';
import { StatusChip } from '../common/StatusChip';
import { JsonViewer } from '../common/JsonViewer';

interface OpDetailDrawerProps {
  op: WorkflowOp | null;
  result: OpResult | null;
  open: boolean;
  onClose: () => void;
}

function SectionTitle({ children }: { children: React.ReactNode }) {
  return (
    <Typography variant="overline" color="text.secondary" sx={{ mt: 2, mb: 0.5, display: 'block' }}>
      {children}
    </Typography>
  );
}

function KindIcon({ kind }: { kind: string }) {
  if (kind === 'js') return <CodeIcon fontSize="small" sx={{ mr: 0.5 }} />;
  if (kind === 'http' || kind === 'fetch') return <HttpIcon fontSize="small" sx={{ mr: 0.5 }} />;
  return null;
}

export function OpDetailDrawer({ op, result, open, onClose }: OpDetailDrawerProps) {
  if (!op) return null;

  const { op: spec } = op;

  return (
    <Drawer
      anchor="right"
      open={open}
      onClose={onClose}
      PaperProps={{ sx: { width: 450, p: 2.5 } }}
    >
      {/* Header */}
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

      <Divider sx={{ my: 2 }} />

      {/* Input */}
      <SectionTitle>Input</SectionTitle>
      <JsonViewer data={spec.Input} maxHeight={200} />

      {/* Dependencies */}
      {spec.DependsOn.length > 0 && (
        <>
          <SectionTitle>Dependencies</SectionTitle>
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
        </>
      )}

      {/* Result */}
      {result && (
        <>
          <SectionTitle>Result</SectionTitle>
          {result.Data !== undefined && result.Data !== null && (
            <Box sx={{ mb: 1 }}>
              <Typography variant="caption" color="text.secondary" gutterBottom>
                Data:
              </Typography>
              <JsonViewer data={result.Data} maxHeight={200} />
            </Box>
          )}
          <Box sx={{ display: 'flex', gap: 2, mt: 0.5 }}>
            <Typography variant="caption" color="text.secondary">
              Artifacts: {result.Artifacts.length}
            </Typography>
            <Typography variant="caption" color="text.secondary">
              Emitted: {result.EmittedIDs.length}
            </Typography>
          </Box>
          {result.EmittedIDs.length > 0 && (
            <Box sx={{ mt: 0.5 }}>
              <Typography variant="caption" color="text.secondary">
                Emitted IDs:
              </Typography>
              <List dense disablePadding>
                {result.EmittedIDs.map((id) => (
                  <ListItem key={id} disablePadding sx={{ py: 0.1 }}>
                    <Typography variant="caption" sx={{ fontFamily: 'monospace', fontSize: '0.75rem' }}>
                      {id}
                    </Typography>
                  </ListItem>
                ))}
              </List>
            </Box>
          )}
        </>
      )}

      {/* Error */}
      {result?.Error && (
        <>
          <SectionTitle>Error</SectionTitle>
          <Box sx={{ display: 'flex', flexDirection: 'column', gap: 0.5 }}>
            <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
              <Typography variant="body2" fontWeight={600}>
                {result.Error.Code}
              </Typography>
              <Chip
                label={result.Error.Retryable ? 'retryable' : 'non-retryable'}
                size="small"
                color={result.Error.Retryable ? 'warning' : 'error'}
                variant="outlined"
              />
            </Box>
            <Typography variant="body2">{result.Error.Message}</Typography>
          </Box>
        </>
      )}

      {/* Retry */}
      <SectionTitle>Retry</SectionTitle>
      <Box sx={{ display: 'flex', flexDirection: 'column', gap: 0.25 }}>
        <Typography variant="caption">
          Attempt: {spec.RetryState.Attempt}/{spec.Retry.MaxAttempts}
        </Typography>
        <Typography variant="caption">
          Backoff: {spec.Retry.BackoffKind}, initial {spec.Retry.InitialBackoff / 1e9}s, max{' '}
          {spec.Retry.MaxBackoff / 1e9}s, multiplier {spec.Retry.Multiplier}x
        </Typography>
        {spec.RetryState.LastError && (
          <Typography variant="caption" color="error">
            Last error: {spec.RetryState.LastError}
          </Typography>
        )}
      </Box>

      {/* Lease */}
      {op.lease && (
        <>
          <SectionTitle>Lease</SectionTitle>
          <Box sx={{ display: 'flex', flexDirection: 'column', gap: 0.25 }}>
            <Typography variant="caption">
              Worker: {op.lease.WorkerID}
            </Typography>
            <Typography variant="caption">
              Acquired: {new Date(op.lease.AcquiredAt).toLocaleString()}
            </Typography>
            <Typography variant="caption">
              Expires: {new Date(op.lease.ExpiresAt).toLocaleString()}
            </Typography>
          </Box>
        </>
      )}
    </Drawer>
  );
}
