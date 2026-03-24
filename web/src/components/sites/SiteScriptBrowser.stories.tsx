import type { Meta, StoryObj } from '@storybook/react';
import { Provider } from 'react-redux';
import { configureStore } from '@reduxjs/toolkit';
import { SiteScriptBrowser } from './SiteScriptBrowser';
import { catalogApi } from '../../api/catalogApi';
import { uiSlice } from '../../store/uiSlice';

function createStoryStore() {
  return configureStore({
    reducer: {
      ui: uiSlice.reducer,
      [catalogApi.reducerPath]: catalogApi.reducer,
    },
    middleware: (getDefault) => getDefault().concat(catalogApi.middleware),
  });
}

const meta: Meta<typeof SiteScriptBrowser> = {
  title: 'Sites/SiteScriptBrowser',
  component: SiteScriptBrowser,
  decorators: [
    (Story) => (
      <Provider store={createStoryStore()}>
        <Story />
      </Provider>
    ),
  ],
};

export default meta;
type Story = StoryObj<typeof SiteScriptBrowser>;

export const Default: Story = {
  args: {
    site: 'hackernews',
    scripts: ['seed.js', 'detail.js', 'export.js'],
  },
};

export const NerevalWithLib: Story = {
  args: {
    site: 'nereval',
    scripts: ['seed.js', 'detail.js', 'lib/utils.js', 'lib/parse.js'],
  },
};
