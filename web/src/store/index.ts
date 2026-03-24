import { configureStore } from '@reduxjs/toolkit';
import { engineApi } from '../api/engineApi';
import { workflowApi } from '../api/workflowApi';
import { queueApi } from '../api/queueApi';
import { catalogApi } from '../api/catalogApi';
import { submissionApi } from '../api/submissionApi';
import { uiSlice } from './uiSlice';

export const store = configureStore({
  reducer: {
    ui: uiSlice.reducer,
    [engineApi.reducerPath]: engineApi.reducer,
    [workflowApi.reducerPath]: workflowApi.reducer,
    [queueApi.reducerPath]: queueApi.reducer,
    [catalogApi.reducerPath]: catalogApi.reducer,
    [submissionApi.reducerPath]: submissionApi.reducer,
  },
  middleware: (getDefaultMiddleware) =>
    getDefaultMiddleware()
      .concat(engineApi.middleware)
      .concat(workflowApi.middleware)
      .concat(queueApi.middleware)
      .concat(catalogApi.middleware)
      .concat(submissionApi.middleware),
});

export type RootState = ReturnType<typeof store.getState>;
export type AppDispatch = typeof store.dispatch;
