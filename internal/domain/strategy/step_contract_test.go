package strategy_test

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/JerrrMi/quant/internal/domain"
	strat "github.com/JerrrMi/quant/internal/domain/strategy"
)

// 约束：`Step` 输入/输出须可被稳定 JSON 序列化（编排、审计、跨进程协议）。
func TestStepInputOutput_JSONSnapshotRoundtrip(t *testing.T) {
	t.Parallel()
	in := baseInputFlat()
	in.Features.WindowStats = map[string]float64{"vol_96": 0.3}
	in.Features.RawTags = map[string]string{"regime": "A"}
	rawIn, err := json.Marshal(in)
	if err != nil {
		t.Fatal(err)
	}
	var decoded strat.AltShortStrategyInput
	if err := json.Unmarshal(rawIn, &decoded); err != nil {
		t.Fatal(err)
	}
	out1, err := strat.Step(decoded)
	if err != nil {
		t.Fatal(err)
	}
	out2, err := strat.Step(in)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(out1, out2) {
		t.Fatalf("json roundtrip input changed semantics:\n%+v\nvs\n%+v", out1, out2)
	}
	rawOut, err := json.Marshal(out1)
	if err != nil {
		t.Fatal(err)
	}
	var outAgain strat.AltShortStrategyOutput
	if err := json.Unmarshal(rawOut, &outAgain); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(out1, outAgain) {
		t.Fatal("output json roundtrip drift")
	}
}

// 空特征图视为低风险占位（缺失键 → 0），不得 panic 或依赖墙钟。
func TestStep_emptyFeatureMaps(t *testing.T) {
	t.Parallel()
	in := baseInputFlat()
	in.Features.Normalized = nil
	in.Features.WindowStats = nil
	in.Features.RawTags = nil
	in.PriorBarClose = 100
	in.BarCurrent.Close = 99.4
	out, err := strat.Step(in)
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Intents) != 1 {
		t.Fatalf("expected open-short on empty maps, got %#v", out.Intents)
	}
}

// nil Minimal 必须使用包内默认阈值，行为与未显式注入一致。
func TestStep_nilMinimalUsesDefaults(t *testing.T) {
	t.Parallel()
	in := baseInputFlat()
	in.Minimal = nil
	in.PriorBarClose = 100
	in.BarCurrent.Close = 99.4
	out, err := strat.Step(in)
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Intents) != 1 {
		t.Fatal(out.Intents)
	}
}

// 止盈分支：空仓且对数收益超过默认 takeProfitLn → 平空意图。
func TestStep_takeProfitClosesShort(t *testing.T) {
	t.Parallel()
	in := baseInputShort(-1)
	in.ShortOpenedAtUnixMs = 1
	in.NowUnixMs = 500
	in.PriorBarClose = 100
	in.BarCurrent.Close = 100.05 // ln(1.0005) > 默认 4e-4
	out, err := strat.Step(in)
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Intents) != 1 || out.Intents[0].Side != domain.SideBuy {
		t.Fatalf("take profit close expected, got %#v", out.Intents)
	}
	if out.Signal.ReasonCodes[0] != strat.ReasonCodeCloseShort {
		t.Fatal(out.Signal.ReasonCodes)
	}
}

// Zero bar timestamp is allowed if OHLC finite — 调度时钟只用 NowUnixMs。
func TestStep_zeroBarTimestampStillPure(t *testing.T) {
	t.Parallel()
	in := baseInputFlat()
	in.BarCurrent.TimestampUnixMs = 0
	out, err := strat.Step(in)
	if err != nil {
		t.Fatal(err)
	}
	if out.Signal.Name == "" {
		t.Fatal("expected named signal")
	}
}
