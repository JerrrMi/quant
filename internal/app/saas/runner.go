// Package saas 实现控制面进程的应用生命周期：WS、调度、数据刷新占位与状态恢复钩子。
package saas

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/JerrrMi/quant/internal/config"
	"github.com/JerrrMi/quant/internal/domain/strategy"
	"github.com/JerrrMi/quant/internal/infra/db/models"
	"github.com/JerrrMi/quant/internal/infra/db/repository"
	"github.com/JerrrMi/quant/internal/infra/lppl"
	"github.com/JerrrMi/quant/internal/infra/marketdata"
	"github.com/JerrrMi/quant/internal/infra/ws"
	"github.com/JerrrMi/quant/internal/scheduler"
	"gorm.io/gorm"
)

// Deps SaaS 运行所需外部依赖（由 Bootstrap 填充子集）。
type Deps struct {
	Logger *slog.Logger
	DB     *gorm.DB
	Cache  any
}

// Runner 封装 HTTP/WS、调度与后台循环。
type Runner struct {
	cfg          config.SaaSConfig
	log          *slog.Logger
	db           *gorm.DB
	httpSrv      *http.Server
	hub          *ws.AgentHub
	orchestrator *scheduler.StepOrchestrator
	trigger      *scheduler.CronTrigger
}

// NewRunner 组装 Runner（不启动监听）。
func NewRunner(cfg config.SaaSConfig, deps Deps) (*Runner, error) {
	if deps.Logger == nil {
		deps.Logger = slog.Default()
	}
	if deps.DB == nil {
		return nil, fmt.Errorf("saas runner: db is nil")
	}
	hub := ws.NewAgentHub()
	mux := http.NewServeMux()
	mux.Handle(cfg.WebSocket.AgentPath, &ws.SaasAgentServer{Hub: hub, Logger: deps.Logger})

	orch := &scheduler.StepOrchestrator{
		Logger:     deps.Logger,
		Stepper:    strategy.MinimalShortStrategy{},
		Bars:       &marketdata.DBBarSeriesReader{DB: deps.DB},
		LPPL:       &lppl.InputAugmentor{Store: &lppl.GormResultStore{DB: deps.DB}},
		Runs:       repository.NewGormStrategyRunRepository(deps.DB),
		Instances:  repository.NewGormInstanceRepository(deps.DB),
		Commands:   repository.NewGormCommandRepository(deps.DB),
		Audit:      repository.NewGormAuditRepository(deps.DB),
		Dispatcher: &HubDispatcher{Hub: hub},
		Model:      cfg.Model,
		DefaultSym: cfg.Scheduler.DefaultSymbol,
		Deadline:   2 * time.Minute,
	}

	tick := time.Duration(cfg.Scheduler.TickIntervalSecs) * time.Second
	trigger := scheduler.NewCronTrigger(cfg.Scheduler.CronExpression, tick, deps.Logger, orch.Tick)

	return &Runner{
		cfg:          cfg,
		log:          deps.Logger,
		db:           deps.DB,
		httpSrv:      &http.Server{Addr: cfg.WebSocket.ListenAddr, Handler: mux},
		hub:          hub,
		orchestrator: orch,
		trigger:      trigger,
	}, nil
}

// Run 阻塞直至 ctx 取消，然后优雅关停 HTTP。
func Run(ctx context.Context, cfg config.SaaSConfig, deps Deps) error {
	r, err := NewRunner(cfg, deps)
	if err != nil {
		return err
	}
	return r.Run(ctx)
}

// runDataLoop 周期性唤醒数据管线占位（不访问交易所）。
func (r *Runner) runDataLoop(ctx context.Context) {
	secs := r.cfg.DataPipeline.RefreshIntervalSecs
	if secs <= 0 {
		secs = r.cfg.Scheduler.TickIntervalSecs
	}
	if secs <= 0 {
		return
	}
	t := time.NewTicker(time.Duration(secs) * time.Second)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			r.log.Debug("data pipeline refresh tick (noop stub)")
		}
	}
}

func (r *Runner) recoverState(ctx context.Context) {
	_ = ctx
	if r.db == nil || r.log == nil {
		return
	}
	var runCount int64
	_ = r.db.WithContext(ctx).Model(&models.StrategyRun{}).Where("status = ?", "running").Count(&runCount).Error
	repo := repository.NewGormAuditRepository(r.db)
	payload, _ := json.Marshal(map[string]any{"running_strategy_runs": runCount})
	_ = repo.Append(ctx, &models.AuditEvent{
		ActorType:    "system",
		ActorID:      "saas",
		Action:       "saas.state_recovery",
		ResourceType: "saas",
		ResourceID:   "bootstrap",
		PayloadJSON:  string(payload),
		OccurredAt:   time.Now().UTC(),
	})
	r.log.Info("saas state recovery done", "running_strategy_runs", runCount)
}

// Run 启动子系统并等待 ctx。
func (r *Runner) Run(ctx context.Context) error {
	r.recoverState(ctx)

	go func() {
		if err := r.httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			r.log.Error("http server error", "err", err)
		}
	}()

	if r.cfg.Scheduler.Enable && r.trigger != nil {
		if err := r.trigger.Start(ctx); err != nil {
			return fmt.Errorf("scheduler trigger: %w", err)
		}
	}

	go r.runDataLoop(ctx)

	if !r.cfg.Scheduler.Enable {
		r.log.Info("scheduler disabled by config")
	}

	r.log.Info("saas listening", "addr", r.cfg.WebSocket.ListenAddr, "ws", r.cfg.WebSocket.AgentPath)

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	r.trigger.Stop()
	if err := r.httpSrv.Shutdown(shutdownCtx); err != nil {
		return err
	}
	return nil
}

// Start 显式启动 HTTP/WS、可选调度与数据循环（非阻塞返回，由 ctx 停止子任务）。
func (r *Runner) Start(ctx context.Context) error {
	r.recoverState(ctx)
	go func() {
		if err := r.httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			r.log.Error("http server error", "err", err)
		}
	}()
	if r.cfg.Scheduler.Enable && r.trigger != nil {
		if err := r.trigger.Start(ctx); err != nil {
			return err
		}
	}
	go r.runDataLoop(ctx)
	return nil
}

// Stop 优雅关停 HTTP。
func (r *Runner) Stop() error {
	r.trigger.Stop()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	return r.httpSrv.Shutdown(shutdownCtx)
}
