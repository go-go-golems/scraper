import {
  Chip,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Typography,
} from '@mui/material';

interface RecentSubmission {
  timestamp: string;
  site: string;
  verb: string;
  workflowId: string;
}

interface RecentSubmissionsTableProps {
  submissions: RecentSubmission[];
}

function formatRelativeTime(dateStr: string): string {
  const now = Date.now();
  const then = new Date(dateStr).getTime();
  const diffMs = now - then;
  const diffSec = Math.floor(diffMs / 1000);

  if (diffSec < 60) return `${diffSec}s ago`;
  const diffMin = Math.floor(diffSec / 60);
  if (diffMin < 60) return `${diffMin}m ago`;
  const diffHr = Math.floor(diffMin / 60);
  if (diffHr < 24) return `${diffHr}h ago`;
  const diffDay = Math.floor(diffHr / 24);
  return `${diffDay}d ago`;
}

export function RecentSubmissionsTable({ submissions }: RecentSubmissionsTableProps) {
  return (
    <TableContainer>
      <Table size="small">
        <TableHead>
          <TableRow>
            <TableCell>Time</TableCell>
            <TableCell>Site</TableCell>
            <TableCell>Verb</TableCell>
            <TableCell>Workflow ID</TableCell>
          </TableRow>
        </TableHead>
        <TableBody>
          {submissions.length === 0 && (
            <TableRow>
              <TableCell colSpan={4} align="center">
                <Typography variant="body2" color="text.disabled" sx={{ py: 2 }}>
                  No submissions yet
                </Typography>
              </TableCell>
            </TableRow>
          )}
          {submissions.map((sub) => (
            <TableRow key={sub.workflowId}>
              <TableCell>
                <Typography variant="caption" color="text.secondary" noWrap>
                  {formatRelativeTime(sub.timestamp)}
                </Typography>
              </TableCell>
              <TableCell>
                <Chip label={sub.site} size="small" variant="outlined" />
              </TableCell>
              <TableCell>
                <Typography variant="body2">{sub.verb}</Typography>
              </TableCell>
              <TableCell>
                <Typography
                  variant="body2"
                  sx={{ fontFamily: 'monospace', fontSize: '0.8rem' }}
                >
                  {sub.workflowId}
                </Typography>
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </TableContainer>
  );
}
