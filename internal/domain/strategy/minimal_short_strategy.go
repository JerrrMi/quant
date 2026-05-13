package strategy

import (
	"errors"
	"fmt"
	"math"
	"strings"

	"github.com/JerrrMi/quant/internal/domain"
)

var (
	ErrEmptySymbol   = errors.New("strategy: empty symbol")
	ErrNonFiniteBars = errors.New("strategy: bar prices must be finite positive numbers")
	ErrNonFiniteRisk = errors.New("strategy: risk snapshot contains non-finite values")
	ErrNonFiniteQty  = errors.New("strategy: net position qty is non-finite")
)

// 机器可读意图/占位枚举（供审计或与 Command 映射）。
const (
	ReasonCodeOpenShort         = "OPEN_SHORT"
	ReasonCodeCloseShort        = "CLOSE_SHORT"
	ReasonCodeReduceShort       = "REDUCE_SHORT"
	ReasonCodeHoldRiskHalted    = "HOLD_RISK_HALTED"
	ReasonCodeHoldNoTrigger     = "HOLD_NO_TRIGGER"
	ReasonCodeHoldHighRiskFloor = "HOLD_HIGH_RISK_BLOCKED"
	ReasonCodeHoldLongBias      = "HOLD_LONG_NOT_MANAGED"
	ReasonCodeHoldShortCarry    = "HOLD_SHORT_CARRY"
)

type resolvedMinimalParams struct {
	maxShortHoldMs      int64
	entryLogReturn      float64
	takeProfitLogReturn float64
	stopLossLogReturn   float64
	reduceLogReturnMin  float64
	openNotionalUSDT    float64
	riskCeilingOpen     float64
	holdValidityExtraMs int64
}

func defaultMinimalParams() resolvedMinimalParams {
	return resolvedMinimalParams{
		maxShortHoldMs:      3_600_000, // holdingMs：空仓最长持有毫秒（逻辑时钟）。
		entryLogReturn:      -5e-4,     // entryThreshLn：≤ 该对数收益才允许开空（下跌占位）。
		takeProfitLogReturn: 4e-4,      // takeProfitLn：价格反弹到一定程度则止盈平空。
		stopLossLogReturn:   12e-4,     // stopLossLn：更强反弹则止损平空。
		reduceLogReturnMin:  2e-4,      // reduceBandLo：介于该值与止盈之间则减仓。
		openNotionalUSDT:    50,
		riskCeilingOpen:     0.82,    // riskOpenBlock：风险高于该水平则禁止新开空。
		holdValidityExtraMs: 180_000, // holdSignalTTL：HOLD 的参考有效期增量（毫秒）。
	}
}

func resolveMinimalParams(min *MinimalSkeletonParams) resolvedMinimalParams {
	base := defaultMinimalParams()
	if min == nil {
		return base
	}
	if min.MaxShortHoldMs > 0 {
		base.maxShortHoldMs = min.MaxShortHoldMs
	}
	if min.EntryLogReturn != 0 {
		base.entryLogReturn = min.EntryLogReturn
	}
	if min.TakeProfitLogReturn > 0 {
		base.takeProfitLogReturn = min.TakeProfitLogReturn
	}
	if min.StopLossLogReturn > 0 {
		base.stopLossLogReturn = min.StopLossLogReturn
	}
	if min.ReduceLogReturnMin > 0 {
		base.reduceLogReturnMin = min.ReduceLogReturnMin
	}
	if base.reduceLogReturnMin >= base.takeProfitLogReturn {
		base.reduceLogReturnMin = base.takeProfitLogReturn * 0.5
	}
	return base
}

