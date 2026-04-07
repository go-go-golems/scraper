import type { Preview } from '@storybook/react-vite';
import React from 'react';
import { ThemeProvider, CssBaseline } from '@mui/material';
import { Provider } from 'react-redux';
import { configureStore } from '@reduxjs/toolkit';
import { initialize, mswLoader } from 'msw-storybook-addon';
import { theme } from '../src/theme';
import { uiSlice } from '../src/store/uiSlice';
import { runtimeEventsApi } from '../src/api/runtimeEventsApi';
import { workflowApi } from '../src/api/workflowApi';
import { catalogApi } from '../src/api/catalogApi';
import { engineApi } from '../src/api/engineApi';
import { queueApi } from '../src/api/queueApi';
import { submissionApi } from '../src/api/submissionApi';

initialize({ onUnhandledRequest: 'bypass' });

function createMockStore() {
  return configureStore({
    reducer: {
      ui: uiSlice.reducer,
      [runtimeEventsApi.reducerPath]: runtimeEventsApi.reducer,
      [workflowApi.reducerPath]: workflowApi.reducer,
      [catalogApi.reducerPath]: catalogApi.reducer,
      [engineApi.reducerPath]: engineApi.reducer,
      [queueApi.reducerPath]: queueApi.reducer,
      [submissionApi.reducerPath]: submissionApi.reducer,
    },
    middleware: (getDefault) =>
      getDefault({ serializableCheck: false })
        .concat(runtimeEventsApi.middleware)
        .concat(workflowApi.middleware)
        .concat(catalogApi.middleware)
        .concat(engineApi.middleware)
        .concat(queueApi.middleware)
        .concat(submissionApi.middleware),
  });
}

const preview: Preview = {
  decorators: [
    (Story) => (
      <Provider store={createMockStore()}>
        <ThemeProvider theme={theme}>
          <CssBaseline />
          <Story />
        </ThemeProvider>
      </Provider>
    ),
  ],
  parameters: {
    controls: {
      matchers: {
        color: /(background|color)$/i,
        date: /Date$/i,
      },
    },
    a11y: {
      test: 'todo',
    },
  },
  loaders: [mswLoader],
};

export default preview;
