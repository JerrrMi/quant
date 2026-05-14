package backtest

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/JerrrMi/quant/internal/domain"
	"github.com/JerrrMi/quant/internal/domain/strategy"
	"github.com/JerrrMi/quant/internal/infra/lppl"
	"github.com/JerrrMi/quant/internal/infra/marketdata"
)

// HistoricalLoader loads sorted closed bars for a symbol (oldest first).
type HistoricalLoader interface {
	LoadRange(ctx context.Context, symbol string) ([]domain.Bar, error)
}

// FileBarLoader reads OHLCV from CSV (no header required if numeric-only; standard header supported).
// Columns: timestamp_ms,open,high,low,close,volume OR unix_ms,o,h,l,c,v
type FileBarLoader struct {
	RootPath string
}

// LoadRange resolves RootPath to a file: if directory, uses {symbol}.csv then bars.csv.
func (l *FileBarLoader) LoadRange(_ context.Context, symbol string) ([]domain.Bar, error) {
	if l == nil || strings.TrimSpace(l.RootPath) == "" {
		return nil, fmt.Errorf("backtest: empty loader path")
	}
	path := l.RootPath
	fi, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if fi.IsDir() {
		cand := filepath.Join(path, symbol+".csv")
		if _, err := os.Stat(cand); err == nil {
			path = cand
		} else {
			alt := filepath.Join(path, "bars.csv")
			if _, err2 := os.Stat(alt); err2 == nil {
				path = alt
			} else {
				return nil, fmt.Errorf("backtest: no csv in dir %q for symbol %q", l.RootPath, symbol)
			}
		}
	}
	return readBarsCSV(path)
}

func readBarsCSV(path string) ([]domain.Bar, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	r := csv.NewReader(f)
	r.TrimLeadingSpace = true

	first, err := r.Read()
	if err != nil {
		return nil, err
	}

	var rows [][]string
	if looksLikeHeader(first) {
		for {
			rec, er := r.Read()
			if errors.Is(er, io.EOF) {
				break
			}
			if er != nil {
				return nil, er
			}
			rows = append(rows, rec)
		}
	} else {
		rows = append(rows, first)
		for {
			rec, er := r.Read()
			if errors.Is(er, io.EOF) {
				break
			}
			if er != nil {
				return nil, er
			}
			rows = append(rows, rec)
		}
	}

	out := make([]domain.Bar, 0, len(rows))
	for i, rec := range rows {
		if len(rec) < 6 {
			return nil, fmt.Errorf("backtest csv %q line %d: need 6 columns", path, i+1)
		}
		ts, err := parseInt64(strings.TrimSpace(rec[0]))
		if err != nil {
			return nil, fmt.Errorf("backtest csv %q line %d ts: %w", path, i+1, err)
		}
		o, err := strconv.ParseFloat(strings.TrimSpace(rec[1]), 64)
		if err != nil {
			return nil, fmt.Errorf("backtest csv %q line %d open: %w", path, i+1, err)
		}
		h, err := strconv.ParseFloat(strings.TrimSpace(rec[2]), 64)
		if err != nil {
			return nil, fmt.Errorf("backtest csv %q line %d high: %w", path, i+1, err)
		}
		lo, err := strconv.ParseFloat(strings.TrimSpace(rec[3]), 64)
		if err != nil {
			return nil, fmt.Errorf("backtest csv %q line %d low: %w", path, i+1, err)
		}
		c, err := strconv.ParseFloat(strings.TrimSpace(rec[4]), 64)
		if err != nil {
			return nil, fmt.Errorf("backtest csv %q line %d close: %w", path, i+1, err)
		}
		v, err := strconv.ParseFloat(strings.TrimSpace(rec[5]), 64)
		if err != nil {
			return nil, fmt.Errorf("backtest csv %q line %d volume: %w", path, i+1, err)
		}
		out = append(out, domain.Bar{
			Open: o, High: h, Low: lo, Close: c, Volume: v,
			TimestampUnixMs: ts,
		})
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("backtest: no bars in %q", path)
	}
	if err := marketdata.ValidateMonotonicTimestamps(out); err != nil {
		return nil, err
	}
	return out, nil
}

func looksLikeHeader(row []string) bool {
	if len(row) == 0 {
		return false
	}
	s := strings.ToLower(strings.TrimSpace(row[0]))
	return strings.Contains(s, "time") || strings.Contains(s, "ts") || strings.Contains(s, "date")
}

