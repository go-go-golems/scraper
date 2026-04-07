import { Chip, Stack } from '@mui/material';
import type { RuntimeEventV1 } from '../../../pb/proto/scraper/runtime/v1/events_pb';
import { RuntimeEventTable } from '../RuntimeEventTable';
import type { ConnectionState } from './helpers';

interface OpRuntimeTabProps {
  events: RuntimeEventV1[];
  loading: boolean;
  connectionState: ConnectionState;
}

export function OpRuntimeTab({
  events,
  loading,
  connectionState,
}: OpRuntimeTabProps) {
  return (
    <>
      <Stack direction="row" spacing={1} sx={{ mb: 1.5 }} flexWrap="wrap" useFlexGap>
        <Chip label={`Stream: ${connectionState}`} size="small" variant="outlined" />
        <Chip label={`${events.length} events`} size="small" variant="outlined" />
      </Stack>
      <RuntimeEventTable
        events={events}
        loading={loading}
        dense
        emptyMessage="No runtime events for this op yet."
      />
    </>
  );
}
