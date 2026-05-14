package lppl

import (
	"context"

	"github.com/JerrrMi/quant/internal/infra/db/models"
)

// FixedLatestStore implements ResultStore by returning a fixed latest row (tests / offline backtest).
type FixedLatestStore struct {
	View *ResultView
}

// Save is a no-op; backtest does not persist LPPL runs through this store.
func (FixedLatestStore) Save(context.Context, *models.LPPLScanResult) error {
	return nil
}

// LatestBySymbol returns View when non-nil; symbol is ignored.
func (s FixedLatestStore) LatestBySymbol(context.Context, string) (*ResultView, error) {
	if s.View == nil {
		return nil, nil
	}
	return s.View, nil
}
