package executor

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/JerrrMi/quant/internal/domain"
	"github.com/JerrrMi/quant/internal/domain/report"
	"github.com/JerrrMi/quant/internal/infra/binance"
)

// Venue abstracts a linear USDT-margined futures API for the Agent executor.
// Implemented by *BinanceVenue for Binance USD-M; no strategy semantics.
type Venue interface {
	SyncTime(ctx context.Context) error
	MarkPrice(ctx context.Context, symbol string) (float64, error)
	LotStep(ctx context.Context, symbol string) (float64, error)
	PlaceMarket(ctx context.Context, in PlaceMarketIn) (*OrderView, error)
	Cancel(ctx context.Context, in CancelIn) (*OrderView, error)
	QueryByClientOrder(ctx context.Context, symbol, clientOID string) (*OrderView, error)
	QueryByVenueOrderID(ctx context.Context, symbol string, orderID int64) (*OrderView, error)
	UserTrades(ctx context.Context, symbol string, orderID int64) ([]report.FillRecord, error)
	AccountAndPositions(ctx context.Context) (*VenueAccountView, error)
	OpenOrders(ctx context.Context, symbol string) ([]OpenOrderView, error)
}

// PlaceMarketIn carries exchange-neutral market order parameters.
type PlaceMarketIn struct {
	Symbol           string
	Side             domain.Side
	QuantityBase     float64
	ReduceOnly       bool
	NewClientOrderID string
}

// CancelIn identifies an order to cancel.
type CancelIn struct {
	Symbol            string
	ClientOrderID     string
	VenueOrderID      int64
	VenueOrderIDValid bool
}

// OrderView normalized venue order snapshot.
type OrderView struct {
	VenueOrderID int64
	ClientOrderID string
	Symbol        string
	Side          domain.Side
	Status        string
	AvgPrice      float64
	ExecutedQty   float64
	UpdateUnixMs  int64
	ReduceOnly    bool
	OrigQuantity  float64
}

// OpenOrderView mirrors a resting order.
type OpenOrderView struct {
	VenueOrderID int64
	Symbol       string
	Side         domain.Side
	Price        float64
	Quantity     float64
	FilledQty    float64
	Status       report.OrderStatus
	ReduceOnly   bool
	UpdateUnixMs int64
}

// VenueAccountView merges wallet + linear positions for DeltaReport.
type VenueAccountView struct {
	EquityUSDT           float64
	WalletBalanceUSDT    float64
	AvailableBalanceUSDT float64
	ExchangeTimeUnixMs   int64
	Positions            []report.PositionSnapshot
}

// BinanceVenue adapts *binance.Client to Venue.
type BinanceVenue struct {
	C *binance.Client
}

func (b *BinanceVenue) SyncTime(ctx context.Context) error {
	if b == nil || b.C == nil {
		return nil
	}
	return b.C.SyncServerTime(ctx)
}

func (b *BinanceVenue) MarkPrice(ctx context.Context, symbol string) (float64, error) {
	return b.C.MarkPrice(ctx, symbol)
}

func (b *BinanceVenue) LotStep(ctx context.Context, symbol string) (float64, error) {
	return b.C.LotStepSize(ctx, symbol)
}

func (b *BinanceVenue) PlaceMarket(ctx context.Context, in PlaceMarketIn) (*OrderView, error) {
	res, err := b.C.PlaceMarket(ctx, binance.PlaceMarketRequest{
		Symbol:           in.Symbol,
		Side:             string(in.Side),
		QuantityBase:     in.QuantityBase,
		ReduceOnly:       in.ReduceOnly,
		NewClientOrderID: in.NewClientOrderID,
	})
	if err != nil {
		return nil, err
	}
	return orderViewFromBinance(res), nil
}

func (b *BinanceVenue) Cancel(ctx context.Context, in CancelIn) (*OrderView, error) {
	var res *binance.OrderResult
	var err error
	switch {
	case in.VenueOrderIDValid:
		res, err = b.C.CancelByOrderID(ctx, in.Symbol, in.VenueOrderID)
	case strings.TrimSpace(in.ClientOrderID) != "":
		res, err = b.C.CancelByClientOrderID(ctx, in.Symbol, in.ClientOrderID)
	default:
		return nil, errors.New("executor: cancel requires venue_order_id or client_order_id")
	}
	if err != nil {
		return nil, err
	}
	return orderViewFromBinance(res), nil
}

func (b *BinanceVenue) QueryByClientOrder(ctx context.Context, symbol, clientOID string) (*OrderView, error) {
	res, err := b.C.QueryOrderByClientOrderID(ctx, symbol, clientOID)
	if err != nil {
		return nil, err
	}
	return orderViewFromBinance(res), nil
}

func (b *BinanceVenue) QueryByVenueOrderID(ctx context.Context, symbol string, orderID int64) (*OrderView, error) {
	res, err := b.C.QueryOrderByID(ctx, symbol, orderID)
	if err != nil {
		return nil, err
	}
	return orderViewFromBinance(res), nil
}

