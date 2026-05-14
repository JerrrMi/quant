package models

import "time"

// BacktestJob 控制台发起的离线回测任务（异步；报告 JSON 与仪表盘契约对齐 internal/backtest）。
type BacktestJob struct {
	ID           uint `gorm:"primaryKey"`
	UserID       uint `gorm:"index;not null"`
	Status       string `gorm:"size:16;index;not null"` // pending|running|finished|failed|cancelled
	TemplateID   uint   `gorm:"index;not null"`
	TemplateName string `gorm:"size:191"`
	InstanceID   *uint  `gorm:"index"`
	Symbol       string `gorm:"size:64;index"`
	MarketKind   string `gorm:"size:16"`
	WindowStart  string `gorm:"size:64"` // RFC3339 或空
	WindowEnd    string `gorm:"size:64"`

	RequestJSON  string `gorm:"type:text;not null"`
	ReportJSON   string `gorm:"type:longtext"`
	LogJSON      string `gorm:"type:text"` // JSON 数组 of string
	ProgressJSON string `gorm:"type:text"` // {"done","total","pct01"}

	ErrorMessage string `gorm:"size:512"`

	CreatedAt  time.Time `gorm:"index"`
	UpdatedAt  time.Time
	StartedAt  *time.Time
	FinishedAt *time.Time
}
