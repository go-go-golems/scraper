import type { Meta, StoryObj } from '@storybook/react-vite';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import { Provider } from 'react-redux';
import { configureStore } from '@reduxjs/toolkit';
import { RuntimeEventsPage } from './RuntimeEventsPage';
import { runtimeEventsApi } from '../api/runtimeEventsApi';
import { uiSlice } from '../store/uiSlice';
import { generateMockEvents } from '../test-utils/mockRuntimeEvents';

const DEFAULT_PARAMS = { limit: 100 };

function createStoryStore(
  seedEvents?: ReturnType<typeof generateMockEvents>,
) {
  const store = configureStore({
    reducer: {
      ui: uiSlice.reducer,
      [runtimeEventsApi.reducerPath]: runtimeEventsApi.reducer,
    },
    middleware: (getDefault) =>
      getDefault().concat(runtimeEventsApi.middleware),
  });

  if (seedEvents) {
    store.dispatch(
      runtimeEventsApi.util.updateQueryData(
        'getRecentRuntimeEvents',
        DEFAULT_PARAMS,
        (draft) => {
          draft.splice(0, draft.length, ...seedEvents);
        },
      ),
    );
  }

  return store;
}

const meta: Meta<typeof RuntimeEventsPage> = {
  title: 'Pages/RuntimeEventsPage',
  component: RuntimeEventsPage,
  decorators: [
    (Story, { parameters }) => {
      const seedCount = parameters.seedEvents ?? 20;
      const store = createStoryStore(
        parameters.noSeed ? undefined : generateMockEvents(seedCount),
      );
      return (
        <Provider store={store}>
          <MemoryRouter initialEntries={['/events']}>
            <Routes>
              <Route path="/events" element={<Story />} />
            </Routes>
          </MemoryRouter>
        </Provider>
      );
    },
  ],
};

export default meta;
type Story = StoryObj<typeof RuntimeEventsPage>;

export const Default: Story = {};

export const Empty: Story = {
  parameters: { noSeed: true },
};

export const WithManyEvents: Story = {
  parameters: { seedEvents: 60 },
};
