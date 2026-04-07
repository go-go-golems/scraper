import type { Meta, StoryObj } from '@storybook/react';
import type { ArtifactSummary } from '../../api/types';
import { ArtifactTable } from './ArtifactTable';

const meta: Meta<typeof ArtifactTable> = {
  title: 'Artifacts/ArtifactTable',
  component: ArtifactTable,
};
export default meta;
type Story = StoryObj<typeof ArtifactTable>;

const OP_NAME_MAP = {
  'wf-1:frontpage-fetch': 'http:frontpage-fetch',
  'wf-1:frontpage-extract': 'js:frontpage-extract',
  'wf-1:item-fetch': 'http:item-fetch',
};

function makeArtifact(overrides: Partial<ArtifactSummary> = {}): ArtifactSummary {
  return {
    id: 'art-1',
    opID: 'wf-1:frontpage-fetch',
    workflowID: 'wf-1',
    name: 'index.html',
    kind: 'http-response-body',
    contentType: 'text/html',
    size: 48_320,
    createdAt: new Date().toISOString(),
    previewable: true,
    previewKind: 'html',
    ...overrides,
  };
}

export const Default: Story = {
  args: {
    artifacts: [
      makeArtifact({ id: 'art-1', name: 'index.html', kind: 'http-response-body', contentType: 'text/html', size: 48_320 }),
      makeArtifact({ id: 'art-2', name: 'summary.json', kind: 'json-output', contentType: 'application/json', size: 2_048 }),
      makeArtifact({ id: 'art-3', name: 'debug.log', kind: 'exec-log', contentType: 'text/plain', size: 12_800 }),
      makeArtifact({ id: 'art-4', name: 'screenshot.png', kind: 'screenshot', contentType: 'image/png', size: 1_048_576 }),
    ],
    selectedId: null,
    onSelectArtifact: () => {},
    opNameMap: OP_NAME_MAP,
  },
};

export const WithSelection: Story = {
  name: 'With selection',
  args: {
    ...Default.args,
    selectedId: 'art-2',
  },
};

export const ManyRows: Story = {
  name: 'Many rows',
  args: {
    artifacts: Array.from({ length: 15 }, (_, i) =>
      makeArtifact({
        id: `art-${i}`,
        name: `artifact-${i}.json`,
        kind: i % 3 === 0 ? 'json-output' : i % 3 === 1 ? 'http-response-body' : 'exec-log',
        contentType: i % 3 === 0 ? 'application/json' : i % 3 === 1 ? 'text/html' : 'text/plain',
        size: (i + 1) * 1024,
      }),
    ),
    selectedId: null,
    onSelectArtifact: () => {},
    opNameMap: OP_NAME_MAP,
  },
};

export const Empty: Story = {
  name: 'Empty',
  args: {
    artifacts: [],
    selectedId: null,
    onSelectArtifact: () => {},
    opNameMap: OP_NAME_MAP,
  },
};
