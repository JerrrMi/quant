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
	"github.com/JerrrMi/quant/internal/lifecycle"
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
	cache        any
	httpSrv      *http.Server
	hub          *ws.AgentHub
	orchestrator *scheduler.StepOrchestrator
	trigger      *scheduler.CronTrigger

	schedCancel context.CancelFunc
	dataCancel  context.CancelFunc
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
		cache:        deps.Cache,
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

func (r *Runner) flushShutdownSnapshot(ctx context.Context) error {
	if r.db == nil {
		r.log.Warn("shutdown snapshot skipped (no database)")
		return nil
	}
	var runCount int64
	_ = r.db.WithContext(ctx).Model(&models.StrategyRun{}).Where("status = ?", "running").Count(&runCount).Error
	repo := repository.NewGormAuditRepository(r.db)
	payload, _ := json.Marshal(map[string]any{
		"running_strategy_runs": runCount,
		"phase":                 "shutdown",
	})
	if err := repo.Append(ctx, &models.AuditEvent{
		ActorType:    "system",
		ActorID:      "saas",
		Action:       "saas.shutdown_snapshot",
		ResourceType: "saas",
		ResourceID:   "shutdown",
		PayloadJSON:  string(payload),
		OccurredAt:   time.Now().UTC(),
	}); err != nil {
		return fmt.Errorf("flush shutdown snapshot: %w", err)
	}
	r.log.Info("shutdown snapshot flushed", "running_strategy_runs", runCount)
	return nil
}

func (r *Runner) registerLifecycleStages(schedCtx, dataCtx context.Context, mgr *lifecycle.Manager) {
	_ = mgr.Register(lifecycle.Component{
		Name: "recover_state",
		Start: func(ctx context.Context) error {
			r.recoverState(ctx)
			return nil
		},
	})
	_ = mgr.Register(lifecycle.Component{
		Name: "http_websocket",
		Start: func(ctx context.Context) error {
			go func() {
				if err := r.httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
					r.log.Error("http server error", "err", err)
				}
			}()
			return nil
		},
	})
	_ = mgr.Register(lifecycle.Component{
		Name: "scheduler_trigger",
		Start: func(ctx context.Context) error {
			if r.cfg.Scheduler.Enable && r.trigger != nil {
				if err := r.trigger.Start(schedCtx); err != nil {
					return fmt.Errorf("scheduler trigger: %w", err)
				}
			}
			if !r.cfg.Scheduler.Enable {
				r.log.Info("scheduler disabled by config")
			}
			return nil
		},
	})
	_ = mgr.Register(lifecycle.Component{
		Name: "data_pipeline_loop",
		Start: func(ctx context.Context) error {
			go r.runDataLoop(dataCtx)
			return nil
		},
	})
}

func (r *Runner) runGracefulShutdown(shutdownCtx context.Context, schedCancel, dataCancel context.CancelFunc) error {
	coord := &lifecycle.ShutdownCoordinator{Logger: r.log, ProcessName: "saas"}
	steps := []lifecycle.ShutdownStep{
		{
			Name: "stop_accepting_new_work",
			Fn: func(ctx context.Context) error {
				if schedCancel != nil {
					schedCancel()
				}
				if dataCancel != nil {
					dataCancel()
				}
				if r.trigger != nil {
					r.trigger.Stop()
				}
				r.log.Info("shutdown step: scheduler and data pipeline stops requested")
				return nil
			},
		},
		{
			Name: "wait_scheduler_tick_idle",
			Fn: func(ctx context.Context) error {
				if r.orchestrator == nil {
					return nil
				}
				waitCtx, wcancel := context.WithTimeout(ctx, 15*time.Second)
				defer wcancel()
				if err := r.orchestrator.WaitIdle(waitCtx); err != nil {
					r.log.Warn("shutdown: scheduler idle wait ended", "err", err)
				}
				return nil
			},
		},
		{
			Name: "flush_shutdown_snapshot",
			Fn:   r.flushShutdownSnapshot,
		},
		{
			Name: "shutdown_http_websocket",
			Fn: func(ctx context.Context) error {
				if err := r.httpSrv.Shutdown(ctx); err != nil {
					return fmt.Errorf("http shutdown: %w", err)
				}
				return nil
			},
		},
		{
			Name: "close_cache_clients",
			Fn: func(ctx context.Context) error {
				_ = ctx
				return lifecycle.CloseIfCloser(r.cache)
			},
		},
		{
			Name: "close_database",
			Fn: func(ctx context.Context) error {
				_ = ctx
				return lifecycle.CloseGormSQL(r.db)
			},
		},
	}
	return coord.Run(shutdownCtx, steps)
}

// Run 启动子系统并等待 ctx。
func (r *Runner) Run(ctx context.Context) error {
	schedCtx, schedCancel := context.WithCancel(ctx)
	defer schedCancel()
	dataCtx, dataCancel := context.WithCancel(ctx)
	defer dataCancel()

	r.schedCancel = schedCancel
	r.dataCancel = dataCancel
	defer func() {
		r.schedCancel = nil
		r.dataCancel = nil
	}()

	mgr := lifecycle.NewManager(r.log)
	r.registerLifecycleStages(schedCtx, dataCtx, mgr)

	if err := mgr.Start(ctx); err != nil {
		stopCtx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		defer cancel()
		_ = mgr.Stop(stopCtx)
		return err
	}

	r.log.Info("saas listening", "addr", r.cfg.WebSocket.ListenAddr, "ws", r.cfg.WebSocket.AgentPath)

	<-ctx.Done()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), lifecycle.DefaultShutdownTimeout)
	defer shutdownCancel()

	errShutdown := r.runGracefulShutdown(shutdownCtx, schedCancel, dataCancel)

	stopMgrCtx, stopMgrCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer stopMgrCancel()
	_ = mgr.Stop(stopMgrCtx)

	return errShutdown
}

// Start 显式启动 HTTP/WS、可选调度与数据循环（非阻塞返回，由 ctx 停止子任务）。
func (r *Runner) Start(ctx context.Context) error {
	schedCtx, schedCancel := context.WithCancel(ctx)
	r.schedCancel = schedCancel
	dataCtx, dataCancel := context.WithCancel(ctx)
	r.dataCancel = dataCancel

	mgr := lifecycle.NewManager(r.log)
	r.registerLifecycleStages(schedCtx, dataCtx, mgr)
	return mgr.Start(ctx)
}

// Stop 触发与 Run 相同的关停顺序（适用于测试或嵌入宿主）。
func (r *Runner) Stop() error {
	shutdownCtx, cancel := context.WithTimeout(context.Background(), lifecycle.DefaultShutdownTimeout)
	defer cancel()
	schedCancel := r.schedCancel
	dataCancel := r.dataCancel
	err := r.runGracefulShutdown(shutdownCtx, schedCancel, dataCancel)
	r.schedCancel = nil
	r.dataCancel = nil
	return err
}
