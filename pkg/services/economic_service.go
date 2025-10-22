package services

import (
	"bytes"
	"encoding/csv"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// SeriesPoint represents a single daily time-series data point.
type SeriesPoint struct {
	Date  time.Time
	Value float64
}

// EconomicService loads market/economic daily time series from CSV files (pseudo-yfinance).
// It supports flexible headers (Date, Close/Adj Close/Value/Price) and caches parsed series.
type EconomicService struct {
	mu           sync.RWMutex
	baseDir      string
	symbolToFile map[string]string
	cache        map[string][]SeriesPoint // symbol -> sorted daily series (ascending by date)
	dateLayouts  []string                 // accepted date formats
	valueColumns []string                 // ordered preference for value columns
}

// NewEconomicService creates a new EconomicService.
// baseDir is optional. symbolToFile may contain absolute or relative paths. Relative paths will be joined with baseDir.
func NewEconomicService(baseDir string, symbolToFile map[string]string) *EconomicService {
	return &EconomicService{
		baseDir:      baseDir,
		symbolToFile: cloneMap(symbolToFile),
		cache:        make(map[string][]SeriesPoint),
		dateLayouts: []string{
			time.RFC3339,
			"2006-01-02",
			"2006-1-2",
			"2006/01/02",
			"2006/1/2",
			"01/02/2006",
			"20060102",
		},
		valueColumns: []string{"Adj Close", "Adj_Close", "AdjClose", "Close", "Price", "Value", "終値"},
	}
}

// RegisterSymbol registers or overrides a symbol-file mapping at runtime.
func (s *EconomicService) RegisterSymbol(symbol, filePath string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.symbolToFile == nil {
		s.symbolToFile = make(map[string]string)
	}
	s.symbolToFile[strings.ToUpper(symbol)] = filePath
	// invalidate cache for this symbol
	delete(s.cache, strings.ToUpper(symbol))
}

// GetMarketSeries returns a daily series for the symbol in [start, end], resampled to daily with forward fill.
// It lazily loads and caches the CSV for the symbol on first call.
func (s *EconomicService) GetMarketSeries(symbol string, start, end time.Time) ([]SeriesPoint, error) {
	if start.After(end) {
		return nil, fmt.Errorf("start after end: %s > %s", start.Format("2006-01-02"), end.Format("2006-01-02"))
	}

	sym := strings.ToUpper(strings.TrimSpace(symbol))
	series, err := s.getOrLoad(sym)
	if err != nil {
		return nil, err
	}

	// cut to requested range
	cut := cutRange(series, start, end)
	if len(cut) == 0 {
		return nil, fmt.Errorf("no data for %s in range %s..%s", sym, start.Format("2006-01-02"), end.Format("2006-01-02"))
	}

	// resample daily with forward fill to ensure contiguous dates
	daily := resampleDailyFFill(cut, start, end)
	return daily, nil
}

// GetPctChange returns percentage change series (e.g., daily returns) for the symbol in range.
func (s *EconomicService) GetPctChange(symbol string, start, end time.Time) ([]SeriesPoint, error) {
	daily, err := s.GetMarketSeries(symbol, start, end)
	if err != nil {
		return nil, err
	}
	if len(daily) < 2 {
		return []SeriesPoint{}, nil
	}
	out := make([]SeriesPoint, 0, len(daily)-1)
	prev := daily[0].Value
	for i := 1; i < len(daily); i++ {
		cur := daily[i].Value
		var pct float64
		if prev != 0 {
			pct = (cur - prev) / prev
		} else {
			pct = 0
		}
		out = append(out, SeriesPoint{Date: daily[i].Date, Value: pct})
		prev = cur
	}
	return out, nil
}

// getOrLoad returns cached series or loads from CSV.
func (s *EconomicService) getOrLoad(symbol string) ([]SeriesPoint, error) {
	s.mu.RLock()
	if series, ok := s.cache[symbol]; ok {
		s.mu.RUnlock()
		return series, nil
	}
	s.mu.RUnlock()

	s.mu.Lock()
	defer s.mu.Unlock()
	if series, ok := s.cache[symbol]; ok { // double-check
		return series, nil
	}

	path, ok := s.symbolToFile[symbol]
	if !ok || path == "" {
		return nil, fmt.Errorf("no CSV mapping for symbol: %s", symbol)
	}
	if !filepath.IsAbs(path) && s.baseDir != "" {
		path = filepath.Join(s.baseDir, path)
	}

	series, err := s.loadCSV(path)
	if err != nil {
		return nil, fmt.Errorf("failed to load CSV for %s: %w", symbol, err)
	}
	if len(series) == 0 {
		return nil, fmt.Errorf("empty series for %s from file %s", symbol, path)
	}

	s.cache[symbol] = series
	return series, nil
}

// loadCSV parses a CSV file and returns a sorted daily series.
// Supported headers: Date, Close, Adj Close, Adj_Close, AdjClose, Price, Value (case-insensitive)
func (s *EconomicService) loadCSV(filePath string) ([]SeriesPoint, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	// read using common parser
	return s.parseCSVReader(csv.NewReader(f))
}

// ParseCSVBytes parses CSV content from bytes and returns a sorted, deduplicated daily series.
func (s *EconomicService) ParseCSVBytes(data []byte) ([]SeriesPoint, error) {
	r := csv.NewReader(bytes.NewReader(data))
	r.FieldsPerRecord = -1
	return s.parseCSVReader(r)
}

// parseCSVReader is a shared implementation for parsing CSV rows into a series.
func (s *EconomicService) parseCSVReader(r *csv.Reader) ([]SeriesPoint, error) {
	rows, err := r.ReadAll()
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, errors.New("csv: no data")
	}

	header := normalizeHeader(rows[0])
	// Accept both English and Japanese date headers
	dateIdx := findIndex(header, []string{"date", "年月日", "日付", "データ日付"})
	if dateIdx == -1 {
		return nil, errors.New("csv: date column not found")
	}
	valIdx := -1
	for _, name := range s.valueColumns {
		idx := findIndex(header, []string{strings.ToLower(name)})
		if idx != -1 {
			valIdx = idx
			break
		}
	}
	// Fallback: pick first header that includes 'close'/'price' or Japanese '終値'
	if valIdx == -1 {
		for i, h := range header {
			if strings.Contains(h, "close") || strings.Contains(h, "price") || strings.Contains(h, "終値") {
				valIdx = i
				break
			}
		}
	}
	if valIdx == -1 {
		return nil, errors.New("csv: value column (Close/Adj Close/Price/Value/終値) not found")
	}

	var series []SeriesPoint
	for i := 1; i < len(rows); i++ {
		row := rows[i]
		if len(row) <= dateIdx || len(row) <= valIdx {
			continue
		}
		dateStr := strings.TrimSpace(row[dateIdx])
		if dateStr == "" {
			continue
		}
		dt, ok := parseAnyDate(dateStr, s.dateLayouts)
		if !ok {
			continue
		}
		valStr := strings.ReplaceAll(strings.TrimSpace(row[valIdx]), ",", "")
		// Remove units or extra chars (e.g., 円, spaces, etc.) by keeping only digits, dot, minus
		valStr = filterNumeric(valStr)
		if valStr == "" {
			continue
		}
		v, err := strconv.ParseFloat(valStr, 64)
		if err != nil {
			continue
		}
		series = append(series, SeriesPoint{Date: dt, Value: v})
	}

	if len(series) == 0 {
		return nil, errors.New("csv: no valid rows")
	}

	// sort and deduplicate by date (last wins)
	sort.Slice(series, func(i, j int) bool { return series[i].Date.Before(series[j].Date) })
	dedup := make([]SeriesPoint, 0, len(series))
	var lastDate time.Time
	for _, p := range series {
		if len(dedup) == 0 || !sameDay(p.Date, lastDate) {
			dedup = append(dedup, SeriesPoint{Date: day(p.Date), Value: p.Value})
			lastDate = p.Date
		} else {
			// overwrite last if duplicate date encountered
			dedup[len(dedup)-1] = SeriesPoint{Date: day(p.Date), Value: p.Value}
		}
	}
	return dedup, nil
}