// MinimalShortStep 为最小做空骨架：只校验快照与占位阈值；禁止 I/O。
func MinimalShortStep(in AltShortStrategyInput) (AltShortStrategyOutput, error) {
	if strings.TrimSpace(in.Symbol) == "" {
		return AltShortStrategyOutput{}, ErrEmptySymbol
	}
	if err := validateFiniteBars(in.BarCurrent.Close, in.PriorBarClose); err != nil {
		return AltShortStrategyOutput{}, err
	}
	if err := validateRiskFinite(in.Risk); err != nil {
		return AltShortStrategyOutput{}, err
	}
	if !finite(in.NetPositionQty) {
		return AltShortStrategyOutput{}, ErrNonFiniteQty
	}

	p := resolveMinimalParams(in.Minimal)

	normalizedRiskSignal := normalizedScalar(in.Features.Normalized, RiskCompositeNormalizedKey)

	logReturnLn := computeLogReturnLn(in.BarCurrent.Close, in.PriorBarClose)

	const qtyEps = 1e-8
	shortNow := in.NetPositionQty < -qtyEps
	flatNow := math.Abs(in.NetPositionQty) <= qtyEps
	longNow := in.NetPositionQty > qtyEps

	if longNow {
		sig := StrategySignal{
			Name:             "minimal_hold_long_bias",
			Strength:         0,
			ReasonCodes:      []string{"HOLD", ReasonCodeHoldLongBias},
			ReasonDetail:     "骨架只演示 flat→short→flat；净多时不主动撮合",
			ValidUntilUnixMs: in.NowUnixMs + p.holdValidityExtraMs,
			Confidence01:     0.08,
		}
		return labeledOutput(in, normalizedRiskSignal, logReturnLn, sig, nil), nil
	}

	if in.Risk.TradingHalted {
		sig := StrategySignal{
			Name:             "minimal_risk_gate",
			Strength:         0,
			ReasonCodes:      []string{"HOLD", ReasonCodeHoldRiskHalted},
			ReasonDetail:     "risk snapshot halted trading gate",
			ValidUntilUnixMs: in.NowUnixMs + p.holdValidityExtraMs,
			Confidence01:     0.85,
		}
		return labeledOutput(in, normalizedRiskSignal, logReturnLn, sig, nil), nil
	}

	if shortNow {
		out, intents, done := decideShortExposure(in, p, logReturnLn)
		if done {
			return finalizeIntents(in, normalizedRiskSignal, logReturnLn, out, intents), nil
		}
		sig := StrategySignal{
			Name:             "minimal_short_carry",
			Strength:         -0.35,
			ReasonCodes:      []string{"HOLD", ReasonCodeHoldShortCarry},
			ReasonDetail:     "maintaining short stub; exits not triggered",
			ValidUntilUnixMs: in.NowUnixMs + p.holdValidityExtraMs,
			Confidence01:     0.4,
		}
		return labeledOutput(in, normalizedRiskSignal, logReturnLn, sig, nil), nil
	}

	if flatNow && normalizedRiskSignal < p.riskCeilingOpen {
		out, intents, done := maybeOpenShort(in, p, normalizedRiskSignal, logReturnLn)
		if done {
			return finalizeIntents(in, normalizedRiskSignal, logReturnLn, out, intents), nil
		}
		sig := StrategySignal{
			Name:             "minimal_hold_flat",
			Strength:         0,
			ReasonCodes:      []string{"HOLD", ReasonCodeHoldNoTrigger},
			ReasonDetail:     "flat book; entry momentum / risk ladder not crossed",
			ValidUntilUnixMs: in.NowUnixMs + p.holdValidityExtraMs,
			Confidence01:     0.22,
		}
		return labeledOutput(in, normalizedRiskSignal, logReturnLn, sig, nil), nil
	}

	if flatNow {
		sig := StrategySignal{
			Name:             "minimal_risk_block_entry",
			Strength:         0,
			ReasonCodes:      []string{"HOLD", ReasonCodeHoldHighRiskFloor},
			ReasonDetail:     fmt.Sprintf("composite risk %.3f ≥ ceiling %.3f", normalizedRiskSignal, p.riskCeilingOpen),
			ValidUntilUnixMs: in.NowUnixMs + p.holdValidityExtraMs,
			Confidence01:     0.52,
		}
		return labeledOutput(in, normalizedRiskSignal, logReturnLn, sig, nil), nil
	}

	sig := StrategySignal{
		Name:             "minimal_hold_guard",
		Strength:         0,
		ReasonCodes:      []string{"HOLD", ReasonCodeHoldNoTrigger},
		ReasonDetail:     "no branch matched (unexpected qty band)",
		ValidUntilUnixMs: in.NowUnixMs + p.holdValidityExtraMs,
		Confidence01:     0.1,
	}
	return labeledOutput(in, normalizedRiskSignal, logReturnLn, sig, nil), nil
}

func labeledOutput(
	in AltShortStrategyInput,
	normalizedRiskSignal, logReturnLn float64,
	signal StrategySignal,
	intents []TradeIntent,
) AltShortStrategyOutput {
	signal.Strength = confidenceToStrength(signal.Confidence01)
	out := AltShortStrategyOutput{
		Signal:      signal,
		Intents:     intents,
		Diagnostics: diagnosticsBase(in.Symbol, in.NowUnixMs, in.StepSequence, normalizedRiskSignal, logReturnLn, in.NetPositionQty),
	}
	return out
}