func (b *BinanceVenue) UserTrades(ctx context.Context, symbol string, orderID int64) ([]report.FillRecord, error) {
	ff, err := b.C.UserTradesForOrder(ctx, symbol, orderID)
	if err != nil {
		return nil, err
	}
	out := make([]report.FillRecord, 0, len(ff))
	for _, t := range ff {
		price, _ := strconv.ParseFloat(t.Price, 64)
		qty, _ := strconv.ParseFloat(t.Qty, 64)
		fee, _ := strconv.ParseFloat(t.Commission, 64)
		side := domain.SideSell
		if strings.EqualFold(t.Side, "BUY") {
			side = domain.SideBuy
		}
		fillID := strconv.FormatInt(t.ID, 10)
		out = append(out, report.FillRecord{
			FillID:                  fillID,
			Symbol:                  t.Symbol,
			Side:                    side,
			Price:                   price,
			Quantity:                qty,
			Fee:                     fee,
			FeeAsset:                t.CommissionAsset,
			ExchangeTradeTimeUnixMs: t.Time,
		})
	}
	return out, nil
}

func (b *BinanceVenue) AccountAndPositions(ctx context.Context) (*VenueAccountView, error) {
	acct, err := b.C.FuturesAccount(ctx)
	if err != nil {
		return nil, err
	}
	v := &VenueAccountView{
		EquityUSDT:           acct.TotalMarginBalance,
		WalletBalanceUSDT:    acct.TotalWalletBalance,
		AvailableBalanceUSDT: acct.AvailableBalance,
		ExchangeTimeUnixMs:   acct.ExchangeTimeUnixMs,
	}
	for _, pr := range acct.PositionsRisk {
		if pr.Symbol == "" {
			continue
		}
		if pr.Nominal == 0 {
			continue
		}
		v.Positions = append(v.Positions, report.PositionSnapshot{
			Symbol:                     pr.Symbol,
			PositionQty:                pr.Nominal,
			EntryPrice:                 parseFloatString(pr.EntryPrice),
			UnrealizedPnlUSDT:          pr.Pnl,
			InitialMarginUSDT:          pr.IniMg,
			MaintenanceMarginUSDT:      pr.MaintMg,
			Leverage:                   pr.Lev,
			ExchangePositionTimeUnixMs: pickTime(pr.UpdateTime, acct.ExchangeTimeUnixMs),
		})
	}
	return v, nil
}

func (b *BinanceVenue) OpenOrders(ctx context.Context, symbol string) ([]OpenOrderView, error) {
	oo, err := b.C.OpenOrders(ctx, symbol)
	if err != nil {
		return nil, err
	}
	out := make([]OpenOrderView, 0, len(oo))
	for _, o := range oo {
		side := domain.SideSell
		if strings.EqualFold(o.Side, "BUY") {
			side = domain.SideBuy
		}
		price, _ := strconv.ParseFloat(o.Price, 64)
		qty, _ := strconv.ParseFloat(o.OrigQty, 64)
		filled, _ := strconv.ParseFloat(o.ExecutedQty, 64)
		ut := pickTime(o.UpdateTime, o.Time)
		out = append(out, OpenOrderView{
			VenueOrderID: o.OrderID,
			Symbol:       o.Symbol,
			Side:         side,
			Price:        price,
			Quantity:     qty,
			FilledQty:    filled,
			Status:       mapBinanceReportOrderStatus(o.Status),
			ReduceOnly:   o.ReduceOnly,
			UpdateUnixMs: ut,
		})
	}
	return out, nil
}

func orderViewFromBinance(r *binance.OrderResult) *OrderView {
	if r == nil {
		return nil
	}
	side := domain.SideSell
	if strings.EqualFold(r.Side, "BUY") {
		side = domain.SideBuy
	}
	orig, _ := strconv.ParseFloat(r.OrigQty, 64)
	return &OrderView{
		VenueOrderID:  r.OrderID,
		ClientOrderID: r.ClientOrderID,
		Symbol:        r.Symbol,
		Side:          side,
		Status:        r.Status,
		AvgPrice:      r.AvgPx,
		ExecutedQty:   r.ExecQty,
		UpdateUnixMs:  r.UpdateTimeUnixMsFilled,
		ReduceOnly:    r.ReduceOnly,
		OrigQuantity:  orig,
	}
}

func mapBinanceReportOrderStatus(s string) report.OrderStatus {
	switch strings.ToUpper(s) {
	case "NEW":
		return report.OrderStatusNew
	case "PARTIALLY_FILLED":
		return report.OrderStatusPartiallyFilled
	case "FILLED":
		return report.OrderStatusFilled
	case "CANCELED", "CANCELLED":
		return report.OrderStatusCanceled
	case "REJECTED":
		return report.OrderStatusRejected
	case "EXPIRED":
		return report.OrderStatusExpired
	default:
		return report.OrderStatusNew
	}
}

func parseFloatString(s string) float64 {
	f, _ := strconv.ParseFloat(s, 64)
	return f
}

func pickTime(a, b int64) int64 {
	if a != 0 {
		return a
	}
	return b
}
