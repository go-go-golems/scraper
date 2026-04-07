import type { Meta, StoryObj } from '@storybook/react';
import { ArtifactPreviewPanel } from './ArtifactPreviewPanel';
import { defaultArtifactHandlers } from '../../stories/msw/handlers';
import { makeArtifact } from '../../stories/msw/handlers';

const meta: Meta<typeof ArtifactPreviewPanel> = {
  title: 'Artifacts/ArtifactPreviewPanel',
  component: ArtifactPreviewPanel,
  parameters: {
    msw: { handlers: defaultArtifactHandlers },
  },
};
export default meta;
type Story = StoryObj<typeof ArtifactPreviewPanel>;

export const NoArtifactSelected: Story = {
  name: 'No artifact selected',
  args: {
    artifact: null,
    onClose: () => {},
  },
};

export const JsonArtifact: Story = {
  name: 'JSON artifact',
  args: {
    artifact: makeArtifact({
      id: 'story-art-json',
      name: 'summary.json',
    }),
    onClose: () => {},
  },
};

export const HtmlArtifact: Story = {
  name: 'HTML artifact',
  args: {
    artifact: makeArtifact({
      id: 'story-art-html',
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

export const LogArtifact: Story = {
  name: 'Log artifact',
  args: {
    artifact: makeArtifact({
      id: 'story-art-log',
      name: 'debug.log',
      kind: 'exec-log',
      contentType: 'text/plain',
      size: 12_800,
    }),
    onClose: () => {},
  },
};

export const BinaryArtifact: Story = {
  name: 'Binary artifact (non-previewable)',
  args: {
    artifact: makeArtifact({
      id: 'story-art-bin',
      name: 'response.bin',
      kind: 'raw',
      contentType: 'application/octet-stream',
      size: 2_097_152,
      previewable: false,
    }),
    onClose: () => {},
  },
};
