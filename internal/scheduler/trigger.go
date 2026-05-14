package scheduler

import (
	"context"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
)

// CronTrigger 以 cron 表达式或固定 ticker 触发编排回调（不含业务判断）。
type CronTrigger struct {
	CronExpr string
	Interval time.Duration
	Logger   *slog.Logger
	onTick   func(context.Context) error

	mu         sync.Mutex
	runCancel  context.CancelFunc
	cron       *cron.Cron
	started    bool
	tickerDone chan struct{}
}

// NewCronTrigger 创建触发器；CronExpr 非空时优先使用 cron，否则使用 Interval。
func NewCronTrigger(cronExpr string, interval time.Duration, log *slog.Logger, onTick func(context.Context) error) *CronTrigger {
	return &CronTrigger{CronExpr: cronExpr, Interval: interval, Logger: log, onTick: onTick}
}

// Start 在后台启动；ctx 或 Stop() 均可终止触发。
func (t *CronTrigger) Start(ctx context.Context) error {
	if t == nil || t.onTick == nil {
		return nil
	}
	log := t.Logger
	if log == nil {
		log = slog.Default()
	}

	t.mu.Lock()
	if t.started {
		t.mu.Unlock()
		return nil
	}
	runCtx, cancel := context.WithCancel(ctx)
	t.runCancel = cancel
	t.started = true

	if strings.TrimSpace(t.CronExpr) != "" {
		c := cron.New()
		_, err := c.AddFunc(t.CronExpr, func() {
			if err := t.onTick(runCtx); err != nil {
				log.Warn("scheduled tick failed", "err", err)
			}
		})
		if err != nil {
			t.runCancel = nil
			t.started = false
			cancel()
			t.mu.Unlock()
			return err
		}
		t.cron = c
		c.Start()
		go func() {
			<-runCtx.Done()
			stopCtx := c.Stop()
			<-stopCtx.Done()
		}()
		t.mu.Unlock()
		return nil
	}

	if t.Interval <= 0 {
		t.runCancel = nil
		t.started = false
		cancel()
		t.mu.Unlock()
		return nil
	}

	t.tickerDone = make(chan struct{})
	go func() {
		defer close(t.tickerDone)
		ticker := time.NewTicker(t.Interval)
		defer ticker.Stop()
		for {
			select {
			case <-runCtx.Done():
				return
			case <-ticker.C:
				if err := t.onTick(runCtx); err != nil {
					log.Warn("scheduled tick failed", "err", err)
				}
			}
		}
	}()
	t.mu.Unlock()
	return nil
}

// Stop 立即停止触发（幂等）；与父 ctx 取消叠加仍安全。
func (t *CronTrigger) Stop() {
	if t == nil {
		return
	}
	t.mu.Lock()
	cancel := t.runCancel
	c := t.cron
	doneCh := t.tickerDone
	t.runCancel = nil
	t.cron = nil
	t.started = false
	t.tickerDone = nil
	t.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	if c != nil {
		stopCtx := c.Stop()
		<-stopCtx.Done()
	}
	if doneCh != nil {
		<-doneCh
	}
}
