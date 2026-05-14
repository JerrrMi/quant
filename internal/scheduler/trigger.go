package scheduler

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/robfig/cron/v3"
)

// CronTrigger 以 cron 表达式或固定 ticker 触发编排回调（不含业务判断）。
type CronTrigger struct {
	CronExpr string
	Interval time.Duration
	Logger   *slog.Logger
	onTick   func(context.Context) error
	cron     *cron.Cron
}

// NewCronTrigger 创建触发器；CronExpr 非空时优先使用 cron，否则使用 Interval。
func NewCronTrigger(cronExpr string, interval time.Duration, log *slog.Logger, onTick func(context.Context) error) *CronTrigger {
	return &CronTrigger{CronExpr: cronExpr, Interval: interval, Logger: log, onTick: onTick}
}

// Start 在后台启动；ctx 取消时停止。
func (t *CronTrigger) Start(ctx context.Context) error {
	if t == nil || t.onTick == nil {
		return nil
	}
	log := t.Logger
	if log == nil {
		log = slog.Default()
	}
	if strings.TrimSpace(t.CronExpr) != "" {
		c := cron.New()
		_, err := c.AddFunc(t.CronExpr, func() {
			if err := t.onTick(ctx); err != nil {
				log.Warn("scheduled tick failed", "err", err)
			}
		})
		if err != nil {
			return err
		}
		t.cron = c
		c.Start()
		go func() {
			<-ctx.Done()
			stopCtx := c.Stop()
			<-stopCtx.Done()
		}()
		return nil
	}
	if t.Interval <= 0 {
		return nil
	}
	go func() {
		ticker := time.NewTicker(t.Interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := t.onTick(ctx); err != nil {
					log.Warn("scheduled tick failed", "err", err)
				}
			}
		}
	}()
	return nil
}

// Stop 为占位：停止由启动时传入的 ctx 取消驱动。
func (t *CronTrigger) Stop() {}
