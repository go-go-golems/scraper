import type { Meta, StoryObj } from '@storybook/react';
import { http, HttpResponse } from 'msw';
import { ScriptViewer } from './ScriptViewer';

const meta: Meta<typeof ScriptViewer> = {
  title: 'Scripts/ScriptViewer',
  component: ScriptViewer,
};
export default meta;
type Story = StoryObj<typeof ScriptViewer>;

const SAMPLE_SCRIPT = `// extract.js — scrapes Hacker News frontpage stories
function extract({ page }) {
  const items = page.querySelectorAll('.athing');
  return items.map(item => {
    const titleEl = item.querySelector('.titleline a');
    const scoreEl = item.querySelector('.score');
    const byEl = item.querySelector('.hnuser');
    return {
      url:     titleEl?.href ?? '',
      title:   titleEl?.textContent ?? '',
      score:   parseInt(scoreEl?.textContent ?? '0', 10),
      author:  byEl?.textContent ?? '',
    };
  });
}

async function paginate({ page, goto }) {
  const nextLink = page.querySelector('.morelink a');
  if (!nextLink) return null;
  return goto(nextLink.href);
}

export { extract, paginate };
`;

export const Default: Story = {
  name: 'Default (JavaScript)',
  parameters: {
    msw: {
      handlers: [
        http.get('/api/v1/catalog/sites/hackernews/scripts/extract.js', () =>
          HttpResponse.text(SAMPLE_SCRIPT, { headers: { 'Content-Type': 'text/javascript' } }),
        ),
      ],
    },
  },
  args: {
    source: SAMPLE_SCRIPT,
    filename: 'extract.js',
  },
};

export const LongScript: Story = {
  name: 'Long script',
  args: {
    source: `// seed.js — initializes the scraper environment
function seed({ browser, goto }) {
  const page = browser.newPage();
  goto('https://news.ycombinator.com');
  page.setViewport({ width: 1280, height: 800 });
  page.setExtraHTTPHeaders({ 'Accept-Language': 'en-US' });
  return { page };
}

// check-duplicate.js — deduplicates based on URL
function checkDuplicate({ db, record }) {
  const existing = db.query(
    'SELECT id FROM items WHERE url = ?',
    [record.Data.url]
  );
  return existing.length === 0;
}

// persist.js — writes to the site database
function persist({ db, record }) {
  if (!record.Data.url) return;
  db.run(
    'INSERT OR IGNORE INTO items(url, title, score, author) VALUES(?, ?, ?, ?)',
    [record.Data.url, record.Data.title, record.Data.score, record.Data.author]
  );
}

export { seed, checkDuplicate, persist };`,
    filename: 'seed.js',
  },
};
