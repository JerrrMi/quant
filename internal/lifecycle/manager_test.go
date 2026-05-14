package lifecycle

import (
	"context"
	"log/slog"
	"sync/atomic"
	"testing"
	"time"
)

func TestManagerStartStopOrder(t *testing.T) {
	ctx := context.Background()
	var seq atomic.Int32
	m := NewManager(slog.Default())

	_ = m.Register(Component{
		Name: "a",
		Start: func(context.Context) error {
			seq.Store(seq.Load()*10 + 1)
			return nil
		},
		Stop: func(context.Context) error {
			seq.Store(seq.Load()*10 + 7)
			return nil
		},
	})
	_ = m.Register(Component{
		Name: "b",
		Start: func(context.Context) error {
			seq.Store(seq.Load()*10 + 2)
			return nil
		},
		Stop: func(context.Context) error {
			seq.Store(seq.Load()*10 + 8)
			return nil
		},
	})

	if err := m.Start(ctx); err != nil {
		t.Fatal(err)
	}
	// Start order → …12
	if got := seq.Load(); got != 12 {
		t.Fatalf("start order got %d want 12", got)
	}

	stopCtx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := m.Stop(stopCtx); err != nil {
		t.Fatal(err)
	}
	// Stop reverse → …,(1287)=stop b then a: from 12 -> *10+8=128 -> *10+7=1287
	if got := seq.Load(); got != 1287 {
		t.Fatalf("stop order got %d want 1287", got)
	}
}

func TestManagerRunHonoursShutdownTimeout(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	doneStop := make(chan struct{})
	m := NewManager(slog.Default())

	_ = m.Register(Component{
		Name: "sleep_stop",
		Start: func(context.Context) error {
			return nil
		},
		Stop: func(stopCtx context.Context) error {
			defer close(doneStop)
			<-stopCtx.Done()
			return stopCtx.Err()
		},
	})

	go func() {
		cancel()
	}()

	opts := RunOpts{ShutdownTimeout: 50 * time.Millisecond}
	err := m.Run(ctx, opts)
	if err == nil {
		t.Fatal("expected stop deadline error")
	}

	select {
	case <-doneStop:
	case <-time.After(time.Second):
		t.Fatal("Stop did not return")
	}
}
