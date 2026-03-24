import { useState } from 'react';
import { Box, IconButton, Collapse } from '@mui/material';
import ExpandMoreIcon from '@mui/icons-material/ExpandMore';
import ExpandLessIcon from '@mui/icons-material/ExpandLess';

interface JsonViewerProps {
  data: unknown;
  maxHeight?: number;
  defaultExpanded?: boolean;
}

export function JsonViewer({ data, maxHeight = 300, defaultExpanded = true }: JsonViewerProps) {
  const [expanded, setExpanded] = useState(defaultExpanded);
  const formatted = JSON.stringify(data, null, 2);

  return (
    <Box>
      <Box sx={{ display: 'flex', justifyContent: 'flex-end' }}>
        <IconButton size="small" onClick={() => setExpanded(!expanded)}>
          {expanded ? <ExpandLessIcon fontSize="small" /> : <ExpandMoreIcon fontSize="small" />}
        </IconButton>
      </Box>
      <Collapse in={expanded}>
        <Box
          sx={{
            maxHeight,
            overflow: 'auto',
            bgcolor: 'grey.50',
            borderRadius: 1,
            border: 1,
            borderColor: 'divider',
            p: 1.5,
          }}
        >
          <pre
            style={{
              margin: 0,
              fontFamily: '"JetBrains Mono", "Fira Code", monospace',
              fontSize: '0.8rem',
              lineHeight: 1.5,
              whiteSpace: 'pre-wrap',
              wordBreak: 'break-word',
            }}
          >
            <code>{formatted}</code>
          </pre>
        </Box>
      </Collapse>
    </Box>
  );
}
