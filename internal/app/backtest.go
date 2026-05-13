package app

import (
	"context"
	"log/slog"

	"github.com/altshort/quant/internal/backtest"
	"github.com/altshort/quant/internal/lifecycle"
)

// RunBacktest 编排回测占位的调用链：历史加载器 → 引擎 → 策略调用器。
// 由 cmd/backtest 的 run() 调用；deps.DB 可为 nil（纯内存回放）。
func RunBacktest(ctx context.Context, deps lifecycle.BootstrapDeps) error {
	_ = ctx
	log := deps.Logger
	if log == nil {
		log = slog.Default()
	}
	loader := backtest.NewHistoricalLoader()
	engine := backtest.NewEngine(loader)
	log.Info("Backtest stub wired", "has_engine", engine != nil)
	// TODO: 接入 domain 策略 Step 与 Bar/Candle 输入
	return backtest.RunOnce(engine, log)
}
