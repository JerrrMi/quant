package repository

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

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

// AuditConsoleListFilter 为控制台统一日志/审计列表的查询条件。
type AuditConsoleListFilter struct {
	Since        *time.Time
	Until        *time.Time
	ActionPrefix string
	ResourceType string
	ResourceID   string
	ActorID      string
	// Level：error | warn | info，按 action 子串启发式过滤（非 slog 级别）。
	Level string
	Limit int
}

// ListConsoleVisible 返回当前用户实例域内可见的审计事件：resource=instance 或属于这些实例的 trade_command。
func (r *GormAuditRepository) ListConsoleVisible(ctx context.Context, instanceIDs []uint, f AuditConsoleListFilter) ([]models.AuditEvent, error) {
	if r.db == nil {
		return nil, errors.New("audit repo: nil db")
	}
	if len(instanceIDs) == 0 {
		return []models.AuditEvent{}, nil
	}
	if f.Limit <= 0 {
		f.Limit = 100
	}
	if f.Limit > 500 {
		f.Limit = 500
	}
	idStrs := make([]string, len(instanceIDs))
	for i, id := range instanceIDs {
		idStrs[i] = strconv.FormatUint(uint64(id), 10)
	}
	cmdSub := r.db.WithContext(ctx).Model(&models.TradeCommandRecord{}).Select("id").Where("instance_id IN ?", instanceIDs)

	q := r.db.WithContext(ctx).Model(&models.AuditEvent{}).
		Where(
			r.db.Where("resource_type = ? AND resource_id IN ?", "instance", idStrs).
				Or("resource_type = ? AND resource_id IN (?)", "trade_command", cmdSub),
		)

	if f.Since != nil {
		q = q.Where("occurred_at >= ?", *f.Since)
	}
	if f.Until != nil {
		q = q.Where("occurred_at <= ?", *f.Until)
	}
	if strings.TrimSpace(f.ActionPrefix) != "" {
		q = q.Where("action LIKE ?", fmtPrefixLike(f.ActionPrefix))
	}
	if strings.TrimSpace(f.ResourceType) != "" {
		q = q.Where("resource_type = ?", strings.TrimSpace(f.ResourceType))
	}
	if strings.TrimSpace(f.ResourceID) != "" {
		q = q.Where("resource_id = ?", strings.TrimSpace(f.ResourceID))
	}
	if strings.TrimSpace(f.ActorID) != "" {
		q = q.Where("actor_id = ?", strings.TrimSpace(f.ActorID))
	}
	switch strings.ToLower(strings.TrimSpace(f.Level)) {
	case "error":
		q = q.Where(
			"(LOWER(action) LIKE ? OR LOWER(action) LIKE ? OR LOWER(action) LIKE ?)",
			"%fail%", "%error%", "%terminate%",
		)
	case "warn":
		q = q.Where(
			"(LOWER(action) LIKE ? OR LOWER(action) LIKE ? OR LOWER(action) LIKE ?)",
			"%warn%", "%pause%", "%drain%",
		)
	case "info":
		q = q.Where(
			"NOT ("+
				"LOWER(action) LIKE ? OR LOWER(action) LIKE ? OR LOWER(action) LIKE ? OR "+
				"LOWER(action) LIKE ? OR LOWER(action) LIKE ? OR LOWER(action) LIKE ?"+
				")",
			"%fail%", "%error%", "%terminate%", "%warn%", "%pause%", "%drain%",
		)
	}

	var out []models.AuditEvent
	if err := q.Order("occurred_at DESC").Limit(f.Limit).Find(&out).Error; err != nil {
		return nil, err
	}
	return out, nil
}

func fmtPrefixLike(prefix string) string {
	p := strings.TrimSpace(prefix)
	if strings.ContainsAny(p, "%_\\") {
		return p
	}
	return p + "%"
}
