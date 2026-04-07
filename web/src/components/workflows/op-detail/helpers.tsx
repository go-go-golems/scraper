import CodeIcon from '@mui/icons-material/Code';
import HttpIcon from '@mui/icons-material/Http';

/**
 * Shared helpers for the op-detail tab subcomponents.
 */

export type ConnectionState = 'connecting' | 'live' | 'error' | 'closed';

export function connectionColor(
  state: ConnectionState,
): 'disabled' | 'success' | 'warning' | 'error' {
  switch (state) {
    case 'live':
      return 'success';
    case 'connecting':
      return 'warning';
    case 'error':
      return 'error';
    default:
      return 'disabled';
  }
}

export function KindIcon({ kind }: { kind: string }) {
  if (kind === 'js') return <CodeIcon fontSize="small" sx={{ mr: 0.5 }} />;
  if (kind === 'http' || kind === 'http/fetch')
    return <HttpIcon fontSize="small" sx={{ mr: 0.5 }} />;
  return null;
}
