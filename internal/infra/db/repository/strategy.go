package repository

import (
	"context"
	"errors"

	"github.com/JerrrMi/quant/internal/infra/db/models"
	"gorm.io/gorm"
)

// StrategyRepository 策略定义持久化；不负责参数校验与 Step()。
type StrategyRepository interface {
	Create(ctx context.Context, row *models.Strategy) error
	GetByID(ctx context.Context, id uint) (*models.Strategy, error)
	ListByUserID(ctx context.Context, userID uint, limit int) ([]models.Strategy, error)
}

// GormStrategyRepository StrategyRepository 的 GORM 骨架实现。
type GormStrategyRepository struct {
	db *gorm.DB
}

func NewGormStrategyRepository(db *gorm.DB) *GormStrategyRepository {
	return &GormStrategyRepository{db: db}
}

func (r *GormStrategyRepository) Create(ctx context.Context, row *models.Strategy) error {
	return r.db.WithContext(ctx).Create(row).Error
}

func (r *GormStrategyRepository) GetByID(ctx context.Context, id uint) (*models.Strategy, error) {
	var s models.Strategy
	if err := r.db.WithContext(ctx).First(&s, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &s, nil
}

func (r *GormStrategyRepository) ListByUserID(ctx context.Context, userID uint, limit int) ([]models.Strategy, error) {
	if limit <= 0 {
		limit = 100
	}
	var out []models.Strategy
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).Limit(limit).Find(&out).Error; err != nil {
		return nil, err
	}
	return out, nil
}
