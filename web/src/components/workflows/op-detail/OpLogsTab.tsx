import { OpExecutionLog, type LogEntry } from '../../logs/OpExecutionLog';

interface OpLogsTabProps {
  entries: LogEntry[];
}

export function OpLogsTab({ entries }: OpLogsTabProps) {
  return <OpExecutionLog entries={entries} loading={false} />;
}
