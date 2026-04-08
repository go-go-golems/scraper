import type { Meta, StoryObj } from '@storybook/react';
import { CodeViewPanel } from './CodeViewPanel';

const meta: Meta<typeof CodeViewPanel> = {
  title: 'Common/CodeViewPanel',
  component: CodeViewPanel,
};
export default meta;
type Story = StoryObj<typeof CodeViewPanel>;

const SAMPLE_DATA = {
  stories: [
    { id: 12345, url: 'https://example.com/1', title: 'Show HN: A cool project', score: 142 },
    { id: 12346, url: 'https://example.com/2', title: 'Another story', score: 89 },
    { id: 12347, url: 'https://example.com/3', title: 'Yet another story', score: 56 },
  ],
  nextPage: '/news?p=2',
  scrapedAt: '2026-04-07T14:32:05Z',
  site: 'hackernews',
  recordCount: 3,
};

export const YamlDefault: Story = {
  name: 'YAML (default)',
  args: {
    data: SAMPLE_DATA,
    label: 'Result',
    defaultFormat: 'yaml',
    formats: ['json', 'yaml'],
    maxHeight: 400,
  },
};

export const JsonDefault: Story = {
  name: 'JSON (default)',
  args: {
    data: SAMPLE_DATA,
    label: 'Result',
    defaultFormat: 'json',
    formats: ['json', 'yaml'],
    maxHeight: 400,
  },
};

export const JsonOnly: Story = {
  name: 'JSON only',
  args: {
    data: SAMPLE_DATA,
    label: 'Result',
    formats: ['json'],
    maxHeight: 300,
  },
};

export const YamlOnly: Story = {
  name: 'YAML only (no toggle)',
  args: {
    data: SAMPLE_DATA,
    label: 'Result',
    formats: ['yaml'],
    maxHeight: 300,
  },
};

export const HtmlArtifact: Story = {
  name: 'HTML artifact (HTML highlighting)',
  args: {
    data: `<!DOCTYPE html>\n<html lang="en">\n<head><title>Hacker News</title></head>\n<body>\n<table class="itemlist">\n<tr class="athing" id="12345">\n<td class="title"><a href="https://example.com">Show HN: A cool project</a></td>\n</tr>\n</table>\n</body>\n</html>`,
    label: 'index.html',
    defaultFormat: 'html',
    formats: ['html', 'json', 'yaml'],
    maxHeight: 300,
  },
};

export const NestedData: Story = {
  name: 'Nested data',
  args: {
    data: {
      outer: {
        inner: {
          deep: {
            value: 'hello world',
            numbers: [1, 2, 3],
            flag: true,
            nullVal: null,
          },
        },
      },
    },
    label: 'Config',
    maxHeight: 300,
  },
};