// Helper: cut series to [start, end]
func cutRange(series []SeriesPoint, start, end time.Time) []SeriesPoint {
	s := day(start)
	e := day(end)
	out := make([]SeriesPoint, 0, len(series))
	for _, p := range series {
		d := day(p.Date)
		if (d.Equal(s) || d.After(s)) && (d.Equal(e) || d.Before(e)) {
			out = append(out, SeriesPoint{Date: d, Value: p.Value})
		}
	}
	return out
}

// resampleDailyFFill ensures a contiguous daily series using forward-fill.
func resampleDailyFFill(series []SeriesPoint, start, end time.Time) []SeriesPoint {
	if len(series) == 0 {
		return []SeriesPoint{}
	}
	m := make(map[time.Time]float64, len(series))
	for _, p := range series {
		m[day(p.Date)] = p.Value
	}
	cur := day(start)
	e := day(end)
	out := make([]SeriesPoint, 0, int(e.Sub(cur).Hours()/24)+1)
	var last float64
	var hasLast bool
	for !cur.After(e) {
		if v, ok := m[cur]; ok {
			last = v
			hasLast = true
			out = append(out, SeriesPoint{Date: cur, Value: v})
		} else {
			if hasLast {
				out = append(out, SeriesPoint{Date: cur, Value: last})
			} else {
				// if no previous value yet, set 0 (or NaN). We choose 0 to avoid NaN propagation.
				out = append(out, SeriesPoint{Date: cur, Value: 0})
			}
		}
		cur = cur.AddDate(0, 0, 1)
	}
	return out
}

