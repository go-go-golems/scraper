import type { Meta, StoryObj } from '@storybook/react';
import { ScriptViewer } from './ScriptViewer';

const meta: Meta<typeof ScriptViewer> = {
  title: 'Scripts/ScriptViewer',
  component: ScriptViewer,
};

export default meta;
type Story = StoryObj<typeof ScriptViewer>;

const shortSource = Array.from({ length: 20 }, (_, i) =>
  `// line ${i + 1}\nconst x${i + 1} = ${i + 1};`,
).join('\n');

const longSource = Array.from({ length: 100 }, (_, i) =>
  `function step${i + 1}() { return fetch("https://example.com/api/v1/items/" + ${i + 1}); }`,
).join('\n');

export const ShortScript: Story = {
  args: {
    source: shortSource,
    filename: 'seed.js',
  },
};

export const LongScript: Story = {
  args: {
    source: longSource,
    filename: 'crawl.js',
  },
};

export const Empty: Story = {
  args: {
    source: '',
    filename: 'empty.js',
  },
};
