import { createSlice } from '@reduxjs/toolkit';
import type { PayloadAction } from '@reduxjs/toolkit';

interface RecentSubmission {
  timestamp: string;
  site: string;
  verb: string;
  workflowId: string;
}

interface UiState {
  workflowFilters: {
    site: string;
    status: string;
  };
  submitForm: {
    selectedSite: string | null;
    selectedVerb: string | null;
    fieldValues: Record<string, unknown>;
  };
  recentSubmissions: RecentSubmission[];
}

const initialState: UiState = {
  workflowFilters: { site: '', status: '' },
  submitForm: { selectedSite: null, selectedVerb: null, fieldValues: {} },
  recentSubmissions: [],
};

export const uiSlice = createSlice({
  name: 'ui',
  initialState,
  reducers: {
    setWorkflowFilters: (state, action: PayloadAction<{ site?: string; status?: string }>) => {
      if (action.payload.site !== undefined) state.workflowFilters.site = action.payload.site;
      if (action.payload.status !== undefined) state.workflowFilters.status = action.payload.status;
    },
    setSelectedSite: (state, action: PayloadAction<string | null>) => {
      state.submitForm.selectedSite = action.payload;
      state.submitForm.selectedVerb = null;
      state.submitForm.fieldValues = {};
    },
    setSelectedVerb: (state, action: PayloadAction<string | null>) => {
      state.submitForm.selectedVerb = action.payload;
      state.submitForm.fieldValues = {};
    },
    setFieldValue: (state, action: PayloadAction<{ field: string; value: unknown }>) => {
      state.submitForm.fieldValues[action.payload.field] = action.payload.value;
    },
    addRecentSubmission: (state, action: PayloadAction<RecentSubmission>) => {
      state.recentSubmissions.unshift(action.payload);
      if (state.recentSubmissions.length > 20) state.recentSubmissions.pop();
    },
  },
});

export const {
  setWorkflowFilters,
  setSelectedSite,
  setSelectedVerb,
  setFieldValue,
  addRecentSubmission,
} = uiSlice.actions;