func finalizeIntents(
	in AltShortStrategyInput,
	normalizedRiskSignal, logReturnLn float64,
	body AltShortStrategyOutput,
	intents []TradeIntent,
) AltShortStrategyOutput {
	body.Intents = intents
	if body.Diagnostics == nil {
		body.Diagnostics = map[string]float64{}
	}
	for k, v := range diagnosticsBase(in.Symbol, in.NowUnixMs, in.StepSequence, normalizedRiskSignal, logReturnLn, in.NetPositionQty) {
		body.Diagnostics[k] = v
	}
	body.Signal.Strength = confidenceToStrength(body.Signal.Confidence01)
	return body
}

func diagnosticsBase(symbol string, nowMs, seq int64, riskNorm, logRet, netQty float64) map[string]float64 {
	const symLen = "symbol_utf8_len"
	return map[string]float64{
		"log_return_ln":         logRet,
		"risk_normalized_01":    riskNorm,
		"net_position_qty_in":   netQty,
		"reference_clock_ms_in": float64(nowMs),
		"step_sequence_in":      float64(seq),
		symLen:                  float64(len(symbol)),
	}
}

func confidenceToStrength(conf float64) float64 {
	conf = clamp01(conf)
	return 2*conf - 1
}

func clamp01(x float64) float64 {
	switch {
	case x < 0:
		return 0
	case x > 1:
		return 1
	default:
		return x
	}
}

// decideShortExposure 处理已有空仓的场景：expire → stop → take-profit → partial reduce。
func decideShortExposure(in AltShortStrategyInput, p resolvedMinimalParams, logReturnLn float64) (AltShortStrategyOutput, []TradeIntent, bool) {
	held := in.ShortOpenedAtUnixMs > 0 && in.NowUnixMs >= in.ShortOpenedAtUnixMs

	expiredShort := held && p.maxShortHoldMs > 0 && in.NowUnixMs-in.ShortOpenedAtUnixMs >= p.maxShortHoldMs
	if expiredShort {
		zi := float64ZeroPtr()
		sig := StrategySignal{
			Name:             "minimal_short_expire",
			Strength:         0,
			ReasonCodes:      []string{ReasonCodeCloseShort},
			ReasonDetail:     "holding TTL exceeded versus ShortOpenedAtUnixMs",
			ValidUntilUnixMs: in.NowUnixMs + p.maxShortHoldMs/10,
			Confidence01:     0.9,
		}
		intents := []TradeIntent{{
			IntentID:          fmt.Sprintf("close-expire-%d", in.StepSequence),
			Symbol:            in.Symbol,
			Side:              domain.SideBuy,
			TargetPositionQty: zi,
			IsReduceOnly:      true,
			Urgency01:         0.82,
		}}
		diags := map[string]float64{"exit_horizon_trigger": 1}
		return AltShortStrategyOutput{Signal: sig, Diagnostics: diags}, intents, true
	}

	stopHit := logReturnLn >= p.stopLossLogReturn
	if stopHit {
		zi := float64ZeroPtr()
		sig := StrategySignal{
			Name:             "minimal_short_stop_stub",
			Strength:         0,
			ReasonCodes:      []string{ReasonCodeCloseShort},
			ReasonDetail:     " rebound log-return exceeded stop ladder",
			ValidUntilUnixMs: in.NowUnixMs + p.maxShortHoldMs/30,
			Confidence01:     0.78,
		}
		intents := []TradeIntent{{
			IntentID:          fmt.Sprintf("close-sl-%d", in.StepSequence),
			Symbol:            in.Symbol,
			Side:              domain.SideBuy,
			TargetPositionQty: zi,
			IsReduceOnly:      true,
			Urgency01:         1,
		}}
		diags := map[string]float64{"stop_ln_trigger": clamp01(logReturnLn / p.stopLossLogReturn)}
		return AltShortStrategyOutput{Signal: sig, Diagnostics: diags}, intents, true
	}

	tpHit := logReturnLn >= p.takeProfitLogReturn
	if tpHit {
		zi := float64ZeroPtr()
		sig := StrategySignal{
			Name:             "minimal_short_tp_stub",
			Strength:         0,
			ReasonCodes:      []string{ReasonCodeCloseShort},
			ReasonDetail:     "favorable rebound for short profit taking (placeholder)",
			ValidUntilUnixMs: in.NowUnixMs + p.maxShortHoldMs/40,
			Confidence01:     0.7,
		}
		intents := []TradeIntent{{
			IntentID:          fmt.Sprintf("close-tp-%d", in.StepSequence),
			Symbol:            in.Symbol,
			Side:              domain.SideBuy,
			TargetPositionQty: zi,
			IsReduceOnly:      true,
			Urgency01:         0.55,
		}}
		diags := map[string]float64{"take_profit_ln_trigger": clamp01(logReturnLn / p.takeProfitLogReturn)}
		return AltShortStrategyOutput{Signal: sig, Diagnostics: diags}, intents, true
	}

	reduceBand := logReturnLn >= p.reduceLogReturnMin && logReturnLn < p.takeProfitLogReturn
	if reduceBand && math.Abs(in.NetPositionQty) > 1e-6 {
		half := in.NetPositionQty * 0.5
		sig := StrategySignal{
			Name:             "minimal_short_reduce_stub",
			Strength:         -0.2,
			ReasonCodes:      []string{ReasonCodeReduceShort},
			ReasonDetail:     "micro rebound window: lighten short exposure halfway",
			ValidUntilUnixMs: in.NowUnixMs + p.holdValidityExtraMs,
			Confidence01:     0.48,
		}
		intents := []TradeIntent{{
			IntentID:          fmt.Sprintf("reduce-%d", in.StepSequence),
			Symbol:            in.Symbol,
			Side:              domain.SideBuy,
			TargetPositionQty: &half,
			IsReduceOnly:      true,
			Urgency01:         0.42,
		}}
		diags := map[string]float64{"reduce_ln_ratio": clamp01((logReturnLn - p.reduceLogReturnMin) / (p.takeProfitLogReturn - p.reduceLogReturnMin + 1e-12))}
		return AltShortStrategyOutput{Signal: sig, Diagnostics: diags}, intents, true
	}

	return AltShortStrategyOutput{}, nil, false
}

