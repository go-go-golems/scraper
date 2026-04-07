import { Box, Button, Typography } from '@mui/material';
import InsertDriveFileIcon from '@mui/icons-material/InsertDriveFile';

interface BinaryFallbackViewProps {
  name: string;
  size: number;
  contentType: string;
  artifactId: string;
}

function formatSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  const kb = bytes / 1024;
  if (kb < 1024) return `${kb.toFixed(1)} KB`;
  const mb = kb / 1024;
  return `${mb.toFixed(1)} MB`;
}

export function BinaryFallbackView({ name, size, contentType, artifactId }: BinaryFallbackViewProps) {
  return (
    <Box
      sx={{
        display: 'flex',
        flexDirection: 'column',
        alignItems: 'center',
        gap: 1.5,
        p: 4,
        textAlign: 'center',
      }}
    >
      <InsertDriveFileIcon sx={{ fontSize: 48, color: 'text.disabled' }} />
      <Typography variant="body2" color="text.secondary">
        Binary file — cannot preview in browser
      </Typography>
      <Box sx={{ display: 'flex', gap: 2 }}>
        <Typography variant="caption" color="text.disabled">
          {formatSize(size)}
        </Typography>
        <Typography variant="caption" color="text.disabled">
          {contentType}
        </Typography>
      </Box>
      <Button
        variant="outlined"
        size="small"
        component="a"
        href={`/api/v1/artifacts/${artifactId}`}
        target="_blank"
        rel="noopener noreferrer"
        download
      >
        Download {name}
      </Button>
    </Box>
  );
}
