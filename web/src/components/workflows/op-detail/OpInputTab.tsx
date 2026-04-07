import { JsonViewer } from '../../common/JsonViewer';

interface OpInputTabProps {
  input: unknown;
}

export function OpInputTab({ input }: OpInputTabProps) {
  return <JsonViewer data={input} maxHeight={500} />;
}