func float64ZeroPtr() *float64 {
	z := 0.0
	return &z
}

func maybeOpenShort(in AltShortStrategyInput, p resolvedMinimalParams, riskNorm, logReturnLn float64) (AltShortStrategyOutput, []TradeIntent, bool) {
	// drawGuard：极端回撤占位；避免在快照标记的高消耗区间继续加注（非完整风控）。
	drawGuard := in.Risk.CurrentDrawdown01 <= 0.97
	if !(drawGuard && logReturnLn <= p.entryLogReturn) {
		return AltShortStrategyOutput{}, nil, false
	}
	notional := p.openNotionalUSDT
	conf := clamp01(0.78 - riskNorm*0.28)
	sig := StrategySignal{
		Name:             "minimal_short_entry_stub",
		ReasonCodes:      []string{ReasonCodeOpenShort},
		ReasonDetail:     "simple negative log-return vs entry ladder (pipeline smoke test)",
		ValidUntilUnixMs: in.NowUnixMs + p.maxShortHoldMs,
		Confidence01:     conf,
	}
	intents := []TradeIntent{{
		IntentID:           fmt.Sprintf("open-short-%d", in.StepSequence),
		Symbol:             in.Symbol,
		Side:               domain.SideSell,
		TargetNotionalUSDT: &notional,
		Urgency01:          0.38,
	}}
	margin := clamp01((p.entryLogReturn - logReturnLn) / (math.Abs(p.entryLogReturn) + 1e-12))
	diags := map[string]float64{"entry_ln_margin": margin}
	return AltShortStrategyOutput{Signal: sig, Diagnostics: diags}, intents, true
}

func validateFiniteBars(close, priorBarClose float64) error {
	if !finite(close) || close <= 0 {
		return ErrNonFiniteBars
	}
	if priorBarClose < 0 || (priorBarClose > 0 && !finite(priorBarClose)) {
		return ErrNonFiniteBars
	}
	return nil
}

func validateRiskFinite(r RiskSnapshot) error {
	for _, x := range []float64{r.MaxLeverage, r.MaxNotionalUSDT, r.CurrentDrawdown01} {
		if !finite(x) {
			return ErrNonFiniteRisk
		}
	}
	return nil
}

func finite(x float64) bool {
	return !math.IsNaN(x) && !math.IsInf(x, 0)
}

func computeLogReturnLn(close, priorClose float64) float64 {
	if priorClose <= 0 {
		return 0
	}
	r := math.Log(close / priorClose)
	if finite(r) {
		return r
	}
	return 0
}

func normalizedScalar(m map[string]float64, key string) float64 {
	if m == nil {
		return 0
	}
	v, ok := m[key]
	if !ok || !finite(v) {
		return 0
	}
	switch {
	case v < 0:
		return 0
	case v > 1:
		return 1
	default:
		return v
	}
}

// MinimalShortStrategy 是无状态骨架实现，满足 Stepper，便于在回测或实盘中按接口注入。
type MinimalShortStrategy struct{}

// Step implements Stepper.
func (MinimalShortStrategy) Step(in AltShortStrategyInput) (AltShortStrategyOutput, error) {
	return MinimalShortStep(in)
}

var _ Stepper = MinimalShortStrategy{}
