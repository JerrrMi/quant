package repository

import (
	"context"
	"errors"

	"github.com/JerrrMi/quant/internal/infra/db/models"
	"gorm.io/gorm"
)

// CommandRepository 交易指令与相关上行记录的持久化骨架（不含发送、重试、风控）。
type CommandRepository interface {
	SaveCommand(ctx context.Context, row *models.TradeCommandRecord) error
	GetCommandByID(ctx context.Context, id string) (*models.TradeCommandRecord, error)
	GetByCorrelationID(ctx context.Context, correlationID string) (*models.TradeCommandRecord, error)
	SaveFill(ctx context.Context, row *models.TradeFillRecord) error
	ListFillsByCommandID(ctx context.Context, commandID string) ([]models.TradeFillRecord, error)
	ListRecentByInstance(ctx context.Context, instanceID uint, limit int) ([]models.TradeCommandRecord, error)
}

// GormCommandRepository CommandRepository 的 GORM 骨架实现。
type GormCommandRepository struct {
	db *gorm.DB
}

func NewGormCommandRepository(db *gorm.DB) *GormCommandRepository {
	return &GormCommandRepository{db: db}
}

func (r *GormCommandRepository) SaveCommand(ctx context.Context, row *models.TradeCommandRecord) error {
	return r.db.WithContext(ctx).Save(row).Error
}

func (r *GormCommandRepository) GetCommandByID(ctx context.Context, id string) (*models.TradeCommandRecord, error) {
	var row models.TradeCommandRecord
	if err := r.db.WithContext(ctx).First(&row, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &row, nil
}

func (r *GormCommandRepository) GetByCorrelationID(ctx context.Context, correlationID string) (*models.TradeCommandRecord, error) {
	var row models.TradeCommandRecord
	if err := r.db.WithContext(ctx).Where("correlation_id = ?", correlationID).First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &row, nil
}

func (r *GormCommandRepository) SaveFill(ctx context.Context, row *models.TradeFillRecord) error {
	return r.db.WithContext(ctx).Save(row).Error
}

func (r *GormCommandRepository) ListFillsByCommandID(ctx context.Context, commandID string) ([]models.TradeFillRecord, error) {
	var out []models.TradeFillRecord
	if err := r.db.WithContext(ctx).Where("command_id = ?", commandID).Find(&out).Error; err != nil {
		return nil, err
	}
	return out, nil
}

func (r *GormCommandRepository) ListRecentByInstance(ctx context.Context, instanceID uint, limit int) ([]models.TradeCommandRecord, error) {
	if limit <= 0 {
		limit = 20
	}
	var out []models.TradeCommandRecord
	if err := r.db.WithContext(ctx).
		Where("instance_id = ?", instanceID).
		Order("created_at desc").
		Limit(limit).
		Find(&out).Error; err != nil {
		return nil, err
	}
	return out, nil
}
