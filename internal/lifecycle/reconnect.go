package lifecycle

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand"
	"sync"
	"time"
)

// ReconnectPolicy configures exponential backoff between logical sessions (e.g. WebSocket connects).
type ReconnectPolicy struct {
	InitialBackoff time.Duration
	MaxBackoff     time.Duration
	// Multiplier applied after each failed session; values < 2 behave like 2.
	Multiplier float64
	// MaxSessions limits reconnect attempts; 0 means unlimited.
	MaxSessions int
	// JitterRatio in [0,1] scales random jitter added on top of the backoff (same semantics as Agent YAML).
	JitterRatio float64
}

// ExpBackoff holds mutable backoff state between disconnect and the next dial attempt.
type ExpBackoff struct {
	policy ReconnectPolicy
	next   time.Duration
	mu     sync.Mutex
}

// NewExpBackoff clones policy defaults (multiplier floor 2) and seeds current delay to InitialBackoff.
func NewExpBackoff(p ReconnectPolicy) *ExpBackoff {
	mul := p.Multiplier
	if mul < 2 {
		mul = 2
	}
	pcopy := p
	pcopy.Multiplier = mul
	init := pcopy.InitialBackoff
	if init <= 0 {
		init = 2 * time.Second
	}
	return &ExpBackoff{policy: pcopy, next: init}
}

// Reset restores delay to the configured initial backoff (call after a healthy session is authenticated).
func (b *ExpBackoff) Reset() {
	if b == nil {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	init := b.policy.InitialBackoff
	if init <= 0 {
		init = 2 * time.Second
	}
	b.next = init
}

// Current returns the delay that would be used on the next NextSleep without consuming it.
func (b *ExpBackoff) Current() time.Duration {
	if b == nil {
		return 0
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.next
}

// NextSleep returns the sleep duration for this disconnect cycle and advances internal backoff.
func (b *ExpBackoff) NextSleep() time.Duration {
	if b == nil {
		return time.Second
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	sleep := applyJitter(b.next, b.policy.JitterRatio)
	maxCap := b.policy.MaxBackoff
	if maxCap > 0 && sleep > maxCap {
		sleep = maxCap
	}
	// Advance for the *next* failure.
	nd := time.Duration(float64(b.next) * b.policy.Multiplier)
	if maxCap > 0 && nd > maxCap {
		nd = maxCap
	}
	b.next = nd
	return sleep
}

func applyJitter(base time.Duration, jitterRatio float64) time.Duration {
	if jitterRatio <= 0 || base <= 0 {
		return base
	}
	// Same shape as Agent jitterMul: multiplier in [1, 1+2*jitterRatio]
	delta := jitterRatio * 2 * rand.Float64()
	m := 1 + delta
	return time.Duration(float64(base) * m)
}

// DisconnectRecorder captures disconnect metadata without interpreting venue-specific errors.
type DisconnectRecorder struct {
	mu sync.RWMutex

	LastReason       error
	LastDisconnectAt time.Time
	Attempts         int
}

// Record stores the latest disconnect reason (best-effort wall clock via caller-supplied instant).
func (r *DisconnectRecorder) Record(reason error, at time.Time) {
	if r == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.LastReason = reason
	r.LastDisconnectAt = UTCOrZero(at)
	r.Attempts++
}

// Snapshot returns a copy for logs/metrics.
func (r *DisconnectRecorder) Snapshot() (reason error, at time.Time, attempts int) {
	if r == nil {
		return nil, time.Time{}, 0
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.LastReason, r.LastDisconnectAt, r.Attempts
}

func UTCOrZero(t time.Time) time.Time {
	if t.IsZero() {
		return time.Time{}
	}
	return t.UTC()
}

// ReconnectHooks are observation / convergence hooks; implementers must keep them cheap and idempotent-friendly.
type ReconnectHooks struct {
	OnDisconnected func(reason error, disconnectedAt time.Time, cumulativeAttempts int)

	// BeforeReconnect fires immediately before sleeping until the next dial attempt.
	BeforeReconnect func(nextBackoff time.Duration, sessionIndex int)

	// AfterAuthSuccess runs once per session after control-plane auth succeeds and before heartbeats/commands.
	// Use this for state synchronization entry points; combine with executor/idempotency guards to avoid duplicate execution.
	AfterAuthSuccess func(ctx context.Context) error
}

// RunReconnectLoop repeatedly runs session until ctx ends. session must return when the transport dies or ctx ends.
// Call ExpBackoff.Reset from AfterAuthSuccess (typically wrapped inside session) so a healthy auth restores normal pacing.
func RunReconnectLoop(ctx context.Context, log *slog.Logger, hooks ReconnectHooks, backoff *ExpBackoff, session func(context.Context) error) error {
	if session == nil {
		return fmt.Errorf("lifecycle: nil reconnect session")
	}
	if backoff == nil {
		return fmt.Errorf("lifecycle: nil backoff")
	}
	policy := backoff.policy
	rec := &DisconnectRecorder{}
	log = lookupLogger(log)

	sessionsStarted := 0
	for {
		if policy.MaxSessions > 0 && sessionsStarted >= policy.MaxSessions {
			return fmt.Errorf("lifecycle: reconnect exhausted (max_sessions=%d)", policy.MaxSessions)
		}

		sessCtx, cancel := context.WithCancel(ctx)
		sessionsStarted++

		err := session(sessCtx)
		cancel()

		if ctx.Err() != nil {
			return ctx.Err()
		}

		at := time.Now().UTC()
		rec.Record(err, at)
		if hooks.OnDisconnected != nil {
			_, _, attempts := rec.Snapshot()
			hooks.OnDisconnected(err, at, attempts)
		} else {
			log.Warn("transport session ended", "err", err, "sessions", sessionsStarted)
		}

		sleep := backoff.NextSleep()
		if hooks.BeforeReconnect != nil {
			hooks.BeforeReconnect(sleep, sessionsStarted)
		} else {
			log.Info("reconnect backoff", "sleep", sleep.String(), "session", sessionsStarted)
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(sleep):
		}
	}
}

func lookupLogger(log *slog.Logger) *slog.Logger {
	if log != nil {
		return log
	}
	return slog.Default()
}

// AgentReconnectPolicy maps Agent YAML reconnect knobs into ReconnectPolicy.
func AgentReconnectPolicy(initialSecs, maxSecs int, jitter float64, maxAttempts int) ReconnectPolicy {
	p := ReconnectPolicy{
		InitialBackoff: durationFromSecs(initialSecs, 2*time.Second),
		MaxBackoff:     durationFromSecs(maxSecs, 0),
		Multiplier:     2,
		MaxSessions:    maxAttempts,
		JitterRatio:    jitter,
	}
	return p
}

func durationFromSecs(secs int, fallback time.Duration) time.Duration {
	if secs <= 0 {
		return fallback
	}
	return time.Duration(secs) * time.Second
}
