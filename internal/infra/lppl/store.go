package lppl

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/JerrrMi/quant/internal/infra/db/models"
	"gorm.io/gorm"
)

// ResultView 为扫描行读取后的结构化视图（承接 models.LPPLScanResult，不含 GORM）。
type ResultView struct {
	Symbol      string
	WindowStart time.Time
	WindowEnd   time.Time
	ComputedAt  time.Time
	JobID       string
	ParamsJSON  string
	ResultJSON  string
}

// ResultStore 控制面对 LPPL 结果的读/写端口。
type ResultStore interface {
	Save(ctx context.Context, row *models.LPPLScanResult) error
	LatestBySymbol(ctx context.Context, symbol string) (*ResultView, error)
}

// GormResultStore ResultStore 的 GORM 实现。
type GormResultStore struct {
	DB *gorm.DB
}

func (s *GormResultStore) Save(ctx context.Context, row *models.LPPLScanResult) error {
	if s.DB == nil {
		return errors.New("lppl: nil db")
	}
	return s.DB.WithContext(ctx).Save(row).Error
}

func (s *GormResultStore) LatestBySymbol(ctx context.Context, symbol string) (*ResultView, error) {
	if s.DB == nil {
		return nil, errors.New("lppl: nil db")
	}
	var row models.LPPLScanResult
	if err := s.DB.WithContext(ctx).Where("symbol = ?", symbol).
		Order("computed_at desc, id desc").First(&row).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &ResultView{
		Symbol:      row.Symbol,
		WindowStart: row.WindowStart,
		WindowEnd:   row.WindowEnd,
		ComputedAt:  row.ComputedAt,
		JobID:       row.JobID,
		ParamsJSON:  row.ParamsJSON,
		ResultJSON:  row.ResultJSON,
	}, nil
}

// ResultScalars 为写入特征切片的规范 JSON 形状（外部作业与适配器之间的契约）。
type ResultScalars struct {
	// BubbleMetric01 为示例无量纲泡沫强度占位 [0,1]；真实 LPPL 参量可扩展字段。
	BubbleMetric01 float64 `json:"bubble_metric_01,omitempty"`
	LinearSlope    float64 `json:"linear_slope,omitempty"`
	Notes          string  `json:"notes,omitempty"`
}

// ParseScalars 尝试从 ResultView.ResultJSON 解码标量；失败时返回零值视图但不报错（特征缺失策略自行处理）。
func ParseScalars(v *ResultView) ResultScalars {
	if v == nil || v.ResultJSON == "" {
		return ResultScalars{}
	}
	var sc ResultScalars
	_ = json.Unmarshal([]byte(v.ResultJSON), &sc)
	return sc
}
