import type { Meta, StoryObj } from '@storybook/react';
import { ArtifactsPanel } from './ArtifactsPanel';
import {
  defaultArtifactHandlers,
  emptyArtifactHandlers,
  STORY_WORKFLOW_ID,
} from '../../stories/msw/handlers';

const meta: Meta<typeof ArtifactsPanel> = {
  title: 'Artifacts/ArtifactsPanel',
  component: ArtifactsPanel,
  parameters: {
    layout: 'fullscreen',
  },
  tags: ['autodocs'],
};
export default meta;
type Story = StoryObj<typeof ArtifactsPanel>;

export const Default: Story = {
  name: 'Default (with artifacts)',
  parameters: {
    msw: { handlers: defaultArtifactHandlers },
  },
  args: {
    workflowId: STORY_WORKFLOW_ID,
  },
};

export const Empty: Story = {
  name: 'Empty (no artifacts)',
  parameters: {
    msw: { handlers: emptyArtifactHandlers },
  },
  args: {
    workflowId: STORY_WORKFLOW_ID,
  },
};
