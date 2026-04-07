import type { Meta, StoryObj } from '@storybook/react';
import type { ArtifactSummary } from '../../api/types';
import { ArtifactPreviewPanel } from './ArtifactPreviewPanel';

const meta: Meta<typeof ArtifactPreviewPanel> = {
  title: 'Artifacts/ArtifactPreviewPanel',
  component: ArtifactPreviewPanel,
};
export default meta;
type Story = StoryObj<typeof ArtifactPreviewPanel>;

function makeArtifact(overrides: Partial<ArtifactSummary> = {}): ArtifactSummary {
  return {
    id: 'art-1',
    opID: 'wf-1:frontpage-extract',
    workflowID: 'wf-1',
    name: 'summary.json',
    kind: 'json-output',
    contentType: 'application/json',
    size: 2_048,
    createdAt: new Date(Date.now() - 3600_000).toISOString(),
    previewable: true,
    previewKind: 'json',
    ...overrides,
  };
}

export const Empty: Story = {
  name: 'No artifact selected',
  args: {
    artifact: null,
    onClose: () => {},
  },
};

export const JsonArtifact: Story = {
  name: 'JSON artifact (previewable)',
  args: {
    artifact: makeArtifact(),
    onClose: () => {},
    onNavigateToOp: () => {},
  },
  loaders: [
    async () => ({
      mockBody: JSON.stringify({ stories: 30, nextPage: '/news?p=2', scrapedAt: new Date().toISOString() }, null, 2),
    }),
  ],
};

export const HtmlArtifact: Story = {
  name: 'HTML artifact',
  args: {
    artifact: makeArtifact({
      id: 'art-2',
      name: 'index.html',
      kind: 'http-response-body',
      contentType: 'text/html',
      size: 48_320,
      previewable: true,
      previewKind: 'html',
    }),
    onClose: () => {},
  },
};

export const BinaryArtifact: Story = {
  name: 'Binary artifact (non-previewable)',
  args: {
    artifact: makeArtifact({
      id: 'art-3',
      name: 'response.bin',
      kind: 'raw',
      contentType: 'application/octet-stream',
      size: 2_097_152,
      previewable: false,
    }),
    onClose: () => {},
  },
};
