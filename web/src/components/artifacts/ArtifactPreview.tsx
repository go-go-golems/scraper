import { Box, Typography } from '@mui/material';
import { CodeViewPanel } from '../common/CodeViewPanel';

interface ArtifactPreviewProps {
  content: string;
  contentType: string;
  name: string;
}

export function ArtifactPreview({ content, contentType, name }: ArtifactPreviewProps) {
  return (
    <Box>
      <Typography variant="subtitle2" sx={{ mb: 1 }}>
        {name}
      </Typography>

      <CodeViewPanel
        data={contentType === 'text/html' || contentType === 'text/plain'
          ? content
          : JSON.parse(content)
        }
        label={name}
        defaultFormat={contentType === 'text/html' || contentType === 'text/plain' ? 'html' : 'yaml'}
        formats={contentType === 'text/html'
          ? ['html', 'json', 'yaml']
          : contentType === 'text/plain'
            ? ['yaml', 'json']
            : ['json', 'yaml']
        }
        maxHeight={360}
      />
    </Box>
  );
}
