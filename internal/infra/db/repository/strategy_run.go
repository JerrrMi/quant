package repository

import (
	"context"
	"errors"
	"time"

	"github.com/JerrrMi/quant/internal/infra/db/models"
	"gorm.io/gorm"
)

// StrategyRunRepository 策略运行周期与 Step 序号游标。
type StrategyRunRepository interface {
	Create(ctx context.Context, row *models.StrategyRun) error
	GetRunningByInstance(ctx context.Context, instanceID uint) (*models.StrategyRun, error)
	EnsureRunningRun(ctx context.Context, instanceID, strategyID uint) (*models.StrategyRun, error)
	UpdateLastStepSequence(ctx context.Context, runID uint, seq int64) error
	StopRunningRunsForInstance(ctx context.Context, instanceID uint) error
}

// GormStrategyRunRepository StrategyRunRepository 的 GORM 实现。
type GormStrategyRunRepository struct {
	db *gorm.DB
}

func NewGormStrategyRunRepository(db *gorm.DB) *GormStrategyRunRepository {
	return &GormStrategyRunRepository{db: db}
}

func (r *GormStrategyRunRepository) Create(ctx context.Context, row *models.StrategyRun) error {
	if r.db == nil {
		return errors.New("strategy run repo: nil db")
	}
	return r.db.WithContext(ctx).Create(row).Error
}

func (r *GormStrategyRunRepository) GetRunningByInstance(ctx context.Context, instanceID uint) (*models.StrategyRun, error) {
	if r.db == nil {
		return nil, errors.New("strategy run repo: nil db")
	}
	var row models.StrategyRun
	if err := r.db.WithContext(ctx).
		Where("instance_id = ? AND status = ?", instanceID, "running").
		Order("id desc").First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &row, nil
}

func (r *GormStrategyRunRepository) EnsureRunningRun(ctx context.Context, instanceID, strategyID uint) (*models.StrategyRun, error) {
	row, err := r.GetRunningByInstance(ctx, instanceID)
	if err != nil {
		return nil, err
	}
	if row != nil {
		return row, nil
	}
	now := time.Now().UTC()
	row = &models.StrategyRun{
		InstanceID:       instanceID,
		StrategyID:       strategyID,
		Status:           "running",
		StartedAt:        &now,
		LastStepSequence: 0,
	}
	if err := r.Create(ctx, row); err != nil {
		return nil, err
	}
	return row, nil
}

func (r *GormStrategyRunRepository) UpdateLastStepSequence(ctx context.Context, runID uint, seq int64) error {
	if r.db == nil {
		return errors.New("strategy run repo: nil db")
	}
	return r.db.WithContext(ctx).Model(&models.StrategyRun{}).
		Where("id = ?", runID).Update("last_step_sequence", seq).Error
}

func (r *GormStrategyRunRepository) StopRunningRunsForInstance(ctx context.Context, instanceID uint) error {
	if r.db == nil {
		return errors.New("strategy run repo: nil db")
	}
	now := time.Now().UTC()
	return r.db.WithContext(ctx).Model(&models.StrategyRun{}).
		Where("instance_id = ? AND status = ?", instanceID, "running").
		Updates(map[string]any{
			"status":    "stopped",
			"ended_at":  &now,
			"updated_at": now,
		}).Error
}
