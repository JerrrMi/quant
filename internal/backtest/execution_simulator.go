package backtest

import (
	"fmt"
	"math"
	"math/rand"

	"github.com/JerrrMi/quant/internal/domain"
	"github.com/JerrrMi/quant/internal/domain/command"
)

// SimParams configures synthetic execution (approximate).
type SimParams struct {
	MakerBps         float64
	TakerBps         float64
	SlippageBps      float64
	UseTaker         bool
	FailureRate      float64
	PartialProb      float64
	PartialMinFrac   float64
	PartialMaxFrac   float64
	LotStep          float64
	FundingBpsPerDay float64
	RNG              *rand.Rand
}

// DefaultSimParams fills zero fields with safe defaults.
func DefaultSimParams() SimParams {
	return SimParams{
		PartialMinFrac: 0.5,
		PartialMaxFrac: 1,
		LotStep:        0.001,
	}
}

// Simulator applies slippage, fees, funding hooks, and stochastic outcomes.
type Simulator struct {
	p SimParams
}

// NewSimulator builds a simulator; merges defaults for partial fill bounds and lot step.
func NewSimulator(p SimParams) *Simulator {
	d := DefaultSimParams()
	if p.PartialMinFrac == 0 && p.PartialMaxFrac == 0 {
		p.PartialMinFrac = d.PartialMinFrac
		p.PartialMaxFrac = d.PartialMaxFrac
	}
	if p.LotStep <= 0 {
		p.LotStep = d.LotStep
	}
	if p.RNG == nil {
		src := rand.NewSource(1)
		p.RNG = rand.New(src)
	}
	if p.PartialMaxFrac < p.PartialMinFrac {
		p.PartialMaxFrac = p.PartialMinFrac
	}
	return &Simulator{p: p}
}

// SimOutcome records one synthetic execution result.
type SimOutcome struct {
	CommandID   string
	IntentID    string
	StepSeq     int64
	FillQtyBase float64
	FillPrice   float64
	FeePaid     float64
	Status      command.CommandStatus
	Message     string
	IsPartial   bool
}

// Run executes cmd against reference mark (typically bar close) and current net base position.
func (s *Simulator) Run(cmd command.TradeCommand, mark float64, curNet float64) (SimOutcome, float64, error) {
	if s == nil {
		return SimOutcome{}, curNet, fmt.Errorf("backtest: nil simulator")
	}
	if mark <= 0 || math.IsNaN(mark) || math.IsInf(mark, 0) {
		return SimOutcome{}, curNet, fmt.Errorf("backtest: invalid mark")
	}
	rng := s.p.RNG
	if rng == nil {
		rng = rand.New(rand.NewSource(1))
	}

	out := SimOutcome{
		CommandID: cmd.CommandID,
		IntentID:  cmd.Intent.IntentID,
		StepSeq:   0,
		Status:    command.CommandStatusCompleted,
	}

	if s.p.FailureRate > 0 && rng.Float64() < s.p.FailureRate {
		out.Status = command.CommandStatusRejected
		out.Message = "simulated reject"
		return out, curNet, nil
	}

	delta, err := intendedBaseDelta(cmd, curNet, mark)
	if err != nil {
		out.Status = command.CommandStatusRejected
		out.Message = err.Error()
		return out, curNet, nil
	}
	if math.Abs(delta) < 1e-18 {
		out.Status = command.CommandStatusCompleted
		out.Message = "no-op delta"
		return out, curNet, nil
	}

	qty := math.Abs(delta)
	qty = roundDownToLot(qty, s.p.LotStep)
	if qty <= 0 {
		out.Status = command.CommandStatusRejected
		out.Message = "quantity after lot rounding is zero"
		return out, curNet, nil
	}

	frac := 1.0
	if s.p.PartialProb > 0 && rng.Float64() < s.p.PartialProb {
		frac = s.p.PartialMinFrac + rng.Float64()*(s.p.PartialMaxFrac-s.p.PartialMinFrac)
		if frac < 1e-6 {
			frac = 1e-6
		}
		out.IsPartial = true
		out.Status = command.CommandStatusWorking
	}
	qty *= frac
	qty = roundDownToLot(qty, s.p.LotStep)
	if qty <= 0 {
		out.Status = command.CommandStatusRejected
		out.Message = "partial rounded to zero"
		return out, curNet, nil
	}

	// Reconstruct signed delta for side.
	signed := qty
	if delta < 0 {
		signed = -qty
	}

	slip := s.p.SlippageBps / 10000
	var fillPx float64
	switch {
	case signed < 0: // sell
		fillPx = mark * (1 - slip)
	case signed > 0: // buy
		fillPx = mark * (1 + slip)
	default:
		fillPx = mark
	}
	if !finiteFloat(fillPx) || fillPx <= 0 {
		out.Status = command.CommandStatusRejected
		out.Message = "invalid fill price"
		return out, curNet, nil
	}

	bps := s.p.MakerBps
	if s.p.UseTaker {
		bps = s.p.TakerBps
	}
	notional := math.Abs(signed) * fillPx
	fee := notional * bps / 10000

	out.FillQtyBase = signed
	out.FillPrice = fillPx
	out.FeePaid = fee

	newNet := curNet + signed
	if out.IsPartial {
		out.Status = command.CommandStatusWorking
	} else {
		out.Status = command.CommandStatusCompleted
	}
	return out, newNet, nil
}

// ApplyFunding adjusts balance for synthetic funding on abs(position)*mark over barDurMs milliseconds.
func ApplyFunding(balance float64, posBase float64, mark float64, barDurMs int64, bpsPerDay float64) float64 {
	if bpsPerDay <= 0 || mark <= 0 {
		return balance
	}
	dayFrac := float64(barDurMs) / (86400_000.0)
	charge := math.Abs(posBase) * mark * (bpsPerDay / 10000) * dayFrac
	return balance - charge
}

func intendedBaseDelta(cmd command.TradeCommand, curNet, mark float64) (float64, error) {
	if cmd.TargetPosition != nil {
		return *cmd.TargetPosition - curNet, nil
	}
	if cmd.Intent.TargetPositionQty != nil {
		return *cmd.Intent.TargetPositionQty - curNet, nil
	}

	var not float64
	if cmd.TargetNotional != nil {
		not = math.Abs(*cmd.TargetNotional)
	} else if cmd.Intent.TargetNotionalUSDT != nil {
		not = math.Abs(*cmd.Intent.TargetNotionalUSDT)
	} else {
		return 0, fmt.Errorf("need target position or notional")
	}
	if not <= 0 {
		return 0, nil
	}
	q := not / mark
	switch cmd.Side {
	case domain.SideSell:
		return -q, nil
	case domain.SideBuy:
		return q, nil
	default:
		return 0, fmt.Errorf("unknown side %q", cmd.Side)
	}
}

func finiteFloat(x float64) bool {
	return !math.IsNaN(x) && !math.IsInf(x, 0)
}

func roundDownToLot(q, step float64) float64 {
	if step <= 0 {
		return q
	}
	return math.Floor(q/step) * step
}
