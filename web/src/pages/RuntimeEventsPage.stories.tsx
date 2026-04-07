import type { Meta, StoryObj } from '@storybook/react-vite';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import { Provider } from 'react-redux';
import { configureStore } from '@reduxjs/toolkit';
import { RuntimeEventsPage } from './RuntimeEventsPage';
import { runtimeEventsApi } from '../api/runtimeEventsApi';
import { uiSlice } from '../store/uiSlice';

function createStoryStore() {
  return configureStore({
    reducer: {
      ui: uiSlice.reducer,
      [runtimeEventsApi.reducerPath]: runtimeEventsApi.reducer,
    },
    middleware: (getDefault) => getDefault().concat(runtimeEventsApi.middleware),
  });
}

const meta: Meta<typeof RuntimeEventsPage> = {
  title: 'Pages/RuntimeEventsPage',
  component: RuntimeEventsPage,
  decorators: [
    (Story) => (
      <Provider store={createStoryStore()}>
        <MemoryRouter initialEntries={['/events']}>
          <Routes>
            <Route path="/events" element={<Story />} />
          </Routes>
        </MemoryRouter>
      </Provider>
    ),
  ],
};

export default meta;
type Story = StoryObj<typeof RuntimeEventsPage>;

export const Default: Story = {};
