import {
  Box,
  IconButton,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Tooltip,
  Typography,
} from '@mui/material';
import OpenInBrowserIcon from '@mui/icons-material/OpenInBrowser';
import CloudDownloadIcon from '@mui/icons-material/CloudDownload';
import HtmlIcon from '@mui/icons-material/Html';
import DataObjectIcon from '@mui/icons-material/DataObject';
import DescriptionIcon from '@mui/icons-material/Description';
import ImageIcon from '@mui/icons-material/Image';
import InsertDriveFileIcon from '@mui/icons-material/InsertDriveFile';
import type { ArtifactSummary } from '../../api/types';

interface ArtifactTableProps {
  artifacts: ArtifactSummary[];
  selectedId: string | null;
  onSelectArtifact: (id: string) => void;
  opNameMap: Record<string, string>; // opId → display name
}

function formatSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  const kb = bytes / 1024;
  if (kb < 1024) return `${kb.toFixed(1)} KB`;
  const mb = kb / 1024;
  return `${mb.toFixed(1)} MB`;
}

function artifactIcon(contentType: string) {
  if (contentType === 'text/html') return <HtmlIcon fontSize="small" />;
  if (contentType === 'application/json') return <DataObjectIcon fontSize="small" />;
  if (contentType.startsWith('image/')) return <ImageIcon fontSize="small" />;
  if (contentType.startsWith('text/')) return <DescriptionIcon fontSize="small" />;
  return <InsertDriveFileIcon fontSize="small" />;
}

function KindChip({ kind }: { kind: string }) {
  return (
    <Box
      component="span"
      sx={{
        fontSize: '0.7rem',
        px: 0.75,
        py: 0.25,
        borderRadius: 1,
        bgcolor: 'action.hover',
        fontFamily: 'monospace',
        whiteSpace: 'nowrap',
      }}
    >
      {kind}
    </Box>
  );
}

export function ArtifactTable({ artifacts, selectedId, onSelectArtifact, opNameMap }: ArtifactTableProps) {
  return (
    <TableContainer>
      <Table size="small" sx={{ minWidth: 600 }}>
        <TableHead>
          <TableRow>
            <TableCell sx={{ fontWeight: 600, width: '35%' }}>Name</TableCell>
            <TableCell sx={{ fontWeight: 600, width: '25%' }}>Op</TableCell>
            <TableCell sx={{ fontWeight: 600, width: '15%' }}>Kind</TableCell>
            <TableCell sx={{ fontWeight: 600, width: '10%' }}>Size</TableCell>
            <TableCell sx={{ fontWeight: 600, width: '15%', textAlign: 'right' }}>Actions</TableCell>
          </TableRow>
        </TableHead>
        <TableBody>
          {artifacts.map((artifact) => (
            <TableRow
              key={artifact.id}
              hover
              selected={artifact.id === selectedId}
              onClick={() => onSelectArtifact(artifact.id)}
              sx={{
                cursor: 'pointer',
                '&:last-child td, &:last-child th': { border: 0 },
              }}
            >
              {/* Name */}
              <TableCell>
                <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, overflow: 'hidden' }}>
                  <Box sx={{ color: 'text.disabled', flexShrink: 0 }}>
                    {artifactIcon(artifact.contentType)}
                  </Box>
                  <Tooltip title={artifact.name} placement="top-start">
                    <Typography
                      variant="body2"
                      sx={{
                        fontFamily: 'monospace',
                        overflow: 'hidden',
                        textOverflow: 'ellipsis',
                        whiteSpace: 'nowrap',
                      }}
                    >
                      {artifact.name}
                    </Typography>
                  </Tooltip>
                </Box>
              </TableCell>

              {/* Op */}
              <TableCell>
                <Tooltip title={artifact.opID} placement="top-start">
                  <Typography
                    variant="caption"
                    sx={{
                      fontFamily: 'monospace',
                      color: 'primary.main',
                      overflow: 'hidden',
                      textOverflow: 'ellipsis',
                      whiteSpace: 'nowrap',
                      display: 'block',
                      maxWidth: '100%',
                    }}
                  >
                    {opNameMap[artifact.opID] ?? artifact.opID}
                  </Typography>
                </Tooltip>
              </TableCell>

              {/* Kind */}
              <TableCell>
                <KindChip kind={artifact.kind} />
              </TableCell>

              {/* Size */}
              <TableCell>
                <Typography variant="caption" color="text.secondary">
                  {formatSize(artifact.size)}
                </Typography>
              </TableCell>

              {/* Actions */}
              <TableCell align="right" onClick={(e) => e.stopPropagation()}>
                <Tooltip title="Preview">
                  <IconButton
                    size="small"
                    onClick={() => onSelectArtifact(artifact.id)}
                    color={artifact.id === selectedId ? 'primary' : 'default'}
                  >
                    <OpenInBrowserIcon fontSize="small" />
                  </IconButton>
                </Tooltip>
                <Tooltip title="Download">
                  <IconButton
                    size="small"
                    component="a"
                    href={`/api/v1/artifacts/${artifact.id}`}
                    target="_blank"
                    rel="noopener noreferrer"
                    download
                  >
                    <CloudDownloadIcon fontSize="small" />
                  </IconButton>
                </Tooltip>
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </TableContainer>
  );
}
