package repository

import (
	"context"
	"errors"

	"github.com/JerrrMi/quant/internal/infra/db/models"
	"gorm.io/gorm"
)

// InstanceRepository 编排实例（User+Strategy+Agent 绑定）读写。
type InstanceRepository interface {
	Create(ctx context.Context, row *models.Instance) error
	GetByID(ctx context.Context, id uint) (*models.Instance, error)
	GetByAgentKey(ctx context.Context, agentKey string) (*models.Instance, error)
	ListActive(ctx context.Context) ([]models.Instance, error)
}

// GormInstanceRepository InstanceRepository 的 GORM 骨架实现。
type GormInstanceRepository struct {
	db *gorm.DB
}

func NewGormInstanceRepository(db *gorm.DB) *GormInstanceRepository {
	return &GormInstanceRepository{db: db}
}

func (r *GormInstanceRepository) Create(ctx context.Context, row *models.Instance) error {
	return r.db.WithContext(ctx).Create(row).Error
}

func (r *GormInstanceRepository) GetByID(ctx context.Context, id uint) (*models.Instance, error) {
	var row models.Instance
	if err := r.db.WithContext(ctx).First(&row, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &row, nil
}

func (r *GormInstanceRepository) GetByAgentKey(ctx context.Context, agentKey string) (*models.Instance, error) {
	var row models.Instance
	if err := r.db.WithContext(ctx).Where("agent_key = ?", agentKey).First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &row, nil
}

func (r *GormInstanceRepository) ListActive(ctx context.Context) ([]models.Instance, error) {
	var rows []models.Instance
	if err := r.db.WithContext(ctx).Where("status = ?", "active").Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}