func parseInt64(s string) (int64, error) {
	return strconv.ParseInt(s, 10, 64)
}

// FilterBarsTimeWindow keeps bars with TimestampUnixMs in [start,end) when bounds are set.
func FilterBarsTimeWindow(bars []domain.Bar, start, end *time.Time) []domain.Bar {
	if start == nil && end == nil {
		out := make([]domain.Bar, len(bars))
		copy(out, bars)
		return out
	}
	out := make([]domain.Bar, 0, len(bars))
	for _, b := range bars {
		t := time.UnixMilli(b.TimestampUnixMs).UTC()
		if start != nil && t.Before(*start) {
			continue
		}
		if end != nil && !t.Before(*end) {
			continue
		}
		out = append(out, b)
	}
	return out
}

// ParseReplayWindow returns optional UTC bounds from config strings (RFC3339).
func ParseReplayWindow(start, end string) (s, e *time.Time, err error) {
	parse := func(raw string) (*time.Time, error) {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			return nil, nil
		}
		t, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			return nil, err
		}
		utc := t.UTC()
		return &utc, nil
	}
	s, err = parse(start)
	if err != nil {
		return nil, nil, fmt.Errorf("replay.window.start: %w", err)
	}
	e, err = parse(end)
	if err != nil {
		return nil, nil, fmt.Errorf("replay.window.end: %w", err)
	}
	return s, e, nil
}

// DataFeed builds AltShortStrategyInput aligned with scheduler.StepOrchestrator.tickInstance
// (recent window, BuildFeatureSnapshot, optional LPPL ApplyLatest).
type DataFeed struct {
	Symbol   string
	FeatSpec marketdata.FeatureWindowSpec
	LPPL     *lppl.InputAugmentor
}

// BuildInput constructs one step input; idx is the index of the current closed bar in bars.
func (d *DataFeed) BuildInput(ctx context.Context, bars []domain.Bar, idx int, netQty float64, shortOpenedAt int64, stepSeq int64, risk strategy.RiskSnapshot) (strategy.AltShortStrategyInput, error) {
	if d == nil || strings.TrimSpace(d.Symbol) == "" {
		return strategy.AltShortStrategyInput{}, fmt.Errorf("backtest: datafeed not configured")
	}
	if idx < 0 || idx >= len(bars) {
		return strategy.AltShortStrategyInput{}, fmt.Errorf("backtest: bar idx out of range")
	}
	need := d.FeatSpec.WindowBars + 1
	if idx+1 < need {
		return strategy.AltShortStrategyInput{}, fmt.Errorf("backtest: need at least %d bars up to idx", need)
	}
	start := idx + 1 - need
	window := bars[start : idx+1]
	barCurrent := window[len(window)-1]
	priorClose := 0.0
	if pc, ok := marketdata.PriorClose(window); ok {
		priorClose = pc
	}
	features, err := marketdata.BuildFeatureSnapshot(window, d.FeatSpec)
	if err != nil {
		return strategy.AltShortStrategyInput{}, err
	}
	in := strategy.AltShortStrategyInput{
		Symbol:              d.Symbol,
		NetPositionQty:      netQty,
		ShortOpenedAtUnixMs: shortOpenedAt,
		PriorBarClose:       priorClose,
		BarCurrent:          barCurrent,
		Features:            features,
		Risk:                risk,
		NowUnixMs:           barCurrent.TimestampUnixMs,
		StepSequence:        stepSeq,
	}
	if d.LPPL != nil {
		_ = d.LPPL.ApplyLatest(ctx, &in, d.Symbol)
	}
	return in, nil
}

// NewLPPLAugmentorFromConfig returns InputAugmentor when enabled; same code path as SaaS (ApplyLatest).
func NewLPPLAugmentorFromConfig(enabled bool, bubble float64, jobID string) *lppl.InputAugmentor {
	if !enabled {
		return nil
	}
	resJSON, _ := json.Marshal(lppl.ResultScalars{BubbleMetric01: bubble})
	view := &lppl.ResultView{
		Symbol:     "",
		ComputedAt: time.Unix(1, 0).UTC(),
		JobID:      jobID,
		ResultJSON: string(resJSON),
	}
	return &lppl.InputAugmentor{Store: lppl.FixedLatestStore{View: view}}
}
