import { Box, Chip, Typography } from '@mui/material';
import CodeMirror from '@uiw/react-codemirror';
import { javascript } from '@codemirror/lang-javascript';

interface ScriptViewerProps {
  source: string;
  filename: string;
}

export function ScriptViewer({ source, filename }: ScriptViewerProps) {
  return (
    <Box>
      <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mb: 1 }}>
        <Typography
          variant="caption"
          color="text.secondary"
          sx={{ fontFamily: 'monospace' }}
        >
          {filename}
        </Typography>
        <Chip label="read only" size="small" sx={{ fontSize: '0.65rem', height: 18 }} />
      </Box>
      <Box
        sx={{
          maxHeight: 500,
          overflow: 'hidden',
          borderRadius: 1,
          border: '1px solid',
          borderColor: 'divider',
        }}
      >
        <CodeMirror
          value={source}
          extensions={[javascript({ jsx: false, typescript: false })]}
          editable={() => false}
          theme="light"
          style={{ height: 500 }}
          basicSetup={{
            lineNumbers: true,
            foldGutter: true,
            highlightActiveLine: false,
            highlightSelectionMatches: false,
            dropCursor: false,
            allowMultipleSelections: false,
            indentOnInput: false,
            autocompletion: false,
            bracketMatching: false,
            closeBrackets: false,
            rectangularSelection: false,
            crosshairCursor: false,
            highlightActiveLineGutter: false,
          }}
        />
      </Box>
    </Box>
  );
}
