package strategy_test

import (
	"errors"
	"math"
	"reflect"
	"testing"

	"github.com/JerrrMi/quant/internal/domain"
	strat "github.com/JerrrMi/quant/internal/domain/strategy"
)

// 骨架当前读取的综合风险占位键。
const riskCompositeKey = strat.RiskCompositeNormalizedKey

func TestStepPureAndDeterministic(t *testing.T) {
	t.Parallel()
	in := baseInputFlat()
	a, err := strat.Step(in)
	if err != nil {
		t.Fatal(err)
	}
	b, err := strat.MinimalShortStep(in)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(a, b) {
		t.Fatalf("Step and MinimalShortStep diverged:\n%+v\nvs\n%+v", a, b)
	}
	c, err := strat.Step(in)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(a, c) {
		t.Fatal("same input produced different outputs")
	}
}

func TestOpenShortWhenMomentumAndRiskAllow(t *testing.T) {
	t.Parallel()
	in := baseInputFlat()
	// Negative log-return vs prior close — entryThresh 默认 -5e-4。
	in.PriorBarClose = 100
	in.BarCurrent.Close = 99.4
	in.Features.Normalized["risk_composite_01"] = 0.1

	out, err := strat.Step(in)
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Intents) != 1 {
		t.Fatalf("expected one intent, got %#v", out.Intents)
	}
	if got := out.Signal.ReasonCodes[0]; got != strat.ReasonCodeOpenShort {
		t.Fatalf("first reason %q want %s", got, strat.ReasonCodeOpenShort)
	}
	if out.Intents[0].Side != domain.SideSell {
		t.Fatalf("open short should Sell side, got %s", out.Intents[0].Side)
	}
	if out.Signal.ValidUntilUnixMs != in.NowUnixMs+3600_000 {
		t.Fatalf("valid until mismatch: got %d want %d", out.Signal.ValidUntilUnixMs, in.NowUnixMs+3600_000)
	}
}

func TestRiskCompositeBlocksFreshShort(t *testing.T) {
	t.Parallel()
	in := baseInputFlat()
	in.PriorBarClose = 100
	in.BarCurrent.Close = 99.4
	in.Features.Normalized["risk_composite_01"] = 0.9

	out, err := strat.Step(in)
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Intents) != 0 {
		t.Fatalf("expected no intents, got %#v", out.Intents)
	}
	found := false
	for _, c := range out.Signal.ReasonCodes {
		if c == strat.ReasonCodeHoldHighRiskFloor {
			found = true
		}
	}
	if !found {
		t.Fatalf("codes %#v missing %s", out.Signal.ReasonCodes, strat.ReasonCodeHoldHighRiskFloor)
	}
}

func TestTradingHaltedHold(t *testing.T) {
	t.Parallel()
	in := baseInputFlat()
	in.Risk.TradingHalted = true
	out, err := strat.Step(in)
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Intents) != 0 || out.Signal.ReasonCodes[1] != strat.ReasonCodeHoldRiskHalted {
		t.Fatalf("unexpected %+v", out)
	}
}

func TestHoldShortCarryUntilExit(t *testing.T) {
	t.Parallel()
	in := baseInputShort(-1)
	in.ShortOpenedAtUnixMs = 1000
	in.NowUnixMs = 1500
	in.PriorBarClose = 100
	in.BarCurrent.Close = 100.0001 // rebound below ladders
	in.Minimal = &strat.MinimalSkeletonParams{
		MaxShortHoldMs: 10_000,
	}
	out, err := strat.Step(in)
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Intents) != 0 {
		t.Fatalf("expected carry hold without intents %#v", out.Intents)
	}
	if out.Signal.ReasonCodes[1] != strat.ReasonCodeHoldShortCarry {
		t.Fatalf("codes %#v", out.Signal.ReasonCodes)
	}
}

func TestCloseShortOnExpiryTTL(t *testing.T) {
	t.Parallel()
	in := baseInputShort(-2)
	in.ShortOpenedAtUnixMs = 1000
	in.NowUnixMs = 4600 // +3600 ms default TTL
	in.PriorBarClose = 100
	in.BarCurrent.Close = 100.001
	in.Minimal = &strat.MinimalSkeletonParams{MaxShortHoldMs: 3600}

	out, err := strat.Step(in)
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Intents) != 1 || out.Intents[0].Side != domain.SideBuy {
		t.Fatalf("close short expected SideBuy %#v", out.Intents)
	}
	if out.Signal.ReasonCodes[0] != strat.ReasonCodeCloseShort {
		t.Fatal(out.Signal.ReasonCodes)
	}
}

func TestCloseShortStopLossLadder(t *testing.T) {
	t.Parallel()
	in := baseInputShort(-1)
	in.ShortOpenedAtUnixMs = 100
	in.NowUnixMs = 200
	in.Minimal = &strat.MinimalSkeletonParams{
		MaxShortHoldMs:    999_999,
		StopLossLogReturn: 5e-4,
	}
	in.PriorBarClose = 100
	in.BarCurrent.Close = 100.06 // ln(1.0006)>5e-4

	out, err := strat.Step(in)
	if err != nil {
		t.Fatal(err)
	}
	if out.Signal.ReasonCodes[0] != strat.ReasonCodeCloseShort || len(out.Intents) != 1 {
		t.Fatalf("%+v", out)
	}
}

