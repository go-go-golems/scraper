import { Box, Typography } from '@mui/material';

interface ScriptViewerProps {
  source: string;
  filename: string;
}

export function ScriptViewer({ source, filename }: ScriptViewerProps) {
  const lines = source.split('\n');

  return (
    <Box>
      <Typography
        variant="caption"
        color="text.secondary"
        sx={{ mb: 0.5, display: 'block' }}
      >
        {filename}
      </Typography>
      <Box
        sx={{
          maxHeight: 500,
          overflow: 'auto',
          backgroundColor: '#f5f5f5',
          borderRadius: 1,
          border: '1px solid',
          borderColor: 'divider',
        }}
      >
        <Box
          component="pre"
          sx={{
            m: 0,
            p: 1.5,
            fontFamily: 'monospace',
            fontSize: '0.8rem',
            lineHeight: 1.6,
            display: 'flex',
          }}
        >
          <Box
            component="code"
            sx={{
              color: '#999',
              textAlign: 'right',
              pr: 2,
              mr: 2,
              borderRight: '1px solid',
              borderColor: 'divider',
              userSelect: 'none',
              flexShrink: 0,
            }}
          >
            {lines.map((_, i) => (
              <Box key={i} component="span" sx={{ display: 'block' }}>
                {i + 1}
              </Box>
            ))}
          </Box>
          <Box component="code" sx={{ whiteSpace: 'pre', flexGrow: 1 }}>
            {source}
          </Box>
        </Box>
      </Box>
    </Box>
  );
}
