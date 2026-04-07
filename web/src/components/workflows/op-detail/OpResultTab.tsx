import { Box, Chip, Link, Typography } from '@mui/material';
import type { OpResult, WorkflowOp } from '../../../api/types';
import { JsonViewer } from '../../common/JsonViewer';

function SectionTitle({ children }: { children: React.ReactNode }) {
  return (
    <Typography
      variant="overline"
      color="text.secondary"
      sx={{ mt: 2, mb: 0.5, display: 'block' }}
    >
      {children}
    </Typography>
  );
}

interface OpResultTabProps {
  result: OpResult | null;
  op: WorkflowOp;
  /** Called when user clicks "See all artifacts from this op" —
   *  parent sets the Artifacts tab filter to this op. */
  onBrowseArtifacts?: (opId: string) => void;
}

export function OpResultTab({ result, op, onBrowseArtifacts }: OpResultTabProps) {
  const spec = op.op;

  if (!result) {
    return (
      <Typography variant="body2" color="text.secondary">
        No result yet
      </Typography>
    );
  }

  return (
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
      )}
      <Box sx={{ display: 'flex', gap: 2, alignItems: 'center' }}>
        <Typography variant="caption" color="text.secondary">
          Artifacts: {result.Artifacts?.length ?? 0}
        </Typography>
        <Typography variant="caption" color="text.secondary">
          Emitted: {result.EmittedIDs?.length ?? 0}
        </Typography>
        {result.Artifacts && result.Artifacts.length > 0 && onBrowseArtifacts && (
          <Link
            component="button"
            variant="caption"
            onClick={() => onBrowseArtifacts(op.op.ID)}
            sx={{ display: 'flex', alignItems: 'center', gap: 0.25 }}
          >
            See all {result.Artifacts.length} artifact{result.Artifacts.length > 1 ? 's' : ''} from this op
          </Link>
        )}
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
  );
}
