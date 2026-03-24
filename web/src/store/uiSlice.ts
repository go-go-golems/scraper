import { createSlice } from '@reduxjs/toolkit';
import type { PayloadAction } from '@reduxjs/toolkit';

export type TabId = 'overview' | 'workflows' | 'queues' | 'submit';

interface RecentSubmission {
  timestamp: string;
  site: string;
  verb: string;
  workflowId: string;
}

interface UiState {
  currentTab: TabId;
  workflowFilters: {
    site: string;
    status: string;
  };
  selectedOpId: string | null;
  opDrawerOpen: boolean;
  submitForm: {
    selectedSite: string | null;
    selectedVerb: string | null;
    fieldValues: Record<string, unknown>;
  };
  recentSubmissions: RecentSubmission[];
}

const initialState: UiState = {
  currentTab: 'overview',
  workflowFilters: { site: '', status: '' },
  selectedOpId: null,
  opDrawerOpen: false,
  submitForm: { selectedSite: null, selectedVerb: null, fieldValues: {} },
  recentSubmissions: [],
};

export const uiSlice = createSlice({
  name: 'ui',
  initialState,
  reducers: {
    setTab: (state, action: PayloadAction<TabId>) => {
      state.currentTab = action.payload;
    },
    setWorkflowFilters: (state, action: PayloadAction<{ site?: string; status?: string }>) => {
      if (action.payload.site !== undefined) state.workflowFilters.site = action.payload.site;
      if (action.payload.status !== undefined) state.workflowFilters.status = action.payload.status;
    },
    selectOp: (state, action: PayloadAction<string | null>) => {
      state.selectedOpId = action.payload;
      state.opDrawerOpen = action.payload !== null;
    },
    closeOpDrawer: (state) => {
      state.opDrawerOpen = false;
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
  setTab,
  setWorkflowFilters,
  selectOp,
  closeOpDrawer,
  setSelectedSite,
  setSelectedVerb,
  setFieldValue,
  addRecentSubmission,
} = uiSlice.actions;
