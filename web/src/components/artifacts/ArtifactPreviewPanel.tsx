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
import OpenInBrowserIcon from '@mui/icons-material/OpenInBrowser';
import CloudDownloadIcon from '@mui/icons-material/CloudDownload';

import type { ArtifactSummary } from '../../api/types';
import { ArtifactPreview } from './ArtifactPreview';
import { BinaryFallbackView } from './BinaryFallbackView';

interface ArtifactPreviewPanelProps {
  artifact: ArtifactSummary | null;
  onClose: () => void;
}

function formatSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  const kb = bytes / 1024;
  if (kb < 1024) return `${kb.toFixed(1)} KB`;
  const mb = kb / 1024;
  return `${mb.toFixed(1)} MB`;
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

export function ArtifactPreviewPanel({ artifact, onClose }: ArtifactPreviewPanelProps) {
  const [body, setBody] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(false);

  useEffect(() => {
    if (!artifact) {
      setBody(null);
      setError(false);
      return;
    }

    if (!artifact.previewable) {
      setBody(null);
      setError(false);
      return;
    }

    setLoading(true);
    setError(false);
    setBody(null);

    fetch(`/api/v1/artifacts/${artifact.id}`)
      .then((r) => {
        if (!r.ok) throw new Error(`HTTP ${r.status}`);
        return r.text();
      })
      .then((text) => {
        setBody(text);
        setLoading(false);
      })
      .catch(() => {
        setError(true);
        setLoading(false);
      });
  }, [artifact]);

  if (!artifact) {
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
          Select an artifact to preview
        </Typography>
      </Box>
    );
  }

  const isImage = artifact.contentType.startsWith('image/');
  const isPreviewable = artifact.previewable && (artifact.contentType === 'text/html' || artifact.contentType === 'text/plain' || artifact.contentType === 'application/json');

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', height: '100%', minHeight: 0 }}>
      {/* Header */}
      <Box sx={{ px: 2, pt: 2, pb: 1, flexShrink: 0 }}>
        <Box sx={{ display: 'flex', alignItems: 'flex-start', justifyContent: 'space-between', gap: 1 }}>
          <Tooltip title={artifact.name}>
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
              {artifact.name}
            </Typography>
          </Tooltip>
          <Tooltip title="Close preview">
            <IconButton size="small" onClick={onClose}>
              <CloseIcon fontSize="small" />
            </IconButton>
          </Tooltip>
        </Box>

        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mt: 0.5, flexWrap: 'wrap' }}>
          <Typography variant="caption" color="text.secondary" sx={{ fontFamily: 'monospace' }}>
            {artifact.opID.split(':').pop()}
          </Typography>
          <Typography variant="caption" color="text.disabled">
            {artifact.kind}
          </Typography>
          <Typography variant="caption" color="text.disabled">
            {artifact.contentType}
          </Typography>
          <Typography variant="caption" color="text.disabled">
            {formatSize(artifact.size)}
          </Typography>
          <Typography variant="caption" color="text.disabled">
            {formatDate(artifact.createdAt)}
          </Typography>
        </Box>
      </Box>

      {/* Action bar */}
      <Box sx={{ display: 'flex', gap: 1, px: 2, pb: 1, flexShrink: 0 }}>
        <Button
          size="small"
          startIcon={<OpenInBrowserIcon />}
          component="a"
          href={`/api/v1/artifacts/${artifact.id}`}
          target="_blank"
          rel="noopener noreferrer"
          variant="outlined"
        >
          Open Raw
        </Button>
        <Button
          size="small"
          startIcon={<CloudDownloadIcon />}
          component="a"
          href={`/api/v1/artifacts/${artifact.id}`}
          target="_blank"
          rel="noopener noreferrer"
          download
          variant="outlined"
        >
          Download
        </Button>
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
            Failed to load artifact body.
          </Typography>
        )}

        {!loading && !error && body && isPreviewable && (
          <ArtifactPreview
            content={body}
            contentType={artifact.contentType}
            name={artifact.name}
          />
        )}

        {!loading && !error && isImage && (
          <Box sx={{ textAlign: 'center' }}>
            <img
              src={`/api/v1/artifacts/${artifact.id}`}
              alt={artifact.name}
              style={{ maxWidth: '100%', borderRadius: 4 }}
            />
            <Button
              variant="outlined"
              size="small"
              component="a"
              href={`/api/v1/artifacts/${artifact.id}`}
              target="_blank"
              rel="noopener noreferrer"
              download
              sx={{ mt: 1 }}
            >
              Download full image
            </Button>
          </Box>
        )}

        {!loading && !error && !body && !isImage && (
          <BinaryFallbackView
            name={artifact.name}
            size={artifact.size}
            contentType={artifact.contentType}
            artifactId={artifact.id}
          />
        )}
      </Box>
    </Box>
  );
}
