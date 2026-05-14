package backtest

import "github.com/JerrrMi/quant/internal/domain"

// DecimateBars keeps every stride-th bar (1-based stride). stride<=1 is a no-op.
// 用于控制台「回放步进」：在不解耦 CSV 时间框架的前提下做降采样。
func DecimateBars(bars []domain.Bar, stride int) []domain.Bar {
	if stride <= 1 || len(bars) == 0 {
		return bars
	}
	out := make([]domain.Bar, 0, len(bars)/stride+1)
	for i := 0; i < len(bars); i += stride {
		out = append(out, bars[i])
	}
	return out
}
