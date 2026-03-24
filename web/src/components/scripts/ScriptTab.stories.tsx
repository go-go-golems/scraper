import type { Meta, StoryObj } from '@storybook/react';
import { ScriptTab } from './ScriptTab';

const meta: Meta<typeof ScriptTab> = {
  title: 'Scripts/ScriptTab',
  component: ScriptTab,
};

export default meta;
type Story = StoryObj<typeof ScriptTab>;

const sampleSource = `import { fetch } from "scraper/http";
import { emit } from "scraper/workflow";

export default async function seed(input) {
  const resp = await fetch(input.baseURL);
  const links = resp.document.querySelectorAll("a.storylink");
  for (const link of links) {
    emit("extract", { url: link.href });
  }
}
`;

export const Loaded: Story = {
  args: {
    site: 'hackernews',
    scriptPath: 'seed.js',
    source: sampleSource,
    loading: false,
    error: null,
  },
};

export const Loading: Story = {
  args: {
    site: 'hackernews',
    scriptPath: 'seed.js',
    source: null,
    loading: true,
    error: null,
  },
};

export const NotFound: Story = {
  args: {
    site: 'hackernews',
    scriptPath: 'missing.js',
    source: null,
    loading: false,
    error: 'Script not found: missing.js',
  },
};
