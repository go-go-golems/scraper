import { useCallback, useState } from 'react';
import { useSelector, useDispatch } from 'react-redux';
import {
  Alert,
  Box,
  Button,
  Card,
  CardContent,
  CircularProgress,
  Grid,
  Snackbar,
  Typography,
} from '@mui/material';
import { SitePicker } from '../components/submit/SitePicker';
import { VerbPicker } from '../components/submit/VerbPicker';
import { VerbParameterForm } from '../components/submit/VerbParameterForm';
import { RecentSubmissionsTable } from '../components/submit/RecentSubmissionsTable';
import { useListSitesQuery, useListVerbsQuery } from '../api/catalogApi';
import { useSubmitWorkflowMutation } from '../api/submissionApi';
import {
  setSelectedSite,
  setSelectedVerb,
  setFieldValue,
  addRecentSubmission,
} from '../store/uiSlice';
import type { RootState } from '../store';

export function SubmitWorkflowPage() {
  const dispatch = useDispatch();
  const { selectedSite, selectedVerb, fieldValues } = useSelector(
    (state: RootState) => state.ui.submitForm,
  );
  const recentSubmissions = useSelector((state: RootState) => state.ui.recentSubmissions);

  const { data: sites = [] } = useListSitesQuery();
  const { data: verbs = [], isFetching: verbsLoading } = useListVerbsQuery(selectedSite!, {
    skip: !selectedSite,
  });

  const [submitWorkflow, { isLoading: submitting }] = useSubmitWorkflowMutation();
  const [snackbar, setSnackbar] = useState<{ open: boolean; message: string; severity: 'success' | 'error' }>({
    open: false,
    message: '',
    severity: 'success',
  });

  const activeVerb = verbs.find((v) => v.name === selectedVerb) ?? null;

  const handleSiteSelect = useCallback(
    (site: string) => dispatch(setSelectedSite(site)),
    [dispatch],
  );

  const handleVerbSelect = useCallback(
    (verb: string) => dispatch(setSelectedVerb(verb)),
    [dispatch],
  );

  const handleFieldChange = useCallback(
    (field: string, value: unknown) => dispatch(setFieldValue({ field, value })),
    [dispatch],
  );

  const handleSubmit = useCallback(async () => {
    if (!selectedSite || !selectedVerb) return;

    try {
      const result = await submitWorkflow({
        site: selectedSite,
        verb: selectedVerb,
        values: fieldValues,
      }).unwrap();

      dispatch(
        addRecentSubmission({
          timestamp: new Date().toISOString(),
          site: selectedSite,
          verb: selectedVerb,
          workflowId: result.workflow.ID,
        }),
      );

      setSnackbar({
        open: true,
        message: `Workflow ${result.workflow.ID} submitted (${result.submittedCount} ops)`,
        severity: 'success',
      });
    } catch (err) {
      const message =
        err && typeof err === 'object' && 'data' in err
          ? String((err as { data: { error?: string } }).data?.error ?? 'Submission failed')
          : 'Submission failed';
      setSnackbar({ open: true, message, severity: 'error' });
    }
  }, [selectedSite, selectedVerb, fieldValues, submitWorkflow, dispatch]);

  const handleCloseSnackbar = useCallback(() => {
    setSnackbar((prev) => ({ ...prev, open: false }));
  }, []);

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 3 }}>
      <Typography variant="h6">Submit Workflow</Typography>

      <Grid container spacing={3}>
        {/* Site picker */}
        <Grid size={{ xs: 12, md: 6 }}>
          <Card>
            <CardContent>
              <Typography variant="subtitle2" gutterBottom>
                Site
              </Typography>
              <SitePicker
                sites={sites}
                selected={selectedSite}
                onSelect={handleSiteSelect}
              />
            </CardContent>
          </Card>
        </Grid>

        {/* Verb picker */}
        <Grid size={{ xs: 12, md: 6 }}>
          <Card>
            <CardContent>
              <Typography variant="subtitle2" gutterBottom>
                Verb
              </Typography>
              {selectedSite ? (
                <VerbPicker
                  verbs={verbs}
                  selected={selectedVerb}
                  onSelect={handleVerbSelect}
                  loading={verbsLoading}
                />
              ) : (
                <Typography variant="body2" color="text.secondary">
                  Select a site first
                </Typography>
              )}
            </CardContent>
          </Card>
        </Grid>
      </Grid>

      {/* Verb parameter form */}
      {activeVerb && (
        <Card>
          <CardContent>
            <Typography variant="subtitle2" gutterBottom>
              Parameters
            </Typography>
            <VerbParameterForm
              verb={activeVerb}
              values={fieldValues}
              onChange={handleFieldChange}
            />
          </CardContent>
        </Card>
      )}

      {/* Submit button */}
      <Box>
        <Button
          variant="contained"
          disabled={!selectedSite || !selectedVerb || submitting}
          onClick={handleSubmit}
          startIcon={submitting ? <CircularProgress size={16} color="inherit" /> : undefined}
        >
          {submitting ? 'Submitting...' : 'Submit Workflow'}
        </Button>
      </Box>

      {/* Recent submissions */}
      {recentSubmissions.length > 0 && (
        <Card>
          <CardContent>
            <Typography variant="subtitle2" gutterBottom>
              Recent Submissions
            </Typography>
            <RecentSubmissionsTable submissions={recentSubmissions} />
          </CardContent>
        </Card>
      )}

      {/* Snackbar */}
      <Snackbar
        open={snackbar.open}
        autoHideDuration={5000}
        onClose={handleCloseSnackbar}
        anchorOrigin={{ vertical: 'bottom', horizontal: 'center' }}
      >
        <Alert
          onClose={handleCloseSnackbar}
          severity={snackbar.severity}
          variant="filled"
          sx={{ width: '100%' }}
        >
          {snackbar.message}
        </Alert>
      </Snackbar>
    </Box>
  );
}
