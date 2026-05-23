import type { Meta, StoryObj } from '@storybook/react';
import { ResultsTable } from './ResultsTable';
import { defaultResultsHandlers, STORY_RESULTS, STORY_WORKFLOW_ID } from '../../stories/msw/handlers';

const meta: Meta<typeof ResultsTable> = {
  title: 'Results/ResultsTable',
  component: ResultsTable,
  parameters: {
    msw: { handlers: defaultResultsHandlers },
  },
};
export default meta;
type Story = StoryObj<typeof ResultsTable>;

export const Default: Story = {
  args: {
    results: STORY_RESULTS,
    selectedId: null,
    onSelectResult: () => {},
    onOpClick: () => {},
  },
};

export const WithSelection: Story = {
  name: 'With selection',
  args: {
    results: STORY_RESULTS,
    selectedId: `${STORY_WORKFLOW_ID}:extract`,
    onSelectResult: () => {},
    onOpClick: () => {},
  },
};
