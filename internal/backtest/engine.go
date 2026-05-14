package backtest

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/JerrrMi/quant/internal/config"
	"github.com/JerrrMi/quant/internal/domain"
	"github.com/JerrrMi/quant/internal/domain/command"
	"github.com/JerrrMi/quant/internal/domain/strategy"
	"github.com/JerrrMi/quant/internal/infra/marketdata"
	"github.com/google/uuid"
)

type portfolio struct {
	balance     float64
	netQty      float64
	avgEntryPx  float64
	shortOpenMS int64
}

func (p *portfolio) unrealized(mark float64) float64 {
	if p.netQty >= -1e-12 {
		return 0
	}
	q := math.Abs(p.netQty)
	return (p.avgEntryPx - mark) * q
}

func (p *portfolio) equity(mark float64) float64 {
	return p.balance + p.unrealized(mark)
}

func (p *portfolio) applySellShort(q, px, fee float64) {
	if q <= 0 {
		return
	}
	if p.netQty >= -1e-12 {
		p.netQty = -q
		p.avgEntryPx = px
	} else {
		oldAbs := math.Abs(p.netQty)
		newAbs := oldAbs + q
		p.avgEntryPx = (oldAbs*p.avgEntryPx + q*px) / newAbs
		p.netQty -= q
	}
	p.balance -= fee
}

func (p *portfolio) applyBuyCover(q, px, fee float64) {
	if q <= 0 || p.netQty >= -1e-12 {
		return
	}
	oldAbs := math.Abs(p.netQty)
	closeQ := math.Min(q, oldAbs)
	realized := (p.avgEntryPx - px) * closeQ
	p.balance += realized - fee
	p.netQty += closeQ
	if math.Abs(p.netQty) < 1e-12 {
		p.netQty = 0
		p.avgEntryPx = 0
	}
}

func (p *portfolio) applySignedFill(signedQty, px, fee float64) {
	if signedQty < 0 {
		p.applySellShort(-signedQty, px, fee)
		return
	}
	if signedQty > 0 {
		p.applyBuyCover(signedQty, px, fee)
	}
}

func (p *portfolio) syncShortOpened(barMS int64, prevNet float64) {
	eps := 1e-9
	shortNow := p.netQty < -eps
	shortPrev := prevNet < -eps
	switch {
	case !shortPrev && shortNow:
		p.shortOpenMS = barMS
	case shortPrev && !shortNow:
		p.shortOpenMS = 0
	}
}

type queuedCmd struct {
	dueIdx int
	cmd    command.TradeCommand
}

// Engine drives replay: shared strategy.Step, TradeCommand-shaped simulation, portfolio MTM.
type Engine struct {
	cfg    config.BacktestConfig
	feed   *DataFeed
	sim    *Simulator
	log    *slog.Logger
	delay  int
	deadMS int64
	// Progress optional: completed replay steps vs total planned steps (after warmup).
	onProgress func(done int, total int)
}

// NewEngine builds an engine; feed and sim must be non-nil for Run.
func NewEngine(cfg config.BacktestConfig, feed *DataFeed, sim *Simulator, log *slog.Logger) *Engine {
	if log == nil {
		log = slog.Default()
	}
	deadMS := cfg.Simulation.CommandDeadlineMs
	if deadMS <= 0 {
		deadMS = 120_000
	}
	return &Engine{cfg: cfg, feed: feed, sim: sim, log: log, delay: cfg.Simulation.DelayBars, deadMS: deadMS}
}

// WithProgress attaches a hook invoked during Run (best-effort; throttling由调用方处理).
func (e *Engine) WithProgress(fn func(done, total int)) *Engine {
	if e == nil {
		return nil
	}
	e.onProgress = fn
	return e
}

