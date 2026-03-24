import { Alert, Box, Skeleton } from '@mui/material';
import { ScriptViewer } from './ScriptViewer';

interface ScriptTabProps {
  site: string;
  scriptPath: string;
  source: string | null;
  loading: boolean;
  error: string | null;
}

export function ScriptTab({
  site: _site,
  scriptPath,
  source,
  loading,
  error,
}: ScriptTabProps) {
  if (loading) {
    return (
      <Box>
        <Skeleton variant="text" width={160} sx={{ mb: 1 }} />
        <Skeleton variant="rectangular" height={300} sx={{ borderRadius: 1 }} />
      </Box>
    );
  }

  if (error) {
    return <Alert severity="error">{error}</Alert>;
  }

  if (source !== null) {
    return <ScriptViewer source={source} filename={scriptPath} />;
  }

  return null;
}