func TestReduceShortBand(t *testing.T) {
	t.Parallel()
	in := baseInputShort(-2)
	in.ShortOpenedAtUnixMs = 10
	in.NowUnixMs = 20
	in.Minimal = &strat.MinimalSkeletonParams{
		MaxShortHoldMs:      999_999,
		ReduceLogReturnMin:  2e-4,
		TakeProfitLogReturn: 8e-4,
		StopLossLogReturn:   5e-3,
	}
	in.PriorBarClose = 100
	in.BarCurrent.Close = 100.03 // ln(1.0003) ~ 3e-4

	out, err := strat.Step(in)
	if err != nil {
		t.Fatal(err)
	}
	if out.Signal.ReasonCodes[0] != strat.ReasonCodeReduceShort || len(out.Intents) != 1 {
		t.Fatalf("%+v intents=%v", out.Signal, out.Intents)
	}
	if math.Abs(*out.Intents[0].TargetPositionQty-(-1)) > 1e-9 {
		t.Fatalf("expected half qty -1 got %v", *out.Intents[0].TargetPositionQty)
	}
}

func TestHoldLongExposure(t *testing.T) {
	t.Parallel()
	in := baseInputFlat()
	in.NetPositionQty = 0.5
	out, err := strat.Step(in)
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Intents) != 0 {
		t.Fatal(out.Intents)
	}
	if out.Signal.ReasonCodes[1] != strat.ReasonCodeHoldLongBias {
		t.Fatal(out.Signal.ReasonCodes)
	}
}

func TestEdgeEmptySymbolRejected(t *testing.T) {
	t.Parallel()
	in := baseInputFlat()
	in.Symbol = "   "
	_, err := strat.Step(in)
	if !errors.Is(err, strat.ErrEmptySymbol) {
		t.Fatalf("got %v", err)
	}
}

func TestEdgeNonFiniteClose(t *testing.T) {
	t.Parallel()
	in := baseInputFlat()
	in.BarCurrent.Close = math.NaN()
	_, err := strat.Step(in)
	if !errors.Is(err, strat.ErrNonFiniteBars) {
		t.Fatalf("got %v", err)
	}
}

func TestEdgeNonFiniteLeverageRisk(t *testing.T) {
	t.Parallel()
	in := baseInputFlat()
	in.Risk.MaxLeverage = math.Inf(1)
	_, err := strat.Step(in)
	if !errors.Is(err, strat.ErrNonFiniteRisk) {
		t.Fatalf("got %v", err)
	}
}

func TestEdgeNonFiniteQty(t *testing.T) {
	t.Parallel()
	in := baseInputFlat()
	in.NetPositionQty = math.Inf(-1)
	_, err := strat.Step(in)
	if !errors.Is(err, strat.ErrNonFiniteQty) {
		t.Fatalf("got %v", err)
	}
}

func TestPriorCloseZeroNeutralLogReturn(t *testing.T) {
	t.Parallel()
	in := baseInputFlat()
	in.PriorBarClose = 0 // 无回看价：骨架把 log-return 归零，开仓条件不触发
	in.BarCurrent.Close = 50
	in.Features.Normalized["risk_composite_01"] = 0

	out, err := strat.Step(in)
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Intents) != 0 || out.Signal.Name != "minimal_hold_flat" {
		t.Fatalf("%+v / %#v", out.Signal, out.Intents)
	}
}

func TestDrawdownNearOneBlocksEntryEvenWithMomentum(t *testing.T) {
	t.Parallel()
	in := baseInputFlat()
	in.Risk.CurrentDrawdown01 = 0.985
	in.PriorBarClose = 100
	in.BarCurrent.Close = 97
	out, err := strat.Step(in)
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Intents) != 0 {
		t.Fatalf("drawdown gate should suppress entry %#v", out.Intents)
	}
}

func baseInputFlat() strat.AltShortStrategyInput {
	return strat.AltShortStrategyInput{
		Symbol:         "BTCUSDT",
		NetPositionQty: 0,
		PriorBarClose:  100,
		BarCurrent: domain.Bar{
			Open: 100, High: 101, Low: 99, Close: 100.01,
			Volume: 123, TimestampUnixMs: 42,
		},
		Features: strat.MarketFeatureSnapshot{
			Normalized: map[string]float64{
				riskCompositeKey: 0.05,
			},
		},
		Risk:         strat.RiskSnapshot{},
		NowUnixMs:    900_000,
		StepSequence: 11,
	}
}

func baseInputShort(qty float64) strat.AltShortStrategyInput {
	in := baseInputFlat()
	in.NetPositionQty = qty
	in.Features.Normalized[riskCompositeKey] = 0.05
	return in
}
