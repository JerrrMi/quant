package lppl

import (
	"context"
)

// ScanTaskHandle 引用一次扫描任务（外部批处理或内嵌作业）。
type ScanTaskHandle struct {
	JobID string
}

// Scanner 提交/触发 LPPL（或同类）扫描；SaaS 仅编排，不在此包实现具体拟合。
type Scanner interface {
	// ScanSymbol 触发对 symbol 的扫描；可异步，返回 job id 供 Poll/查库。
	ScanSymbol(ctx context.Context, symbol string) (ScanTaskHandle, error)
}

// NoopScanner 占位：不触发外部系统，供单测与离线管线。
type NoopScanner struct{}

func (NoopScanner) ScanSymbol(ctx context.Context, symbol string) (ScanTaskHandle, error) {
	_ = ctx
	_ = symbol
	return ScanTaskHandle{}, nil
}
