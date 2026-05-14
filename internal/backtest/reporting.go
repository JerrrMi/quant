package backtest

import (
	"math"
	"sort"

	"github.com/JerrrMi/quant/internal/domain/command"
)

// EquityPoint is one mark-to-market sample after a step completes.
type EquityPoint struct {
	UnixMs        int64   `json:"unix_ms"`
	Equity        float64 `json:"equity"`
	Balance       float64 `json:"balance"`
	NetPosition   float64 `json:"net_position"`
	Mark          float64 `json:"mark"`
	StepSequence  int64   `json:"step_sequence"`
	TradedNotional float64 `json:"traded_notional_step,omitempty"`
}

// CommandStat summarizes synthetic execution diagnostics per command.
type CommandStat struct {
	CommandID string `json:"command_id"`
	IntentID  string `json:"intent_id"`
	Status    string `json:"status"`
	Partial   bool   `json:"partial"`
	Message   string `json:"message,omitempty"`
}

// PerformanceMetrics are backtest-level KPIs (JSON-friendly for export).
type PerformanceMetrics struct {
	InitialEquity    float64 `json:"initial_equity"`
	FinalEquity      float64 `json:"final_equity"`
	TotalReturn      float64 `json:"total_return"`
	MaxDrawdown01    float64 `json:"max_drawdown_01"`
	WinRate          float64 `json:"win_rate"`
	NumRoundTrips    int     `json:"num_round_trips"`
	TurnoverRatio    float64 `json:"turnover_ratio"`
	CommandHitRate   float64 `json:"command_hit_rate"`
	CommandFailRate  float64 `json:"command_fail_rate"`
	PartialFillRate  float64 `json:"partial_fill_rate"`
	AvgEquity        float64 `json:"avg_equity"`
	CumulativeNetFees float64 `json:"cumulative_net_fees,omitempty"`
}

// BacktestReport aggregates series and aggregates for visualization/export.
type BacktestReport struct {
	Metrics      PerformanceMetrics `json:"metrics"`
	EquityCurve  []EquityPoint      `json:"equity_curve"`
	CommandStats []CommandStat      `json:"command_stats"`
}

// TradeRing holds short round-trips for win-rate (entry from flat→short, exit short→flat).
type tradeRecorder struct {
	inTrade     bool
	entryEquity float64
	wins        int
	count       int
}

func (t *tradeRecorder) onEquity(e float64, wasShort, isShort bool) {
	if !t.inTrade && !wasShort && isShort {
		t.inTrade = true
		t.entryEquity = e
		return
	}
	if t.inTrade && wasShort && !isShort {
		t.inTrade = false
		t.count++
		if e > t.entryEquity {
			t.wins++
		}
	}
}

// BuildReport computes metrics from equity path and command outcomes.
func BuildReport(initial float64, curve []EquityPoint, outcomes []SimOutcome, cumulativeFees float64) BacktestReport {
	rep := BacktestReport{
		EquityCurve: append([]EquityPoint(nil), curve...),
		CommandStats: nil,
	}
	tr := tradeRecorder{}
	eps := 1e-8

	peak := initial
	maxDD := 0.0
	var sumE float64
	sumTurnoverNum := 0.0
	prevWasShort := false

	for i, pt := range curve {
		sumE += pt.Equity
		sumTurnoverNum += math.Abs(pt.TradedNotional)
		if pt.Equity > peak {
			peak = pt.Equity
		}
		if peak > eps {
			dd := (peak - pt.Equity) / peak
			if dd > maxDD {
				maxDD = dd
			}
		}
		isShort := pt.NetPosition < -eps
		tr.onEquity(pt.Equity, prevWasShort, isShort)
		prevWasShort = isShort
		_ = i
	}

	nCmd := len(outcomes)
	var nHit, nFail, nPart float64
	for _, o := range outcomes {
		switch o.Status {
		case command.CommandStatusRejected, command.CommandStatusExpired:
			nFail++
		case command.CommandStatusWorking, command.CommandStatusCompleted:
			if o.IsPartial {
				nPart++
			}
			nHit++
		default:
			nHit++
		}
		rep.CommandStats = append(rep.CommandStats, CommandStat{
			CommandID: o.CommandID,
			IntentID:  o.IntentID,
			Status:    string(o.Status),
			Partial:   o.IsPartial,
			Message:   o.Message,
		})
	}
	sort.Slice(rep.CommandStats, func(i, j int) bool {
		return rep.CommandStats[i].IntentID < rep.CommandStats[j].IntentID
	})

	final := initial
	if len(curve) > 0 {
		final = curve[len(curve)-1].Equity
	}
	avgE := 0.0
	if len(curve) > 0 {
		avgE = sumE / float64(len(curve))
	}
	tvr := 0.0
	if avgE > eps {
		tvr = sumTurnoverNum / avgE
	}

	winRate := 0.0
	if tr.count > 0 {
		winRate = float64(tr.wins) / float64(tr.count)
	}

	hitRate := 0.0
	failRate := 0.0
	partRate := 0.0
	if nCmd > 0 {
		hitRate = nHit / float64(nCmd)
		failRate = nFail / float64(nCmd)
		partRate = nPart / float64(nCmd)
	}

	totRet := 0.0
	if initial > eps {
		totRet = (final - initial) / initial
	}

	rep.Metrics = PerformanceMetrics{
		InitialEquity:      initial,
		FinalEquity:        final,
		TotalReturn:        totRet,
		MaxDrawdown01:      maxDD,
		WinRate:            winRate,
		NumRoundTrips:      tr.count,
		TurnoverRatio:      tvr,
		CommandHitRate:    hitRate,
		CommandFailRate:   failRate,
		PartialFillRate: partRate,
		AvgEquity:          avgE,
		CumulativeNetFees:  cumulativeFees,
	}
	return rep
}
