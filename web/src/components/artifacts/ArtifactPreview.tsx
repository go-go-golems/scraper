import { Box, Typography } from '@mui/material';
import { JsonViewer } from '../common/JsonViewer';

interface ArtifactPreviewProps {
  content: string;
  contentType: string;
  name: string;
}

const monoStyle = {
  margin: 0,
  fontFamily: '"JetBrains Mono", "Fira Code", monospace',
  fontSize: '0.8rem',
  lineHeight: 1.5,
  whiteSpace: 'pre-wrap',
  wordBreak: 'break-word',
} as const;

export function ArtifactPreview({ content, contentType, name }: ArtifactPreviewProps) {
  return (
    <Box>
      <Typography variant="subtitle2" sx={{ mb: 1 }}>
        {name}
      </Typography>

      <Box
        sx={{
          maxHeight: 400,
          overflow: 'auto',
          bgcolor: 'grey.50',
          borderRadius: 1,
          border: 1,
          borderColor: 'divider',
          p: 1.5,
        }}
      >
        {contentType === 'application/json' ? (
          <JsonViewer data={JSON.parse(content)} maxHeight={360} />
        ) : contentType === 'text/html' ? (
          <pre style={monoStyle}>
            <code>{content}</code>
          </pre>
        ) : contentType === 'text/plain' ? (
          <pre style={monoStyle}>{content}</pre>
        ) : (
          <Typography variant="body2" color="text.secondary">
            Binary content &mdash; download the artifact to view.
          </Typography>
        )}
      </Box>
    </Box>
  );
}
