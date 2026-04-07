import type { Meta, StoryObj } from '@storybook/react';
import { OpArtifactsTab } from './OpArtifactsTab';
import { createArtifactSummary } from '../../../stories/__fixtures__/factories';

const meta: Meta<typeof OpArtifactsTab> = {
  title: 'Workflows/OpDetail/OpArtifactsTab',
  component: OpArtifactsTab,
};

export default meta;
type Story = StoryObj<typeof OpArtifactsTab>;

const htmlBody = '<!DOCTYPE html>\n<html>\n<head><title>Hacker News</title></head>\n<body>\n<table>\n  <tr><td>1.</td><td><a href="...">Show HN: Something cool</a></td></tr>\n</table>\n</body>\n</html>';

const jsonBody = '{\n  "Content-Type": "text/html; charset=utf-8",\n  "Server": "nginx"\n}';

export const NoArtifacts: Story = {
  args: {
    artifacts: [],
    artifactBodies: {},
    selectedArtifactId: null,
    onSelectArtifact: () => {},
  },
};

export const SingleArtifact: Story = {
  args: {
    artifacts: [
      createArtifactSummary({ id: 'art-001', name: 'frontpage.html', kind: 'html', contentType: 'text/html', size: 12345 }),
    ],
    artifactBodies: { 'art-001': htmlBody },
    selectedArtifactId: null,
    onSelectArtifact: () => {},
  },
};

export const MultipleArtifacts: Story = {
  args: {
    artifacts: [
      createArtifactSummary({ id: 'art-001', name: 'frontpage.html', kind: 'html', contentType: 'text/html', size: 12345 }),
      createArtifactSummary({ id: 'art-002', name: 'headers.json', kind: 'json', contentType: 'application/json', size: 842 }),
      createArtifactSummary({ id: 'art-003', name: 'links.json', kind: 'json', contentType: 'application/json', size: 2048 }),
    ],
    artifactBodies: {
      'art-001': htmlBody,
      'art-002': jsonBody,
      'art-003': JSON.stringify([{ url: 'https://example.com/1' }, { url: 'https://example.com/2' }], null, 2),
    },
    selectedArtifactId: null,
    onSelectArtifact: () => {},
  },
};

export const ArtifactSelected: Story = {
  args: {
    artifacts: [
      createArtifactSummary({ id: 'art-001', name: 'frontpage.html', kind: 'html', contentType: 'text/html', size: 12345 }),
      createArtifactSummary({ id: 'art-002', name: 'headers.json', kind: 'json', contentType: 'application/json', size: 842 }),
    ],
    artifactBodies: {
      'art-001': htmlBody,
      'art-002': jsonBody,
    },
    selectedArtifactId: 'art-002',
    onSelectArtifact: () => {},
  },
};
