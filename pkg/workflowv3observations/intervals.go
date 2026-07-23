package workflowv3observations

import (
	"sort"
	"time"
)

type interval struct {
	start time.Time
	end   time.Time
}

func intersectIntervals(input []interval, boundary interval) []interval {
	ret := make([]interval, 0, len(input))
	for _, current := range input {
		start, end := current.start, current.end
		if start.Before(boundary.start) {
			start = boundary.start
		}
		if end.After(boundary.end) {
			end = boundary.end
		}
		if end.Before(start) {
			continue
		}
		ret = append(ret, interval{start: start, end: end})
	}
	return ret
}

func intervalMicros(intervals []interval) (int64, int64, int) {
	var sum, union int64
	var peak int
	valid := make([]interval, 0, len(intervals))
	for _, current := range intervals {
		if current.start.IsZero() || current.end.Before(current.start) {
			continue
		}
		sum += current.end.Sub(current.start).Microseconds()
		valid = append(valid, current)
	}
	if len(valid) == 0 {
		return sum, 0, 0
	}
	sort.Slice(valid, func(i, j int) bool {
		if valid[i].start.Equal(valid[j].start) {
			return valid[i].end.Before(valid[j].end)
		}
		return valid[i].start.Before(valid[j].start)
	})
	start, end := valid[0].start, valid[0].end
	for _, current := range valid[1:] {
		if !current.start.After(end) {
			if current.end.After(end) {
				end = current.end
			}
			continue
		}
		union += end.Sub(start).Microseconds()
		start, end = current.start, current.end
	}
	union += end.Sub(start).Microseconds()

	type endpoint struct {
		at    time.Time
		delta int
	}
	points := make([]endpoint, 0, len(valid)*2)
	for _, current := range valid {
		if current.start.Equal(current.end) {
			continue
		}
		points = append(points, endpoint{at: current.start, delta: 1}, endpoint{at: current.end, delta: -1})
	}
	sort.Slice(points, func(i, j int) bool {
		if points[i].at.Equal(points[j].at) {
			return points[i].delta < points[j].delta // half-open: ends before starts
		}
		return points[i].at.Before(points[j].at)
	})
	active := 0
	for _, point := range points {
		active += point.delta
		if active > peak {
			peak = active
		}
	}
	return sum, union, peak
}
