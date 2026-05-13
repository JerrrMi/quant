package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/altshort/quant/internal/app"
	"github.com/altshort/quant/internal/config"
	"github.com/altshort/quant/internal/infra"
	"github.com/altshort/quant/internal/lifecycle"
)

func main() {
	if err := run(context.Background()); err != nil {
		slog.Error("saas exited", "err", err)
		os.Exit(1)
	}
}

// run 加载 SaaS 配置，初始化 Logger 与 GORM（SQLite），组装 BootstrapDeps 后进入 app.RunSaaS。
// 后续 HTTP、AutoMigrate、服务注册在此函数内向下扩展。
func run(ctx context.Context) error {
	slog.Info("AltShort SaaS process starting")

	cfgPath := filepath.Clean("configs/saas.yaml")
	cfg, err := config.LoadSaaSConfig(cfgPath)
	if err != nil {
		return fmt.Errorf("load saas config: %w", err)
	}

	log := infra.NewLogger("saas", cfg.Logging.Level)
	dsn := cfg.Database.DSN
	if dsn == "" {
		dsn = "file::memory:?cache=shared"
	}
	db, err := infra.OpenSQLite(dsn)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	deps := lifecycle.NewBootstrapDeps(log, db)

	log.Info("SaaS bootstrap complete", "config", cfgPath, "listen", cfg.HTTP.ListenAddr)

	if err := app.RunSaaS(ctx, cfg, deps); err != nil {
		return fmt.Errorf("app.RunSaaS: %w", err)
	}
	return nil
}