// Run executes one full walk over bars. bars must be non-decreasing by TimestampUnixMs.
func (e *Engine) Run(ctx context.Context, bars []domain.Bar) (*BacktestReport, error) {
	if e == nil || e.feed == nil || e.sim == nil {
		return nil, fmt.Errorf("backtest: incomplete engine")
	}
	if len(bars) == 0 {
		return nil, fmt.Errorf("backtest: no bars")
	}
	init, err := strconv.ParseFloat(strings.TrimSpace(e.cfg.Capital.InitialQuote), 64)
	if err != nil || init <= 0 || math.IsNaN(init) {
		return nil, fmt.Errorf("backtest: invalid capital.initial_quote")
	}

	windowBars := e.feed.FeatSpec.WindowBars
	if windowBars <= 0 {
		windowBars = 96
		e.feed.FeatSpec = marketdata.DefaultFeatureWindowSpec(windowBars)
	}
	featSpec := e.feed.FeatSpec
	need := featSpec.WindowBars + 1
	startIdx := e.cfg.Replay.WarmupBars
	if startIdx < featSpec.WindowBars {
		startIdx = featSpec.WindowBars
	}
	if startIdx+1 < need {
		return nil, fmt.Errorf("backtest: need more bars (warmup/lookback)")
	}
	if startIdx >= len(bars) {
		return nil, fmt.Errorf("backtest: warmup/lookback exceeds series length")
	}

	var port portfolio
	port.balance = init

	var pending []queuedCmd
	var curve []EquityPoint
	var outcomes []SimOutcome
	var cumFees float64
	stepSeq := int64(0)
	totalSteps := len(bars) - startIdx
	if totalSteps < 0 {
		totalSteps = 0
	}

	for idx := startIdx; idx < len(bars); idx++ {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		bar := bars[idx]
		mark := bar.Close
		var barDur int64 = 60_000
		if idx > 0 {
			d := bar.TimestampUnixMs - bars[idx-1].TimestampUnixMs
			if d > 0 {
				barDur = d
			}
		}

		port.balance = ApplyFunding(port.balance, port.netQty, mark, barDur, e.cfg.Simulation.FundingBpsPerDay)

		var tradedNotionalStep float64
		pending, tradedNotionalStep = e.flushPending(pending, idx, bar, &port, &outcomes, &cumFees)

		stepSeq++
		in, err := e.feed.BuildInput(ctx, bars, idx, port.netQty, port.shortOpenMS, stepSeq, strategy.RiskSnapshot{})
		if err != nil {
			return nil, err
		}

		out, err := strategy.Step(in)
		if err != nil {
			return nil, fmt.Errorf("backtest: strategy step: %w", err)
		}

		deadline := in.NowUnixMs + e.deadMS
		for i := range out.Intents {
			intent := out.Intents[i]
			tc := e.tradeCommandFromIntent(in, intent, deadline)
			if e.delay <= 0 {
				before := port.netQty
				simO, tns := e.executeSim(tc, mark, &port, &cumFees)
				tradedNotionalStep += tns
				outcomes = append(outcomes, simO)
				port.syncShortOpened(bar.TimestampUnixMs, before)
			} else {
				pending = append(pending, queuedCmd{dueIdx: idx + e.delay, cmd: tc})
			}
		}

		mark = bar.Close
		eq := port.equity(mark)
		curve = append(curve, EquityPoint{
			UnixMs:         bar.TimestampUnixMs,
			Equity:         eq,
			Balance:        port.balance,
			NetPosition:    port.netQty,
			Mark:           mark,
			StepSequence:   stepSeq,
			TradedNotional: tradedNotionalStep,
		})
		done := idx - startIdx + 1
		if e.onProgress != nil && totalSteps > 0 {
			e.onProgress(done, totalSteps)
		}
	}

	rep := BuildReport(init, curve, outcomes, cumFees)
	return &rep, nil
}

func (e *Engine) flushPending(
	pending []queuedCmd,
	idx int,
	bar domain.Bar,
	port *portfolio,
	outcomes *[]SimOutcome,
	cumFees *float64,
) ([]queuedCmd, float64) {
	mark := bar.Close
	var keep []queuedCmd
	var traded float64
	for _, q := range pending {
		if q.dueIdx > idx {
			keep = append(keep, q)
			continue
		}
		before := port.netQty
		simO, tns := e.executeSim(q.cmd, mark, port, cumFees)
		traded += tns
		*outcomes = append(*outcomes, simO)
		port.syncShortOpened(bar.TimestampUnixMs, before)
	}
	return keep, traded
}

func (e *Engine) executeSim(tc command.TradeCommand, mark float64, port *portfolio, cumFees *float64) (SimOutcome, float64) {
	simO, _, err := e.sim.Run(tc, mark, port.netQty)
	if err != nil {
		simO.Status = command.CommandStatusRejected
		simO.Message = err.Error()
		return simO, 0
	}
	if simO.Status == command.CommandStatusRejected {
		return simO, 0
	}
	port.applySignedFill(simO.FillQtyBase, simO.FillPrice, simO.FeePaid)
	*cumFees += simO.FeePaid
	notional := math.Abs(simO.FillQtyBase * simO.FillPrice)
	return simO, notional
}

