package models

// AgentDedupKey records idempotency keys applied on the Agent (local DB only; not SaaS authority).
// Survives process restarts so SaaS command replays do not re-submit to the venue.
type AgentDedupKey struct {
	CorrelationKey  string `gorm:"primaryKey;size:191"`
	CommandID       string `gorm:"size:64"`
	ClientOrderID   string `gorm:"size:64"`
	ExchangeOrderID string `gorm:"size:32"`
	LastStatus      string `gorm:"size:32"`
	UpdatedAtUnixMs int64
}
