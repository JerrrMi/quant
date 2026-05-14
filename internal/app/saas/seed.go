package saas

import (
	"context"
	"errors"
	"fmt"

	"github.com/JerrrMi/quant/internal/infra/db/models"
	"gorm.io/gorm"
)

const consoleUserEmail = "console@system.local"

// EnsureConsoleSeed 创建控制台默认租户用户与内置目录模板（最小做空骨架）。
// 幂等：已存在则跳过对应插入。
func EnsureConsoleSeed(ctx context.Context, db *gorm.DB) error {
	if db == nil {
		return errors.New("ensure seed: nil db")
	}
	var count int64
	if err := db.WithContext(ctx).Model(&models.User{}).Where("email = ?", consoleUserEmail).Count(&count).Error; err != nil {
		return fmt.Errorf("ensure seed: count user: %w", err)
	}
	var user models.User
	if count == 0 {
		user = models.User{
			Email:       consoleUserEmail,
			DisplayName: "Console Default",
		}
		if err := db.WithContext(ctx).Create(&user).Error; err != nil {
			return fmt.Errorf("ensure seed: create user: %w", err)
		}
	} else {
		if err := db.WithContext(ctx).Where("email = ?", consoleUserEmail).First(&user).Error; err != nil {
			return fmt.Errorf("ensure seed: load user: %w", err)
		}
	}

	var tplCount int64
	if err := db.WithContext(ctx).Model(&models.Strategy{}).
		Where("kind = ? AND is_catalog = ?", "minimal_short", true).
		Count(&tplCount).Error; err != nil {
		return fmt.Errorf("ensure seed: count template: %w", err)
	}
	if tplCount > 0 {
		return nil
	}

	cfg := `{"template_defaults":{"signal_lookback":96},"risk_hints":{"note_zh":"仓位与资金限额由实例 ParamsJSON 覆盖；策略 Step() 不读取该字段。"}}`
	row := models.Strategy{
		UserID:        user.ID,
		Name:          "最小做空山寨币（骨架）",
		Kind:          "minimal_short",
		Version:       1,
		ConfigJSON:    cfg,
		Description:   "AltShort 内置最小做空骨架：纯函数 Step()，占位阈值与意图链路；用于连通编排与执行适配。",
		IsCatalog:     true,
		Markets:       "futures",
		AllowLive:     true,
		AllowBacktest: true,
	}
	if err := db.WithContext(ctx).Create(&row).Error; err != nil {
		return fmt.Errorf("ensure seed: create catalog strategy: %w", err)
	}
	return nil
}
