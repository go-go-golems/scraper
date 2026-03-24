import { useState } from 'react';
import {
  Alert,
  Box,
  List,
  ListItemButton,
  ListItemText,
  Skeleton,
  Typography,
} from '@mui/material';
import { ScriptViewer } from '../scripts/ScriptViewer';
import { useGetScriptQuery } from '../../api/catalogApi';

interface SiteScriptBrowserProps {
  site: string;
  scripts: string[];
}

function groupScripts(scripts: string[]): { topLevel: string[]; groups: Record<string, string[]> } {
  const topLevel: string[] = [];
  const groups: Record<string, string[]> = {};

  for (const s of scripts) {
    const slashIdx = s.indexOf('/');
    if (slashIdx > 0) {
      const dir = s.substring(0, slashIdx);
      if (!groups[dir]) groups[dir] = [];
      groups[dir].push(s);
    } else {
      topLevel.push(s);
    }
  }

  return { topLevel, groups };
}

function ScriptContent({ site, path }: { site: string; path: string }) {
  const { data, isLoading, error } = useGetScriptQuery({ site, path });

  if (isLoading) {
    return (
      <Box>
        <Skeleton variant="text" width={160} sx={{ mb: 1 }} />
        <Skeleton variant="rectangular" height={300} sx={{ borderRadius: 1 }} />
      </Box>
    );
  }

  if (error) {
    return <Alert severity="error">Failed to load script</Alert>;
  }

  if (data) {
    return <ScriptViewer source={data.source} filename={path} />;
  }

  return null;
}

export function SiteScriptBrowser({ site, scripts }: SiteScriptBrowserProps) {
  const [selected, setSelected] = useState<string | null>(null);
  const { topLevel, groups } = groupScripts(scripts);

  return (
    <Box sx={{ display: 'flex', gap: 2, minHeight: 400 }}>
      {/* Left: file list */}
      <Box
        sx={{
          width: 240,
          flexShrink: 0,
          borderRight: '1px solid',
          borderColor: 'divider',
          overflow: 'auto',
        }}
      >
        <List dense disablePadding>
          {topLevel.map((s) => (
            <ListItemButton
              key={s}
              selected={selected === s}
              onClick={() => setSelected(s)}
            >
              <ListItemText
                primary={s}
                primaryTypographyProps={{ variant: 'body2', fontFamily: 'monospace', fontSize: '0.8rem' }}
              />
            </ListItemButton>
          ))}
          {Object.entries(groups).map(([dir, files]) => (
            <Box key={dir}>
              <Typography
                variant="caption"
                color="text.secondary"
                sx={{ px: 2, pt: 1, display: 'block', fontWeight: 600 }}
              >
                {dir}/
              </Typography>
              {files.map((s) => (
                <ListItemButton
                  key={s}
                  selected={selected === s}
                  onClick={() => setSelected(s)}
                  sx={{ pl: 4 }}
                >
                  <ListItemText
                    primary={s.substring(dir.length + 1)}
                    primaryTypographyProps={{ variant: 'body2', fontFamily: 'monospace', fontSize: '0.8rem' }}
                  />
                </ListItemButton>
              ))}
            </Box>
          ))}
        </List>
      </Box>

      {/* Right: script viewer */}
      <Box sx={{ flexGrow: 1, overflow: 'auto' }}>
        {selected ? (
          <ScriptContent site={site} path={selected} />
        ) : (
          <Box
            sx={{
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              height: '100%',
            }}
          >
            <Typography variant="body2" color="text.disabled">
              Select a script
            </Typography>
          </Box>
        )}
      </Box>
    </Box>
  );
}
