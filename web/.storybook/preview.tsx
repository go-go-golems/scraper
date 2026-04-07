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

// MSW is lazy-loaded so it doesn't break vitest (which runs in Node/Playwright)
// where the service worker doesn't exist.
let mswLoader: (() => Promise<void>) | undefined;

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
  loaders: [
    // Defer MSW loader to runtime so it's only loaded in interactive Storybook
    async () => {
      if (!mswLoader) {
        try {
          const mswAddon = await import('msw-storybook-addon');
          mswAddon.initialize({ onUnhandledRequest: 'bypass' });
          mswLoader = mswAddon.mswLoader;
        } catch {
          // MSW not available (e.g., in vitest)
        }
      }
      if (mswLoader) {
        return mswLoader();
      }
    },
  ],
};

export default preview;
