import type { Meta, StoryObj } from '@storybook/react';
import { ResultsPanel } from './ResultsPanel';
import { defaultResultsHandlers, emptyResultsHandlers, STORY_WORKFLOW_ID } from '../../stories/msw/handlers';

const meta: Meta<typeof ResultsPanel> = {
  title: 'Results/ResultsPanel',
  component: ResultsPanel,
  parameters: {
    layout: 'fullscreen',
    msw: { handlers: defaultResultsHandlers },
  },
  tags: ['autodocs'],
};
export default meta;
type Story = StoryObj<typeof ResultsPanel>;

export const Default: Story = {
  name: 'Default (with results)',
  args: {
    workflowId: STORY_WORKFLOW_ID,
    onOpClick: () => {},
  },
};

export const Empty: Story = {
  name: 'Empty (no results)',
  parameters: {
    msw: { handlers: emptyResultsHandlers },
  },
  args: {
    workflowId: STORY_WORKFLOW_ID,
    onOpClick: () => {},
  },
};
