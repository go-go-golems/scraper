import { Box, Typography } from '@mui/material';
import type { ArtifactSummary } from '../../../api/types';
import { ArtifactList } from '../../artifacts/ArtifactList';
import { ArtifactPreview } from '../../artifacts/ArtifactPreview';

interface OpArtifactsTabProps {
  artifacts: ArtifactSummary[];
  artifactBodies: Record<string, string>;
  selectedArtifactId: string | null;
  onSelectArtifact: (id: string | null) => void;
}

export function OpArtifactsTab({
  artifacts,
  artifactBodies,
  selectedArtifactId,
  onSelectArtifact,
}: OpArtifactsTabProps) {
  if (artifacts.length === 0) {
    return (
      <Typography variant="body2" color="text.secondary">
        No artifacts
      </Typography>
    );
  }

  const selectedArtifact = selectedArtifactId
    ? artifacts.find((a) => a.id === selectedArtifactId) ?? null
    : null;

  return (
    <>
      <ArtifactList
        artifacts={artifacts}
        selectedId={selectedArtifactId}
        onSelect={onSelectArtifact}
      />
      {selectedArtifact && artifactBodies[selectedArtifact.id] && (
        <Box sx={{ mt: 2 }}>
          <ArtifactPreview
            content={artifactBodies[selectedArtifact.id]}
            contentType={selectedArtifact.contentType}
            name={selectedArtifact.name}
          />
        </Box>
      )}
    </>
  );
}
