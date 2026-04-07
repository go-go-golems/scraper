import type { Meta, StoryObj } from '@storybook/react';
import { OpScriptTab } from './OpScriptTab';

const meta: Meta<typeof OpScriptTab> = {
  title: 'Workflows/OpDetail/OpScriptTab',
  component: OpScriptTab,
};

export default meta;
type Story = StoryObj<typeof OpScriptTab>;

const sampleSource = `const helpers = require("./lib/frontpage");

module.exports = function(ctx) {
  const input = ctx.input;
  ctx.log("Starting seed workflow");
  // emit fetch ops
  for (let i = 0; i < input.maxPages; i++) {
    ctx.emit({
      kind: "http/fetch",
      queue: "site:hackernews:http",
      input: { request: { url: input.baseURL } }
    });
  }
  ctx.log("Emitting " + input.maxPages + " fetch ops");
  ctx.log("Done");
};`;

export const WithSource: Story = {
  args: {
    site: 'hackernews',
    scriptPath: 'seed.js',
    source: sampleSource,
    loading: false,
  },
};

export const Loading: Story = {
  args: {
    site: 'hackernews',
    scriptPath: 'seed.js',
    source: null,
    loading: true,
  },
};
