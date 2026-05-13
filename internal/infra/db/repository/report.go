package repository

import (
	"context"

	"github.com/JerrrMi/quant/internal/infra/db/models"
	"gorm.io/gorm"
)

// ReportRepository Agent 回报与控制面侧观察类写入的骨架（不含解析、聚合、告警）。
type ReportRepository interface {
	SaveReport(ctx context.Context, row *models.AgentReportRecord) error
	ListRecentByInstance(ctx context.Context, instanceID uint, limit int) ([]models.AgentReportRecord, error)
}

// GormReportRepository ReportRepository 的 GORM 骨架实现。
type GormReportRepository struct {
	db *gorm.DB
}

func NewGormReportRepository(db *gorm.DB) *GormReportRepository {
	return &GormReportRepository{db: db}
}

func (r *GormReportRepository) SaveReport(ctx context.Context, row *models.AgentReportRecord) error {
	return r.db.WithContext(ctx).Create(row).Error
}

func (r *GormReportRepository) ListRecentByInstance(ctx context.Context, instanceID uint, limit int) ([]models.AgentReportRecord, error) {
	if limit <= 0 {
		limit = 50
	}
	var out []models.AgentReportRecord
	if err := r.db.WithContext(ctx).
		Where("instance_id = ?", instanceID).
		Order("received_at DESC").
		Limit(limit).
		Find(&out).Error; err != nil {
		return nil, err
	}
	return out, nil
}
