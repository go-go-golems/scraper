import { useEffect, useState } from 'react';
import {
  Box,
  Button,
  CircularProgress,
  Divider,
  IconButton,
  Tooltip,
  Typography,
} from '@mui/material';
import CloseIcon from '@mui/icons-material/Close';
import OpenInNewIcon from '@mui/icons-material/OpenInNew';
import type { OpResult, WorkflowResultSummary } from '../../api/types';
import { JsonViewer } from '../common/JsonViewer';

interface ResultPreviewPanelProps {
  result: WorkflowResultSummary | null;
  workflowId: string;
  onClose: () => void;
  /** Navigate to the Ops tab and open OpDetailDrawer for this op */
  onOpClick: (opId: string) => void;
}

function formatDate(iso: string): string {
  try {
    const d = new Date(iso);
    const now = Date.now();
    const diffMs = now - d.getTime();
    const diffMin = Math.floor(diffMs / 60000);
    if (diffMin < 60) return `${diffMin}m ago`;
    const diffHr = Math.floor(diffMin / 60);
    if (diffHr < 24) return `${diffHr}h ago`;
    const diffDay = Math.floor(diffHr / 24);
    return `${diffDay}d ago`;
  } catch {
    return iso;
  }
}

export function ResultPreviewPanel({ result, workflowId, onClose, onOpClick }: ResultPreviewPanelProps) {
  const [body, setBody] = useState<OpResult | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(false);

  useEffect(() => {
    if (!result) {
      setBody(null);
      setError(false);
      return;
    }

    setLoading(true);
    setError(false);
    setBody(null);

    fetch(`/api/v1/workflows/${workflowId}/ops/${result.opID}/result`)
      .then((r) => {
        if (!r.ok) throw new Error(`HTTP ${r.status}`);
        return r.json() as Promise<{ result: OpResult }>;
      })
      .then(({ result: opResult }) => {
        setBody(opResult);
        setLoading(false);
      })
      .catch(() => {
        setError(true);
        setLoading(false);
      });
  }, [result, workflowId]);

  if (!result) {
    return (
      <Box
        sx={{
          p: 3,
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          height: '100%',
        }}
      >
        <Typography variant="body2" color="text.disabled">
          Select a result to preview
        </Typography>
      </Box>
    );
  }

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', height: '100%', minHeight: 0 }}>
      {/* Header */}
      <Box sx={{ px: 2, pt: 2, pb: 1, flexShrink: 0 }}>
        <Box sx={{ display: 'flex', alignItems: 'flex-start', justifyContent: 'space-between', gap: 1 }}>
          <Tooltip title={result.opID}>
            <Typography
              variant="subtitle2"
              sx={{
                fontFamily: 'monospace',
                overflow: 'hidden',
                textOverflow: 'ellipsis',
                whiteSpace: 'nowrap',
                maxWidth: '80%',
              }}
            >
              {result.opID.split(':').pop()}
            </Typography>
          </Tooltip>
          <Tooltip title="Close preview">
            <IconButton size="small" onClick={onClose}>
              <CloseIcon fontSize="small" />
            </IconButton>
          </Tooltip>
        </Box>

        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mt: 0.5, flexWrap: 'wrap' }}>
          <Typography variant="caption" color="text.disabled">
            {result.kind}
          </Typography>
          <Typography
            variant="caption"
            sx={{
              color: result.status === 'succeeded' ? 'success.dark' : 'error.dark',
              fontFamily: 'monospace',
              textTransform: 'capitalize',
            }}
          >
            {result.status}
          </Typography>
          <Typography variant="caption" color="text.disabled">
            {result.recordCount} record{result.recordCount !== 1 ? 's' : ''}
          </Typography>
          <Typography variant="caption" color="text.disabled">
            {result.artifactCount} artifact{result.artifactCount !== 1 ? 's' : ''}
          </Typography>
          <Typography variant="caption" color="text.disabled">
            {formatDate(result.completedAt)}
          </Typography>
        </Box>
      </Box>

      {/* Action bar */}
      <Box sx={{ display: 'flex', gap: 1, px: 2, pb: 1, flexShrink: 0 }}>
        <Button
          size="small"
          variant="outlined"
          startIcon={<OpenInNewIcon />}
          onClick={() => onOpClick(result.opID)}
        >
          Open in Drawer
        </Button>
        {result.error && (
          <Tooltip title={result.error.Message}>
            <Typography variant="caption" color="error" sx={{ alignSelf: 'center' }}>
              {result.error.Code}
            </Typography>
          </Tooltip>
        )}
      </Box>

      <Divider />

      {/* Preview content */}
      <Box sx={{ flex: 1, overflow: 'auto', p: 2, minHeight: 0 }}>
        {loading && (
          <Box sx={{ display: 'flex', justifyContent: 'center', p: 4 }}>
            <CircularProgress size={24} />
          </Box>
        )}

        {error && (
          <Typography color="error" variant="body2">
            Failed to load result body.
          </Typography>
        )}

        {!loading && !error && body && (
          <Box>
            <Typography variant="caption" color="text.secondary" sx={{ mb: 1, display: 'block' }}>
              Op Result — {body.Data ? 'has data payload' : 'no data payload'}, {body.Records?.length ?? 0} records, {body.Artifacts?.length ?? 0} artifacts
            </Typography>
            {body.Data && <JsonViewer data={body.Data} maxHeight={500} />}
            {body.Records && body.Records.length > 0 && (
              <>
                <Typography variant="caption" color="text.secondary" sx={{ mt: 1, display: 'block' }}>
                  Records ({body.Records.length})
                </Typography>
                <JsonViewer data={body.Records} maxHeight={300} />
              </>
            )}
          </Box>
        )}

        {!loading && !error && !body && (
          <Box sx={{ p: 4, textAlign: 'center' }}>
            <Typography variant="body2" color="text.disabled">
              No result data available.
            </Typography>
          </Box>
        )}
      </Box>
    </Box>
  );
}
