// Package lifecycle organizes process startup/shutdown; it must not contain trading or strategy logic.
package lifecycle

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"
)

// Phase reports coarse manager progress for observability.
type Phase string

const (
	PhaseIdle     Phase = "idle"
	PhaseStarting Phase = "starting"
	PhaseRunning  Phase = "running"
	PhaseStopping Phase = "stopping"
	PhaseStopped  Phase = "stopped"
)

// Component is a registered unit with optional Start/Stop hooks.
// Start and Stop must not perform domain-specific trading decisions—only wiring and I/O teardown.
type Component struct {
	Name string
	// Start is invoked in registration order. Long-running work should spawn its own goroutine
	// and return quickly so later components can start.
	Start func(ctx context.Context) error
	// Stop is invoked in reverse registration order during graceful shutdown.
	Stop func(ctx context.Context) error
}

// ComponentHealth is a single component's last known lifecycle signal.
type ComponentHealth struct {
	Name           string
	Started        bool
	StartErr       string
	Stopped        bool
	StopErr        string
	LastTransition string
}

// ManagerSnapshot is an atomic-ish view of registration and health for logs/metrics.
type ManagerSnapshot struct {
	Phase      Phase
	Components []ComponentHealth
}

// Manager wires ordered startup/shutdown and carries optional shared dependencies for injection.
type Manager struct {
	log *slog.Logger

	mu sync.RWMutex
	// deps is an opaque bag processes may use to pass handles between components (logger, DB, etc.).
	deps any

	phase atomic.Value // Phase

	components []Component
	health     []ComponentHealth
}

// NewManager constructs an empty manager. Logger may be nil (falls back to slog.Default).
func NewManager(log *slog.Logger) *Manager {
	m := &Manager{log: log}
	m.phase.Store(PhaseIdle)
	return m
}

// SetDeps replaces the dependency bag seen by observability helpers; safe before Start.
func (m *Manager) SetDeps(deps any) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.deps = deps
}

// Deps returns the last SetDeps value (may be nil).
func (m *Manager) Deps() any {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.deps
}

// Register appends a component. Call before Start. Name must be non-empty.
func (m *Manager) Register(c Component) error {
	if m == nil {
		return fmt.Errorf("lifecycle: nil manager")
	}
	if c.Name == "" {
		return fmt.Errorf("lifecycle: component name is required")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.currentPhaseLocked() != PhaseIdle {
		return fmt.Errorf("lifecycle: register %q after start", c.Name)
	}
	m.components = append(m.components, c)
	m.health = append(m.health, ComponentHealth{Name: c.Name})
	return nil
}

func (m *Manager) currentPhaseLocked() Phase {
	v, _ := m.phase.Load().(Phase)
	return v
}

func (m *Manager) setPhase(p Phase) {
	m.phase.Store(p)
}

func (m *Manager) logf() *slog.Logger {
	if m != nil && m.log != nil {
		return m.log
	}
	return slog.Default()
}

// Start runs each registered Component.Start in order. On failure, already-started components
// receive Stop in reverse order (best-effort).
func (m *Manager) Start(ctx context.Context) error {
	if m == nil {
		return fmt.Errorf("lifecycle: nil manager")
	}
	m.mu.Lock()
	if m.currentPhaseLocked() != PhaseIdle {
		ph := m.currentPhaseLocked()
		m.mu.Unlock()
		return fmt.Errorf("lifecycle: start called from phase %s", ph)
	}
	comps := append([]Component(nil), m.components...)
	m.mu.Unlock()

	m.setPhase(PhaseStarting)
	log := m.logf()

	for i := range comps {
		c := comps[i]
		if c.Start == nil {
			m.markHealth(i, func(h *ComponentHealth) {
				h.LastTransition = "start_skipped"
			})
			continue
		}
		log.Info("lifecycle component starting", "component", c.Name)
		err := c.Start(ctx)
		if err != nil {
			m.markHealth(i, func(h *ComponentHealth) {
				h.StartErr = err.Error()
				h.LastTransition = "start_failed"
			})
			log.Error("lifecycle component start failed", "component", c.Name, "err", err)
			_ = m.stopPartial(ctx, i-1, comps)
			m.setPhase(PhaseStopped)
			return fmt.Errorf("lifecycle: start %s: %w", c.Name, err)
		}
		m.markHealth(i, func(h *ComponentHealth) {
			h.Started = true
			h.LastTransition = "started"
		})
		log.Info("lifecycle component started", "component", c.Name)
	}

	m.setPhase(PhaseRunning)
	return nil
}

func (m *Manager) markHealth(idx int, fn func(*ComponentHealth)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if idx < 0 || idx >= len(m.health) {
		return
	}
	fn(&m.health[idx])
}

// Stop runs Component.Stop in reverse registration order.
func (m *Manager) Stop(ctx context.Context) error {
	if m == nil {
		return fmt.Errorf("lifecycle: nil manager")
	}
	m.mu.Lock()
	comps := append([]Component(nil), m.components...)
	m.mu.Unlock()

	m.setPhase(PhaseStopping)
	log := m.logf()

	var firstErr error
	for i := len(comps) - 1; i >= 0; i-- {
		c := comps[i]
		if c.Stop == nil {
			m.markHealth(i, func(h *ComponentHealth) {
				h.LastTransition = "stop_skipped"
			})
			continue
		}
		log.Info("lifecycle component stopping", "component", c.Name)
		err := c.Stop(ctx)
		if err != nil {
			log.Error("lifecycle component stop failed", "component", c.Name, "err", err)
			m.markHealth(i, func(h *ComponentHealth) {
				h.StopErr = err.Error()
				h.LastTransition = "stop_failed"
			})
			if firstErr == nil {
				firstErr = fmt.Errorf("lifecycle: stop %s: %w", c.Name, err)
			}
			continue
		}
		m.markHealth(i, func(h *ComponentHealth) {
			h.Stopped = true
			h.LastTransition = "stopped"
		})
		log.Info("lifecycle component stopped", "component", c.Name)
	}

	m.setPhase(PhaseStopped)
	return firstErr
}

func (m *Manager) stopPartial(ctx context.Context, lastStartedIdx int, comps []Component) error {
	log := m.logf()
	var firstErr error
	for i := lastStartedIdx; i >= 0; i-- {
		c := comps[i]
		if c.Stop == nil {
			continue
		}
		log.Warn("lifecycle rollback stop", "component", c.Name)
		if err := c.Stop(ctx); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

// Snapshot returns registration health and phase for observability.
func (m *Manager) Snapshot() ManagerSnapshot {
	m.mu.RLock()
	defer m.mu.RUnlock()
	p, _ := m.phase.Load().(Phase)
	h := append([]ComponentHealth(nil), m.health...)
	return ManagerSnapshot{Phase: p, Components: h}
}

// RunOpts configures blocking Run helpers.
type RunOpts struct {
	// ShutdownTimeout bounds Stop after ctx is cancelled. Zero defaults to DefaultShutdownTimeout.
	ShutdownTimeout time.Duration
}

// Run executes Start, waits until ctx is cancelled, then Stop with a bounded timeout.
func (m *Manager) Run(ctx context.Context, opts RunOpts) error {
	if err := m.Start(ctx); err != nil {
		return err
	}
	<-ctx.Done()

	timeout := opts.ShutdownTimeout
	if timeout <= 0 {
		timeout = DefaultShutdownTimeout
	}
	stopCtx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return m.Stop(stopCtx)
}
