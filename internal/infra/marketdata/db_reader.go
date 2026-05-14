package marketdata

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/JerrrMi/quant/internal/domain"
	"github.com/JerrrMi/quant/internal/infra/db/models"
	"gorm.io/gorm"
)

// DBBarSeriesReader 从 MarketSnapshot 表读取最近 closed bars（SaaS 已采纳快照，非直连交易所）。
type DBBarSeriesReader struct {
	DB *gorm.DB
}

// LatestClosedBar 实现 strategy.StrategyDataProvider：返回该标的最近一次落库快照中的 Bar。
func (r *DBBarSeriesReader) LatestClosedBar(symbol string) (domain.Bar, error) {
	return r.latestClosedBar(context.Background(), symbol)
}

func (r *DBBarSeriesReader) latestClosedBar(ctx context.Context, symbol string) (domain.Bar, error) {
	bars, err := r.RecentClosedBars(ctx, symbol, 1)
	if err != nil {
		return domain.Bar{}, err
	}
	if len(bars) == 0 {
		return domain.Bar{}, fmt.Errorf("marketdata: no bar for symbol %q", symbol)
	}
	return bars[len(bars)-1], nil
}

// RecentClosedBars 按时间正序返回至多 max 条最近快照（基于 captured_at desc 取盘尾再反转为 chronology）。
func (r *DBBarSeriesReader) RecentClosedBars(ctx context.Context, symbol string, max int) ([]domain.Bar, error) {
	if r.DB == nil || max <= 0 {
		return nil, nil
	}
	var rows []models.MarketSnapshot
	if err := r.DB.WithContext(ctx).Where("symbol = ?", symbol).
		Order("captured_at desc, id desc").Limit(max).Find(&rows).Error; err != nil {
		return nil, err
	}
	for i, j := 0, len(rows)-1; i < j; i, j = i+1, j-1 {
		rows[i], rows[j] = rows[j], rows[i]
	}
	out := make([]domain.Bar, 0, len(rows))
	for _, row := range rows {
		var payload SnapshotPayload
		if err := json.Unmarshal([]byte(row.PayloadJSON), &payload); err != nil {
			continue
		}
		if payload.Bar.TimestampUnixMs == 0 && payload.Bar.Close == 0 {
			continue
		}
		out = append(out, payload.Bar)
	}
	return out, nil
}
