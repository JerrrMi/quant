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
