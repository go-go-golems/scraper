import { Component, type ErrorInfo, type ReactNode } from 'react';
import { Box, Button, Card, CardContent, Typography } from '@mui/material';
import ReportProblemIcon from '@mui/icons-material/ReportProblem';

interface AppErrorBoundaryProps {
  children: ReactNode;
}

interface AppErrorBoundaryState {
  hasError: boolean;
  error: Error | null;
}

export class AppErrorBoundary extends Component<
  AppErrorBoundaryProps,
  AppErrorBoundaryState
> {
  constructor(props: AppErrorBoundaryProps) {
    super(props);
    this.state = { hasError: false, error: null };
  }

  static getDerivedStateFromError(error: Error): AppErrorBoundaryState {
    return { hasError: true, error };
  }

  componentDidCatch(error: Error, errorInfo: ErrorInfo): void {
    console.error('[AppErrorBoundary] Unhandled error:', error, errorInfo);
  }

  private handleRetry = () => {
    this.setState({ hasError: false, error: null });
  };

  render() {
    if (this.state.hasError && this.state.error) {
      return (
        <Box
          sx={{
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            minHeight: '50vh',
            p: 3,
          }}
        >
          <Card sx={{ maxWidth: 560, width: '100%' }}>
            <CardContent sx={{ textAlign: 'center', py: 5, px: 4 }}>
              <ReportProblemIcon
                sx={{ fontSize: 56, color: 'error.main', mb: 2 }}
              />
              <Typography variant="h5" gutterBottom>
                Something went wrong
              </Typography>
              <Typography variant="body2" color="text.secondary" sx={{ mb: 1 }}>
                {this.state.error.message}
              </Typography>
              {import.meta.env.DEV && this.state.error.stack && (
                <Box
                  component="pre"
                  sx={{
                    mt: 2,
                    mb: 2,
                    p: 2,
                    bgcolor: 'grey.100',
                    borderRadius: 1,
                    overflow: 'auto',
                    maxHeight: 200,
                    textAlign: 'left',
                    fontSize: '0.75rem',
                    fontFamily: 'monospace',
                    whiteSpace: 'pre-wrap',
                    wordBreak: 'break-word',
                  }}
                >
                  {this.state.error.stack}
                </Box>
              )}
              <Button
                variant="contained"
                onClick={this.handleRetry}
                sx={{ mt: 2 }}
              >
                Try Again
              </Button>
            </CardContent>
          </Card>
        </Box>
      );
    }

    return this.props.children;
  }
}
