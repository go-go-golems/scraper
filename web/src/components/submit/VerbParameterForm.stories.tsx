import type { Meta, StoryObj } from '@storybook/react';
import { VerbParameterForm } from './VerbParameterForm';
import type { VerbSummary } from '../../api/types';

const meta: Meta<typeof VerbParameterForm> = {
  title: 'Submit/VerbParameterForm',
  component: VerbParameterForm,
};

export default meta;
type Story = StoryObj<typeof VerbParameterForm>;

const hackernewsSeed: VerbSummary = {
  name: 'seed',
  fullPath: 'seed',
  commandPath: 'site hackernews run seed',
  functionName: 'seed',
  short: 'Submit the hackernews seed workflow',
  long: '',
  sections: [
    {
      slug: 'default',
      title: '',
      description: '',
      fields: [
        {
          name: 'base-url',
          type: 'string',
          help: 'Starting URL for the scraper',
          default: 'https://news.ycombinator.com/',
          required: false,
        },
        {
          name: 'max-pages',
          type: 'int',
          help: 'Maximum number of pages to scrape',
          default: 1,
          required: false,
        },
      ],
    },
  ],
};

const jsDemoSeed: VerbSummary = {
  name: 'seed',
  fullPath: 'seed',
  commandPath: 'site js-demo run seed',
  functionName: 'seed',
  short: 'Submit the js-demo seed workflow',
  long: '',
  sections: [
    {
      slug: 'default',
      title: '',
      description: '',
      fields: [
        {
          name: 'count',
          type: 'int',
          help: 'Number of items to generate',
          default: 10,
          required: true,
        },
        {
          name: 'multiplier',
          type: 'float',
          help: 'Score multiplier applied to each result',
          default: 1.5,
          required: false,
        },
        {
          name: 'prefix',
          type: 'string',
          help: 'Prefix for generated item names',
          default: 'item',
          required: false,
        },
      ],
    },
  ],
};

const nerevalSeed: VerbSummary = {
  name: 'seed',
  fullPath: 'seed',
  commandPath: 'site nereval run seed',
  functionName: 'seed',
  short: 'Submit the nereval seed workflow',
  long: '',
  sections: [
    {
      slug: 'location',
      title: 'Location',
      description: 'Geographic target for NER evaluation',
      fields: [
        {
          name: 'town',
          type: 'string',
          help: 'Town name for evaluation corpus',
          default: 'Springfield',
          required: true,
          choices: ['Springfield', 'Portland', 'Franklin', 'Clinton'],
        },
      ],
    },
    {
      slug: 'crawl',
      title: 'Crawl settings',
      description: '',
      fields: [
        {
          name: 'base_url',
          type: 'string',
          help: 'Base URL for the crawl target',
          default: 'https://example.com/',
          required: true,
        },
        {
          name: 'max_pages',
          type: 'int',
          help: 'Maximum pages to crawl',
          default: 5,
          required: false,
        },
      ],
    },
  ],
};

const emptyVerb: VerbSummary = {
  name: 'ping',
  fullPath: 'ping',
  commandPath: 'site hackernews run ping',
  functionName: 'ping',
  short: 'Health check with no parameters',
  long: '',
  sections: [
    {
      slug: 'default',
      title: '',
      description: '',
      fields: [],
    },
  ],
};

export const HackernewsSeed: Story = {
  args: {
    verb: hackernewsSeed,
    values: {},
    onChange: () => {},
  },
};

export const JsDemoSeed: Story = {
  args: {
    verb: jsDemoSeed,
    values: {},
    onChange: () => {},
  },
};

export const NerevalSeed: Story = {
  args: {
    verb: nerevalSeed,
    values: {},
    onChange: () => {},
  },
};

export const EmptyFields: Story = {
  args: {
    verb: emptyVerb,
    values: {},
    onChange: () => {},
  },
};
