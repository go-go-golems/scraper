import { useState, useCallback } from 'react';
import { Box, Button, Collapse, IconButton, ToggleButton, ToggleButtonGroup, Tooltip, Typography } from '@mui/material';
import ExpandLessIcon from '@mui/icons-material/ExpandLess';
import ExpandMoreIcon from '@mui/icons-material/ExpandMore';
import ContentCopyIcon from '@mui/icons-material/ContentCopy';
import CheckIcon from '@mui/icons-material/Check';
import CodeMirror from '@uiw/react-codemirror';
import { json as jsonLang } from '@codemirror/lang-json';
import { yaml as yamlLang } from '@codemirror/lang-yaml';
import { html as htmlLang } from '@codemirror/lang-html';
import jsYaml from 'js-yaml';
import type { Extension } from '@codemirror/state';

type DataFormat = 'json' | 'yaml' | 'html';

/** Label for each format in the toggle button */
const FORMAT_LABELS: Record<DataFormat, string> = {
  json: 'JSON',
  yaml: 'YAML',
  html: 'HTML',
};

interface CodeViewPanelProps {
  /** The data to render — pass as string for HTML/text content */
  data: unknown;
  /** Label shown in the header bar */
  label?: string;
  /** Default format shown on first render */
  defaultFormat?: DataFormat;
  /** Which format toggle buttons to show */
  formats?: DataFormat[];
  maxHeight?: number;
}

const THEME = {
  '&': {
    fontSize: '0.8rem',
    fontFamily: '"JetBrains Mono", "Fira Code", monospace',
    height: '100%',
  },
  '.cm-scroller': {
    overflow: 'auto',
    fontFamily: '"JetBrains Mono", "Fira Code", monospace',
    fontSize: '0.8rem',
    lineHeight: 1.5,
  },
  '.cm-content': {
    whiteSpace: 'pre-wrap',
    wordBreak: 'break-word',
    padding: '12px',
  },
  '.cm-gutters': {
    backgroundColor: '#f0f0f0',
    borderRight: '1px solid #ddd',
    color: '#aaa',
  },
  '.cm-lineNumbers .cm-gutterElement': {
    minWidth: '2em',
    padding: '0 8px 0 4px',
  },
};

function dataAsString(data: unknown): string {
  if (typeof data === 'string') return data;
  return JSON.stringify(data, null, 2);
}

export function CodeViewPanel({
  data,
  label,
  defaultFormat = 'yaml',
  formats = ['json', 'yaml'],
  maxHeight = 400,
}: CodeViewPanelProps) {
  const [expanded, setExpanded] = useState(true);
  const [format, setFormat] = useState<DataFormat>(defaultFormat);
  const [copied, setCopied] = useState(false);

  const extensions: Extension[] = [
    format === 'json' ? jsonLang() :
    format === 'yaml' ? yamlLang() :
    htmlLang(),
  ];

  const content = dataAsString(data);

  const handleCopy = useCallback(async () => {
    try {
      await navigator.clipboard.writeText(content);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch {
      // clipboard unavailable
    }
  }, [content]);

  const handleFormatChange = useCallback(
    (_: unknown, next: DataFormat | null) => {
      if (next !== null) setFormat(next);
    },
    [],
  );

  const showToggle = formats.length > 1;

  return (
    <Box>
      {/* Header bar */}
      <Box
        sx={{
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center',
          mb: 0.5,
          gap: 1,
        }}
      >
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
          {label && (
            <Typography variant="caption" color="text.secondary" sx={{ fontFamily: 'monospace' }}>
              {label}
            </Typography>
          )}
          {showToggle && (
            <ToggleButtonGroup
              value={format}
              exclusive
              onChange={handleFormatChange}
              size="small"
            >
              {formats.map((f) => (
                <ToggleButton key={f} value={f} sx={{ py: 0.25, px: 1, fontSize: '0.7rem', textTransform: 'none' }}>
                  {FORMAT_LABELS[f]}
                </ToggleButton>
              ))}
            </ToggleButtonGroup>
          )}
        </Box>

        <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5 }}>
          <Tooltip title={copied ? 'Copied!' : 'Copy to clipboard'}>
            <Button
              size="small"
              variant="outlined"
              startIcon={copied ? <CheckIcon fontSize="small" /> : <ContentCopyIcon fontSize="small" />}
              onClick={handleCopy}
              sx={{ fontSize: '0.7rem', py: 0.25, minWidth: 0, px: 1 }}
              color={copied ? 'success' : 'primary'}
            >
              {copied ? 'Copied' : 'Copy'}
            </Button>
          </Tooltip>
          <IconButton size="small" onClick={() => setExpanded((e) => !e)}>
            {expanded ? <ExpandLessIcon fontSize="small" /> : <ExpandMoreIcon fontSize="small" />}
          </IconButton>
        </Box>
      </Box>

      {/* CodeMirror editor — read-only */}
      <Collapse in={expanded}>
        <Box
          sx={{
            maxHeight,
            overflow: 'hidden',
            borderRadius: 1,
            border: '1px solid',
            borderColor: 'divider',
          }}
        >
          <CodeMirror
            value={content}
            extensions={extensions}
            editable={() => false}
            theme="light"
            style={{ height: maxHeight }}
            basicSetup={{
              lineNumbers: true,
              foldGutter: false,
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
      </Collapse>
    </Box>
  );
}
