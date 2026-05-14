package lppl

import (
	"context"
	"strconv"

	"github.com/JerrrMi/quant/internal/domain/strategy"
)

const (
	FeatureKeyBubbleMetric = "lppl_bubble_metric_01"
	RawTagJobID            = "lppl_job_id"
)

// InputAugmentor 将最新 LPPL 扫描结果融合到策略输入（仅填特征与标签，不改交易状态字段）。
type InputAugmentor struct {
	Store ResultStore
}

// ApplyLatest 读取 symbol 最新扫描结果并写入 in.Features；无结果时 no-op。
func (a *InputAugmentor) ApplyLatest(ctx context.Context, in *strategy.AltShortStrategyInput, symbol string) error {
	if a == nil || a.Store == nil || in == nil {
		return nil
	}
	v, err := a.Store.LatestBySymbol(ctx, symbol)
	if err != nil || v == nil {
		return err
	}
	sc := ParseScalars(v)
	if in.Features.Normalized == nil {
		in.Features.Normalized = map[string]float64{}
	}
	if in.Features.RawTags == nil {
		in.Features.RawTags = map[string]string{}
	}
	if sc.BubbleMetric01 != 0 {
		in.Features.Normalized[FeatureKeyBubbleMetric] = sc.BubbleMetric01
	}
	if v.JobID != "" {
		in.Features.RawTags[RawTagJobID] = v.JobID
	}
	if v.ComputedAt.Unix() != 0 {
		in.Features.RawTags["lppl_computed_unix"] = strconv.FormatInt(v.ComputedAt.Unix(), 10)
	}
	return nil
}
