import type { Meta, StoryObj } from '@storybook/react';
import { ArtifactList } from './ArtifactList';
import { createArtifactSummary } from '../../stories/__fixtures__/factories';

const meta: Meta<typeof ArtifactList> = {
  title: 'Artifacts/ArtifactList',
  component: ArtifactList,
  args: {
    selectedId: null,
    onSelect: () => {},
  },
};

export default meta;
type Story = StoryObj<typeof ArtifactList>;

export const Default: Story = {
  args: {
    artifacts: [
      createArtifactSummary({ id: 'art-001', name: 'page.html', contentType: 'text/html', kind: 'page', size: 24_320 }),
      createArtifactSummary({ id: 'art-002', name: 'data.json', contentType: 'application/json', kind: 'extract', size: 5_128 }),
    ],
  },
};

export const Single: Story = {
  args: {
    artifacts: [
      createArtifactSummary({ id: 'art-001', name: 'result.json', contentType: 'application/json', kind: 'extract', size: 1_024 }),
    ],
  },
};

export const Empty: Story = {
  args: {
    artifacts: [],
  },
};

export const Many: Story = {
  args: {
    artifacts: [
      createArtifactSummary({ id: 'art-001', name: 'index.html', contentType: 'text/html', kind: 'page', size: 45_200 }),
      createArtifactSummary({ id: 'art-002', name: 'listing.json', contentType: 'application/json', kind: 'extract', size: 12_800 }),
      createArtifactSummary({ id: 'art-003', name: 'debug.log', contentType: 'text/plain', kind: 'log', size: 3_400 }),
      createArtifactSummary({ id: 'art-004', name: 'screenshot.png', contentType: 'image/png', kind: 'screenshot', size: 1_048_576 }),
      createArtifactSummary({ id: 'art-005', name: 'detail.html', contentType: 'text/html', kind: 'page', size: 31_744 }),
      createArtifactSummary({ id: 'art-006', name: 'items.json', contentType: 'application/json', kind: 'extract', size: 8_192 }),
      createArtifactSummary({ id: 'art-007', name: 'headers.txt', contentType: 'text/plain', kind: 'metadata', size: 512 }),
      createArtifactSummary({ id: 'art-008', name: 'response.bin', contentType: 'application/octet-stream', kind: 'raw', size: 2_097_152 }),
    ],
  },
};
