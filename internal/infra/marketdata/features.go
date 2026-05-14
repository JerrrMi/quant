package marketdata

import (
	"fmt"
	"math"
	"strconv"

	"github.com/JerrrMi/quant/internal/domain"
	"github.com/JerrrMi/quant/internal/domain/strategy"
)

// FeatureWindowSpec 描述写入 MarketFeatureSnapshot 的键名约定（编排侧常量，非策略逻辑）。
type FeatureWindowSpec struct {
	WindowBars int
	ZScoreEps  float64
	// NormalizedLogReturnKey 当前点 log return 经 NormalizeTanh01 后的键。
	NormalizedLogReturnKey string
	// WindowLogReturnZKey 当前 log return 相对前序窗口的 Z-score，经 tanh 压限。
	WindowLogReturnZKey string
}

// DefaultFeatureWindowSpec 提供与 MinimalShort 骨架兼容的占位键名。
func DefaultFeatureWindowSpec(windowBars int) FeatureWindowSpec {
	return FeatureWindowSpec{
		WindowBars:             windowBars,
		ZScoreEps:              1e-8,
		NormalizedLogReturnKey: "lr_tanh01",
		WindowLogReturnZKey:    "lr_z_tanh",
	}
}

// BuildFeatureSnapshot 从价格窗口构造策略输入特征切片（无量纲/归一化占位）。
func BuildFeatureSnapshot(window []domain.Bar, spec FeatureWindowSpec) (strategy.MarketFeatureSnapshot, error) {
	if spec.WindowBars <= 0 {
		return strategy.MarketFeatureSnapshot{}, fmt.Errorf("marketdata: window bars must be positive")
	}
	w := BuildPriceWindow(window, spec.WindowBars)
	if len(w) == 0 {
		return strategy.MarketFeatureSnapshot{}, fmt.Errorf("marketdata: empty window")
	}
	feat := strategy.MarketFeatureSnapshot{
		Normalized:  map[string]float64{},
		WindowStats: map[string]float64{},
		RawTags:     map[string]string{},
	}
	if len(w) >= 2 {
		if lr, err := LogReturnLast(w); err == nil {
			feat.Normalized[spec.NormalizedLogReturnKey] = NormalizeTanh01(lr)
		}
	}
	if len(w) >= 3 {
		if z, err := WindowZScoreLast(w, spec.ZScoreEps); err == nil {
			feat.Normalized[spec.WindowLogReturnZKey] = math.Tanh(z)
		}
		closes := make([]float64, len(w))
		for i, b := range w {
			closes[i] = b.Close
		}
		if lrSeries, err := LogReturnsFromCloses(closes); err == nil && len(lrSeries) > 0 {
			mean, std := WindowMeanStd(lrSeries)
			feat.WindowStats["logret_mean_"+strconv.Itoa(spec.WindowBars)] = mean
			feat.WindowStats["logret_std_"+strconv.Itoa(spec.WindowBars)] = std
		}
	}
	return feat, nil
}
