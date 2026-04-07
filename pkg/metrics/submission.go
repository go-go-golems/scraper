package metrics

func (r *Registry) ObserveSubmissionAccepted(site string, verb string) {
	if r == nil {
		return
	}
	r.WorkflowsSubmittedTotal.WithLabelValues(site, verb).Inc()
}

func (r *Registry) ObserveSubmissionFailure(site string, verb string, errorCode string) {
	if r == nil {
		return
	}
	if errorCode == "" {
		errorCode = "unknown"
	}
	r.SubmissionFailuresTotal.WithLabelValues(site, verb, errorCode).Inc()
}
