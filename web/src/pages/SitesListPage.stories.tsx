import type { Meta, StoryObj } from '@storybook/react';
import { MemoryRouter } from 'react-router-dom';
import { Provider } from 'react-redux';
import { configureStore } from '@reduxjs/toolkit';
import { SitesListPage } from './SitesListPage';
import { catalogApi } from '../api/catalogApi';
import { uiSlice } from '../store/uiSlice';

function createStoryStore() {
  return configureStore({
    reducer: {
      ui: uiSlice.reducer,
      [catalogApi.reducerPath]: catalogApi.reducer,
    },
    middleware: (getDefault) => getDefault().concat(catalogApi.middleware),
  });
}

const meta: Meta<typeof SitesListPage> = {
  title: 'Pages/SitesListPage',
  component: SitesListPage,
  decorators: [
    (Story) => (
      <Provider store={createStoryStore()}>
        <MemoryRouter initialEntries={['/sites']}>
          <Story />
        </MemoryRouter>
      </Provider>
    ),
  ],
};

export default meta;
type Story = StoryObj<typeof SitesListPage>;

export const Default: Story = {};
