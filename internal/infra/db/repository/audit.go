package repository

import (
	"context"
	"errors"

	"github.com/JerrrMi/quant/internal/infra/db/models"
	"gorm.io/gorm"
)

// AuditRepository 控制面审计事件追加。
type AuditRepository interface {
	Append(ctx context.Context, row *models.AuditEvent) error
	ListRecentForResource(ctx context.Context, resourceType, resourceID string, limit int) ([]models.AuditEvent, error)
}

// GormAuditRepository AuditRepository 的 GORM 实现。
type GormAuditRepository struct {
	db *gorm.DB
}

func NewGormAuditRepository(db *gorm.DB) *GormAuditRepository {
	return &GormAuditRepository{db: db}
}

func (r *GormAuditRepository) Append(ctx context.Context, row *models.AuditEvent) error {
	if r.db == nil {
		return errors.New("audit repo: nil db")
	}
	return r.db.WithContext(ctx).Create(row).Error
}

func (r *GormAuditRepository) ListRecentForResource(ctx context.Context, resourceType, resourceID string, limit int) ([]models.AuditEvent, error) {
	if r.db == nil {
		return nil, errors.New("audit repo: nil db")
	}
	if limit <= 0 {
		limit = 50
	}
	var out []models.AuditEvent
	q := r.db.WithContext(ctx).Where("resource_id = ?", resourceID)
	if resourceType != "" {
		q = q.Where("resource_type = ?", resourceType)
	}
	if err := q.Order("occurred_at desc").Limit(limit).Find(&out).Error; err != nil {
		return nil, err
	}
	return out, nil
}
