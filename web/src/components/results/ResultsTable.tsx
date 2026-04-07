import {
  Box,
  Button,
  Link,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Tooltip,
  Typography,
} from '@mui/material';
import OpenInNewIcon from '@mui/icons-material/OpenInNew';
import type { WorkflowResultSummary } from '../../api/types';

interface ResultsTableProps {
  results: WorkflowResultSummary[];
  selectedId: string | null;
  onSelectResult: (opId: string) => void;
  onOpClick: (opId: string) => void;
}

function StatusChip({ status }: { status: string }) {
  const color: Record<string, 'success' | 'error' | 'warning' | 'info' | 'default'> = {
    succeeded: 'success',
    failed: 'error',
    running: 'warning',
    pending: 'info',
    canceled: 'default',
  };
  return (
    <Box
      component="span"
      sx={{
        fontSize: '0.7rem',
        px: 0.75,
        py: 0.25,
        borderRadius: 1,
        bgcolor: `${color[status] ?? 'default'}.light`,
        color: `${color[status] ?? 'default'}.dark`,
        fontFamily: 'monospace',
        whiteSpace: 'nowrap',
        textTransform: 'capitalize',
      }}
    >
      {status}
    </Box>
  );
}

function CountCell({ value, unit }: { value: number; unit: string }) {
  if (value === 0) return <Typography variant="caption" color="text.disabled">—</Typography>;
  return (
    <Typography variant="body2" sx={{ fontFamily: 'monospace' }}>
      {value} <Typography component="span" variant="caption" color="text.secondary">{unit}</Typography>
    </Typography>
  );
}

export function ResultsTable({ results, selectedId, onSelectResult, onOpClick }: ResultsTableProps) {
  return (
    <TableContainer>
      <Table size="small" sx={{ minWidth: 600 }}>
        <TableHead>
          <TableRow>
            <TableCell sx={{ fontWeight: 600, width: '28%' }}>Op</TableCell>
            <TableCell sx={{ fontWeight: 600, width: '12%' }}>Kind</TableCell>
            <TableCell sx={{ fontWeight: 600, width: '12%' }}>Status</TableCell>
            <TableCell sx={{ fontWeight: 600, width: '12%' }}>Records</TableCell>
            <TableCell sx={{ fontWeight: 600, width: '12%' }}>Artifacts</TableCell>
            <TableCell sx={{ fontWeight: 600, width: '12%' }}>Error</TableCell>
            <TableCell sx={{ fontWeight: 600, width: '12%', textAlign: 'right' }}>Actions</TableCell>
          </TableRow>
        </TableHead>
        <TableBody>
          {results.map((result) => (
            <TableRow
              key={result.opID}
              hover
              selected={result.opID === selectedId}
              onClick={() => onSelectResult(result.opID)}
              sx={{
                cursor: 'pointer',
                '&:last-child td, &:last-child th': { border: 0 },
              }}
            >
              {/* Op */}
              <TableCell>
                <Tooltip title={result.opID} placement="top-start">
                  <Link
                    component="button"
                    variant="body2"
                    sx={{
                      fontFamily: 'monospace',
                      color: 'primary.main',
                      overflow: 'hidden',
                      textOverflow: 'ellipsis',
                      whiteSpace: 'nowrap',
                      display: 'block',
                      textAlign: 'left',
                      cursor: 'pointer',
                    }}
                    onClick={(e) => {
                      e.stopPropagation();
                      onOpClick(result.opID);
                    }}
                  >
                    {result.opID.split(':').pop()}
                  </Link>
                </Tooltip>
              </TableCell>

              {/* Kind */}
              <TableCell>
                <Typography variant="caption" sx={{ fontFamily: 'monospace' }}>
                  {result.kind}
                </Typography>
              </TableCell>

              {/* Status */}
              <TableCell>
                <StatusChip status={result.status} />
              </TableCell>

              {/* Records */}
              <TableCell>
                <CountCell value={result.recordCount} unit="records" />
              </TableCell>

              {/* Artifacts */}
              <TableCell>
                <CountCell value={result.artifactCount} unit="artifacts" />
              </TableCell>

              {/* Error */}
              <TableCell>
                {result.error ? (
                  <Tooltip title={`${result.error.Code}: ${result.error.Message}`} placement="top-start">
                    <Typography variant="caption" color="error" sx={{ fontFamily: 'monospace' }}>
                      {result.error.Code}
                    </Typography>
                  </Tooltip>
                ) : (
                  <Typography variant="caption" color="text.disabled">—</Typography>
                )}
              </TableCell>

              {/* Actions */}
              <TableCell align="right" onClick={(e) => e.stopPropagation()}>
                <Tooltip title="Open in drawer">
                  <Button
                    size="small"
                    variant="text"
                    endIcon={<OpenInNewIcon sx={{ fontSize: 14 }} />}
                    onClick={() => onOpClick(result.opID)}
                    sx={{ fontSize: '0.7rem', py: 0.25 }}
                  >
                    Drawer
                  </Button>
                </Tooltip>
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </TableContainer>
  );
}