// Utility helpers
func parseAnyDate(s string, layouts []string) (time.Time, bool) {
	// common yfinance style may include time; trim timezone if present
	s = strings.TrimSpace(s)
	for _, layout := range layouts {
		if t, err := time.Parse(layout, s); err == nil {
			return day(t), true
		}
	}
	// try to split date part if time included
	if i := strings.IndexAny(s, " T"); i > 0 {
		part := s[:i]
		for _, layout := range layouts {
			if t, err := time.Parse(layout, part); err == nil {
				return day(t), true
			}
		}
	}
	return time.Time{}, false
}

func normalizeHeader(hdr []string) []string {
	out := make([]string, len(hdr))
	for i, v := range hdr {
		// Remove UTF-8 BOM if present, then trim and lowercase
		v = strings.TrimPrefix(v, "\ufeff")
		out[i] = strings.ToLower(strings.TrimSpace(v))
	}
	return out
}

func findIndex(hdr []string, candidates []string) int {
	for i, v := range hdr {
		for _, c := range candidates {
			if v == c {
				return i
			}
		}
	}
	return -1
}

func sameDay(a, b time.Time) bool { return day(a).Equal(day(b)) }
func day(t time.Time) time.Time   { return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC) }

func cloneMap(m map[string]string) map[string]string {
	if m == nil {
		return nil
	}
	out := make(map[string]string, len(m))
	for k, v := range m {
		out[strings.ToUpper(k)] = v
	}
	return out
}

// filterNumeric keeps digits, dot, and minus to parse numbers like "35,000円" -> "35000".
func filterNumeric(s string) string {
	b := make([]rune, 0, len(s))
	for _, r := range s {
		if (r >= '0' && r <= '9') || r == '.' || r == '-' {
			b = append(b, r)
		}
	}
	return string(b)
}
