// Package marketdata 提供市场快照与特征窗口的编排侧抽象；不做交易所直连。
package marketdata

import (
	"context"
	"fmt"

	"github.com/JerrrMi/quant/internal/domain"
)

// SnapshotPayload 为落库或跨进程传递用的快照载荷形状（JSON 友好）。
// 与 models.MarketSnapshot.PayloadJSON 对齐；新增字段须向后兼容解码。
type SnapshotPayload struct {
	Bar           domain.Bar `json:"bar"`
	BidPrice      float64    `json:"bid_price,omitempty"`
	AskPrice      float64    `json:"ask_price,omitempty"`
	LastTradePx   float64    `json:"last_trade_px,omitempty"`
	SourceEventMs int64      `json:"source_event_unix_ms,omitempty"`
}

// ClosedBarSnapshotReader 返回最新一根「已完结」K 线量级的快照（原始量纲）。
type ClosedBarSnapshotReader interface {
	LatestClosedBar(ctx context.Context, symbol string) (domain.Bar, error)
}

// PriceWindowReader 返回按时间正序排列的最近 closed bars（含最新一根）。
type PriceWindowReader interface {
	RecentClosedBars(ctx context.Context, symbol string, max int) ([]domain.Bar, error)
}

// BuildPriceWindow 从全序列尾部截取至多 window 根；不足则返回已有前缀。
func BuildPriceWindow(bars []domain.Bar, window int) []domain.Bar {
	if window <= 0 || len(bars) == 0 {
		return nil
	}
	if len(bars) <= window {
		out := make([]domain.Bar, len(bars))
		copy(out, bars)
		return out
	}
	return bars[len(bars)-window:]
}

// PriorClose 返回窗口中倒数第二根收盘价；元素不足 2 时返回 0, false。
func PriorClose(window []domain.Bar) (float64, bool) {
	if len(window) < 2 {
		return 0, false
	}
	return window[len(window)-2].Close, true
}

// ValidateMonotonicTimestamps 确保 bars 时间非递减；用于管线自检。
func ValidateMonotonicTimestamps(bars []domain.Bar) error {
	var prev int64
	for i, b := range bars {
		if b.TimestampUnixMs < prev {
			return fmt.Errorf("marketdata: bar %d timestamp regresses", i)
		}
		prev = b.TimestampUnixMs
	}
	return nil
}
