package services

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// AggregatedPoint represents a weekly/monthly aggregated value over a period.
type AggregatedPoint struct {
	Period    string  // e.g., 2024-W12 or 2024-03
	StartDate string  // YYYY-MM-DD (inclusive)
	EndDate   string  // YYYY-MM-DD (inclusive)
	Value     float64 // aggregated value
}

// AggregateSeriesDailyTo aggregates economic daily series to the specified granularity.
// daily: []struct{Date string; Value float64}; granularity: "weekly"|"monthly"; method: "sum"|"mean"|"last"
func AggregateSeriesDailyTo(daily []struct {
	Date  string
	Value float64
}, granularity, method string) []AggregatedPoint {
	if granularity == "daily" {
		// Not applicable; return empty
		return nil
	}
	groups := make(map[string][]struct {
		t time.Time
		v float64
	})
	meta := make(map[string]struct{ start, end time.Time })
	order := make([]string, 0)
	for _, d := range daily {
		if d.Date == "" {
			continue
		}
		t, err := time.Parse("2006-01-02", d.Date)
		if err != nil {
			continue
		}
		var key string
		var start, end time.Time
		switch granularity {
		case "weekly":
			// ISO week; use Monday-Sunday
			// find Monday of this week
			weekday := int(t.Weekday())
			if weekday == 0 {
				weekday = 7
			}
			start = t.AddDate(0, 0, -(weekday - 1))
			end = start.AddDate(0, 0, 6)
			y, w := start.ISOWeek()
			key = fmt.Sprintf("%04d-W%02d", y, w)
		case "monthly":
			start = time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
			end = start.AddDate(0, 1, -1)
			key = start.Format("2006-01")
		default:
			continue
		}
		if _, ok := groups[key]; !ok {
			order = append(order, key)
		}
		groups[key] = append(groups[key], struct {
			t time.Time
			v float64
		}{t: t, v: d.Value})
		if m, ok := meta[key]; ok {
			if d, _ := time.Parse("2006-01-02", m.start.Format("2006-01-02")); t.Before(d) {
				m.start = t
			}
			if t.After(m.end) {
				m.end = t
			}
			meta[key] = m
		} else {
			meta[key] = struct{ start, end time.Time }{start: start, end: end}
		}
	}
	// stable order by key
	sort.Strings(order)
	out := make([]AggregatedPoint, 0, len(order))
	for _, key := range order {
		values := groups[key]
		if len(values) == 0 {
			continue
		}
		var val float64
		switch strings.ToLower(method) {
		case "mean", "avg":
			for _, x := range values {
				val += x.v
			}
			val /= float64(len(values))
		case "last":
			// pick last by time
			sort.Slice(values, func(i, j int) bool { return values[i].t.Before(values[j].t) })
			val = values[len(values)-1].v
		default: // sum
			for _, x := range values {
				val += x.v
			}
		}
		m := meta[key]
		out = append(out, AggregatedPoint{Period: key, StartDate: m.start.Format("2006-01-02"), EndDate: m.end.Format("2006-01-02"), Value: val})
	}
	return out
}

// AggregateSalesDailyTo aggregates sales daily series to weekly/monthly using method.
func AggregateSalesDailyTo(daily []struct {
	Date  string
	Sales float64
}, granularity, method string) []AggregatedPoint {
	econLike := make([]struct {
		Date  string
		Value float64
	}, 0, len(daily))
	for _, d := range daily {
		econLike = append(econLike, struct {
			Date  string
			Value float64
		}{Date: d.Date, Value: d.Sales})
	}
	return AggregateSeriesDailyTo(econLike, granularity, method)
}
