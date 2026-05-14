package lifecycle

import (
	"context"
	"log/slog"
	"sync/atomic"
	"testing"
)

func TestShutdownCoordinatorOrdering(t *testing.T) {
	var seq atomic.Int32
	coord := &ShutdownCoordinator{Logger: slog.Default(), ProcessName: "test"}
	ctx := context.Background()
	steps := []ShutdownStep{
		{Name: "one", Fn: func(context.Context) error {
			seq.Add(1)
			return nil
		}},
		{Name: "two", Fn: func(context.Context) error {
			seq.Add(10)
			return nil
		}},
	}
	if err := coord.Run(ctx, steps); err != nil {
		t.Fatal(err)
	}
	if seq.Load() != 11 {
		t.Fatalf("got %d want 11", seq.Load())
	}
}

func TestShutdownCoordinatorPropagatesStepError(t *testing.T) {
	coord := &ShutdownCoordinator{Logger: slog.Default(), ProcessName: "test"}
	ctx := context.Background()
	steps := []ShutdownStep{
		{Name: "fail", Fn: func(context.Context) error {
			return context.Canceled
		}},
	}
	err := coord.Run(ctx, steps)
	if err == nil {
		t.Fatal("expected error")
	}
}
