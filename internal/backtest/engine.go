// Package backtest 提供回测引擎与数据加载占位；不访问实盘密钥。
package backtest

import (
	"context"
	"log/slog"

	"github.com/altshort/quant/internal/domain"
)

// HistoricalLoader 抽象历史 K 线/成交来源；后续可接文件、DB、Parquet。
type HistoricalLoader interface {
	// LoadRange 返回区间内按时间排序的 Bar 序列；占位返回空切片。
	LoadRange(ctx context.Context, symbol string) ([]domain.Bar, error)
}

// Loader 是 HistoricalLoader 的骨架实现。
type Loader struct{}

// NewHistoricalLoader 构造加载器；由 RunBacktest 与 Engine 注入。
func NewHistoricalLoader() *Loader {
	return &Loader{}
}

// LoadRange 占位实现：暂无数据。
func (l *Loader) LoadRange(ctx context.Context, symbol string) ([]domain.Bar, error) {
	_ = l
	_ = ctx
	_ = symbol
	return nil, nil
}

// Engine 驱动回放与费用/滑点模型占位。
type Engine struct {
	loader HistoricalLoader
}

// NewEngine 创建回测引擎；loader 由 HistoricalLoader 实现方传入。
func NewEngine(loader HistoricalLoader) *Engine {
	return &Engine{loader: loader}
}

// RunOnce 执行单轮占位回放：记录日志后返回，供 cmd/backtest 验证闭环。
func RunOnce(e *Engine, log *slog.Logger) error {
	if log != nil {
		log.Info("backtest.RunOnce: no bars loaded yet (stub)")
	}
	_ = e.loader
	return nil
}
