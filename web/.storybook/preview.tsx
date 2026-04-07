import type { Preview } from '@storybook/react-vite';
import React from 'react';
import { ThemeProvider, CssBaseline } from '@mui/material';
import { Provider } from 'react-redux';
import { configureStore } from '@reduxjs/toolkit';
import { theme } from '../src/theme';
import { uiSlice } from '../src/store/uiSlice';
import { runtimeEventsApi } from '../src/api/runtimeEventsApi';

function createMockStore() {
  return configureStore({
    reducer: {
      ui: uiSlice.reducer,
      [runtimeEventsApi.reducerPath]: runtimeEventsApi.reducer,
    },
    middleware: (getDefault) =>
      getDefault().concat(runtimeEventsApi.middleware),
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
};

export default preview;
