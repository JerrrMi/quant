package agentstate

import (
	"context"
	"time"

	"github.com/JerrrMi/quant/internal/infra/db/models"
	"gorm.io/gorm"
)

// DedupStore persists command idempotency keys for Agent execution.
type DedupStore struct {
	db *gorm.DB
}

// NewDedupStore wires a GORM connection; nil db yields a no-store (Get always not found).
func NewDedupStore(db *gorm.DB) *DedupStore {
	return &DedupStore{db: db}
}

// Get returns a row by correlation key or gorm.ErrRecordNotFound.
func (s *DedupStore) Get(ctx context.Context, correlationKey string) (*models.AgentDedupKey, error) {
	if s == nil || s.db == nil {
		return nil, gorm.ErrRecordNotFound
	}
	var row models.AgentDedupKey
	tx := s.db.WithContext(ctx).Where("correlation_key = ?", correlationKey).First(&row)
	if tx.Error != nil {
		return nil, tx.Error
	}
	return &row, nil
}

// Upsert writes or replaces the row for correlationKey.
func (s *DedupStore) Upsert(ctx context.Context, row *models.AgentDedupKey) error {
	if s == nil || s.db == nil {
		return gorm.ErrInvalidDB
	}
	if row == nil {
		return gorm.ErrInvalidData
	}
	row.UpdatedAtUnixMs = time.Now().UnixMilli()
	return s.db.WithContext(ctx).Save(row).Error
}
