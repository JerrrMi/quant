package lifecycle

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"time"

	"gorm.io/gorm"
)

// DefaultShutdownTimeout caps graceful shutdown steps after SIGTERM when callers do not override.
const DefaultShutdownTimeout = 30 * time.Second

// ShutdownStep is an ordered unit executed by ShutdownCoordinator.
type ShutdownStep struct {
	Name string
	Fn   func(ctx context.Context) error
}

// ShutdownCoordinator runs shutdown steps with structured logging. It performs no domain logic inside Fn callbacks.
type ShutdownCoordinator struct {
	Logger      *slog.Logger
	ProcessName string
}

// Run executes steps in order until Fn returns error or ctx expires.
func (c *ShutdownCoordinator) Run(ctx context.Context, steps []ShutdownStep) error {
	if c == nil {
		return fmt.Errorf("lifecycle: nil shutdown coordinator")
	}
	log := c.Logger
	if log == nil {
		log = slog.Default()
	}
	proc := c.ProcessName
	if proc == "" {
		proc = "process"
	}

	log.Info("graceful shutdown starting", "process", proc, "steps", len(steps))

	var firstErr error
	for _, step := range steps {
		if step.Fn == nil {
			continue
		}
		name := step.Name
		if name == "" {
			name = "anonymous"
		}
		log.Info("shutdown step begin", "process", proc, "step", name)
		err := step.Fn(ctx)
		if err != nil {
			log.Error("shutdown step failed", "process", proc, "step", name, "err", err)
			if firstErr == nil {
				firstErr = fmt.Errorf("shutdown step %s: %w", name, err)
			}
			continue
		}
		log.Info("shutdown step complete", "process", proc, "step", name)
	}

	if firstErr != nil {
		log.Warn("graceful shutdown finished with errors", "process", proc, "err", firstErr)
		return firstErr
	}
	log.Info("graceful shutdown complete", "process", proc)
	return nil
}

// CloseGormSQL closes the underlying *sql.DB from a GORM handle when present.
func CloseGormSQL(db *gorm.DB) error {
	if db == nil {
		return nil
	}
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// CloseIfCloser invokes Close on values implementing io.Closer (e.g. cache clients).
func CloseIfCloser(v any) error {
	if v == nil {
		return nil
	}
	cl, ok := v.(io.Closer)
	if !ok {
		return nil
	}
	return cl.Close()
}
