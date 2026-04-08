import { CodeViewPanel } from '../../common/CodeViewPanel';

interface OpInputTabProps {
  input: unknown;
}

export function OpInputTab({ input }: OpInputTabProps) {
  return (
    <CodeViewPanel
      data={input}
      label="Input"
      defaultFormat="yaml"
      formats={['json', 'yaml']}
      maxHeight={500}
    />
  );
}
