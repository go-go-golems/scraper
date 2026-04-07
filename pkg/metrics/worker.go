package metrics

func (r *Registry) SetWorkerUp(workerID string, up bool) {
	if r == nil || workerID == "" {
		return
	}
	if up {
		r.WorkersUp.WithLabelValues(workerID).Set(1)
		return
	}
	r.WorkersUp.WithLabelValues(workerID).Set(0)
}
