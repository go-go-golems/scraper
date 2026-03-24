import type { Meta, StoryObj } from '@storybook/react';
import { ArtifactPreview } from './ArtifactPreview';

const meta: Meta<typeof ArtifactPreview> = {
  title: 'Artifacts/ArtifactPreview',
  component: ArtifactPreview,
};

export default meta;
type Story = StoryObj<typeof ArtifactPreview>;

export const Html: Story = {
  args: {
    name: 'page.html',
    contentType: 'text/html',
    content: `<!DOCTYPE html>
<html lang="en">
<head><title>Hacker News</title></head>
<body>
  <table>
    <tr class="athing" id="12345">
      <td class="title"><a href="https://example.com">Show HN: A cool project</a></td>
    </tr>
    <tr>
      <td class="subtext">
        <span class="score">142 points</span> by user1
      </td>
    </tr>
  </table>
</body>
</html>`,
  },
};

export const Json: Story = {
  args: {
    name: 'data.json',
    contentType: 'application/json',
    content: JSON.stringify(
      {
        items: [
          { id: 1, title: 'Show HN: A cool project', score: 142, url: 'https://example.com' },
          { id: 2, title: 'Ask HN: Best practices for scraping', score: 89, url: null },
        ],
        nextPage: '/news?p=2',
        scrapedAt: '2026-03-23T14:32:05Z',
      },
      null,
      2,
    ),
  },
};

export const PlainText: Story = {
  args: {
    name: 'debug.log',
    contentType: 'text/plain',
    content: `[2026-03-23T14:31:58Z] Starting JS extraction for hackernews
[2026-03-23T14:31:59Z] Loaded seed.js script (2.4 KB)
[2026-03-23T14:32:00Z] Navigating to https://news.ycombinator.com/
[2026-03-23T14:32:02Z] Page loaded, running extraction
[2026-03-23T14:32:04Z] Found 30 items on page 1
[2026-03-23T14:32:05Z] Extraction complete, emitting 30 child ops`,
  },
};

export const Binary: Story = {
  args: {
    name: 'screenshot.png',
    contentType: 'image/png',
    content: '',
  },
};
