import { List, ListItemButton, ListItemIcon, ListItemText, Typography } from '@mui/material';
import HtmlIcon from '@mui/icons-material/Html';
import DataObjectIcon from '@mui/icons-material/DataObject';
import DescriptionIcon from '@mui/icons-material/Description';
import InsertDriveFileIcon from '@mui/icons-material/InsertDriveFile';
import type { ArtifactSummary } from '../../api/types';

interface ArtifactListProps {
  artifacts: ArtifactSummary[];
  selectedId: string | null;
  onSelect: (id: string) => void;
}

function formatSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  const kb = bytes / 1024;
  if (kb < 1024) return `${kb.toFixed(1)} KB`;
  const mb = kb / 1024;
  return `${mb.toFixed(1)} MB`;
}

function fileTypeIcon(contentType: string) {
  if (contentType === 'text/html') return <HtmlIcon />;
  if (contentType === 'application/json') return <DataObjectIcon />;
  if (contentType.startsWith('text/')) return <DescriptionIcon />;
  return <InsertDriveFileIcon />;
}

export function ArtifactList({ artifacts, selectedId, onSelect }: ArtifactListProps) {
  if (artifacts.length === 0) {
    return (
      <Typography variant="body2" color="text.secondary" sx={{ p: 2 }}>
        No artifacts
      </Typography>
    );
  }

  return (
    <List disablePadding>
      {artifacts.map((artifact) => (
        <ListItemButton
          key={artifact.id}
          selected={artifact.id === selectedId}
          onClick={() => onSelect(artifact.id)}
        >
          <ListItemIcon sx={{ minWidth: 40 }}>
            {fileTypeIcon(artifact.contentType)}
          </ListItemIcon>
          <ListItemText
            primary={artifact.name}
            secondary={`${artifact.kind} \u00b7 ${formatSize(artifact.size)}`}
          />
        </ListItemButton>
      ))}
    </List>
  );
}
