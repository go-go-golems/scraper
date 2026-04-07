import { ScriptTab } from '../../scripts/ScriptTab';

interface OpScriptTabProps {
  site: string;
  scriptPath: string;
  source: string | null;
  loading: boolean;
}

export function OpScriptTab({ site, scriptPath, source, loading }: OpScriptTabProps) {
  return (
    <ScriptTab
      site={site}
      scriptPath={scriptPath}
      source={source}
      loading={loading}
      error={null}
    />
  );
}
