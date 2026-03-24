import type { Meta, StoryObj } from '@storybook/react';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import { Provider } from 'react-redux';
import { configureStore } from '@reduxjs/toolkit';
import { SiteDetailPage } from './SiteDetailPage';
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

const meta: Meta<typeof SiteDetailPage> = {
  title: 'Pages/SiteDetailPage',
  component: SiteDetailPage,
  decorators: [
    (Story) => (
      <Provider store={createStoryStore()}>
        <MemoryRouter initialEntries={['/sites/hackernews']}>
          <Routes>
            <Route path="/sites/:siteName" element={<Story />} />
          </Routes>
        </MemoryRouter>
      </Provider>
    ),
  ],
};

export default meta;
type Story = StoryObj<typeof SiteDetailPage>;

export const Default: Story = {};
