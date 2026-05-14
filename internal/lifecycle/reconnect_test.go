package lifecycle

import (
	"testing"
	"time"
)

func TestExpBackoffCapsAndReset(t *testing.T) {
	p := ReconnectPolicy{
		InitialBackoff: time.Second,
		MaxBackoff:     3 * time.Second,
		Multiplier:     2,
		MaxSessions:    0,
		JitterRatio:    0,
	}
	b := NewExpBackoff(p)

	first := b.NextSleep()
	if first != time.Second {
		t.Fatalf("first sleep got %v want 1s", first)
	}
	secondBase := b.Current()
	if secondBase != 2*time.Second {
		t.Fatalf("internal delay got %v want 2s", secondBase)
	}
	b.Reset()
	if cur := b.Current(); cur != time.Second {
		t.Fatalf("after reset got %v want 1s", cur)
	}
}