func (e *Engine) tradeCommandFromIntent(in strategy.AltShortStrategyInput, intent strategy.TradeIntent, deadline int64) command.TradeCommand {
	id := uuid.NewString()
	inst := "backtest"
	stratID := strings.TrimSpace(e.cfg.Model.ID)
	if stratID == "" {
		stratID = "backtest_model"
	}
	idem := command.IdempotencyKey(fmt.Sprintf("bt:inst:%s:step:%d:intent:%s", inst, in.StepSequence, intent.IntentID))
	return command.TradeCommand{
		CommandID:      id,
		InstanceID:     inst,
		StrategyID:     stratID,
		Symbol:         in.Symbol,
		Side:           intent.Side,
		Intent:         intent,
		TargetNotional: intent.TargetNotionalUSDT,
		TargetPosition: intent.TargetPositionQty,
		ReduceOnly:     intent.IsReduceOnly,
		DeadlineUnixMs: deadline,
		Nonce:          uuid.NewString(),
		IdempotencyKey: idem,
		Kind:           command.CommandKindPlace,
	}
}

// ProgressFunc 可选进度回调：done/total 为有效步进区间内的已完成步数与总步数。
type ProgressFunc func(done int, total int)

// BacktestFromConfig loads bars via file loader, applies replay window, runs Engine。
// progress 为可选变参（至多一项），由控制面任务用于写进度。
func BacktestFromConfig(ctx context.Context, cfg config.BacktestConfig, log *slog.Logger, progress ...ProgressFunc) (*BacktestReport, error) {
	twS, twE, err := ParseReplayWindow(cfg.Replay.Window.Start, cfg.Replay.Window.End)
	if err != nil {
		return nil, err
	}
	loader := &FileBarLoader{RootPath: cfg.Data.Path}
	bars, err := loader.LoadRange(ctx, cfg.Data.Symbol)
	if err != nil {
		return nil, err
	}
	bars = FilterBarsTimeWindow(bars, twS, twE)
	stride := cfg.Replay.BarStride
	if stride <= 0 {
		stride = 1
	}
	bars = DecimateBars(bars, stride)
	if len(bars) == 0 {
		return nil, fmt.Errorf("backtest: no bars after time filter")
	}

	feed := &DataFeed{
		Symbol:   cfg.Data.Symbol,
		FeatSpec: marketdata.DefaultFeatureWindowSpec(96),
		LPPL:     NewLPPLAugmentorFromConfig(cfg.LPPL.Enabled, cfg.LPPL.BubbleMetric01, cfg.LPPL.JobID),
	}
	if cfg.Model.Values != nil {
		if v, ok := cfg.Model.Values["signal_lookback"]; ok {
			w := int(v)
			if w > 0 {
				feed.FeatSpec = marketdata.DefaultFeatureWindowSpec(w)
			}
		}
	}

	var rng *rand.Rand
	if cfg.Simulation.RNGSeed != 0 {
		rng = rand.New(rand.NewSource(cfg.Simulation.RNGSeed))
	} else {
		rng = rand.New(rand.NewSource(time.Now().UnixNano()))
	}
	sim := NewSimulator(SimParams{
		MakerBps:         cfg.Fees.MakerBps,
		TakerBps:         cfg.Fees.TakerBps,
		SlippageBps:      cfg.Slippage.Bps,
		UseTaker:         cfg.Simulation.UseTakerFees,
		FailureRate:      cfg.Simulation.FailureRate,
		PartialProb:      cfg.Simulation.PartialFillProb,
		PartialMinFrac:   cfg.Simulation.PartialFillMinFrac,
		PartialMaxFrac:   cfg.Simulation.PartialFillMaxFrac,
		LotStep:          cfg.Simulation.LotStep,
		FundingBpsPerDay: cfg.Simulation.FundingBpsPerDay,
		RNG:              rng,
	})

	eng := NewEngine(cfg, feed, sim, log)
	if len(progress) > 0 && progress[0] != nil {
		eng = eng.WithProgress(progress[0])
	}
	return eng.Run(ctx, bars)
}

// RunOnce is kept for backward compatibility with older callers; it runs a minimal in-process demo when bars are empty (returns error).
func RunOnce(_ *Engine, _ *slog.Logger) error {
	return fmt.Errorf("backtest.RunOnce is deprecated: use Engine.Run or BacktestFromConfig")
}
