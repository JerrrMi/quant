package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/altshort/quant/internal/app"
	"github.com/altshort/quant/internal/infra"
	"github.com/altshort/quant/internal/lifecycle"
)

func main() {
	if err := run(context.Background()); err != nil {
		slog.Error("backtest exited", "err", err)
		os.Exit(1)
	}
}

// run 初始化回测依赖并调用 app.RunBacktest；历史数据加载器、引擎与策略调用链在 internal/backtest 逐步充实。
func run(ctx context.Context) error {
	slog.Info("AltShort Backtest process starting")

	log := infra.NewLogger("backtest", "info")
	// 回测可不连持久库；传 nil DB 表示纯内存路径。
	deps := lifecycle.NewBootstrapDeps(log, nil)

	// --- 预留：internal/backtest HistoricalLoader 从磁盘/DB 加载 ---
	// --- 预留：Engine 接入 domain.Strategy Step ---

	if err := app.RunBacktest(ctx, deps); err != nil {
		return fmt.Errorf("app.RunBacktest: %w", err)
	}
	return nil
}
