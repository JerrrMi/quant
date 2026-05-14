package binance

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// OrderResult summarizes a futures order REST response subset.
type OrderResult struct {
	OrderID                  int64  `json:"orderId"`
	Symbol                   string `json:"symbol"`
	ClientOrderID            string `json:"clientOrderId"`
	Status                   string `json:"status"`
	Side                     string `json:"side"`
	Type                     string `json:"type"`
	Price                    string `json:"price"`
	AvgPrice                 string `json:"avgPrice"`
	OrigQty                  string `json:"origQty"`
	ExecutedQty              string `json:"executedQty"`
	CumulativeQuoteQty       string `json:"cumQuote"`
	ReduceOnly               bool   `json:"reduceOnly"`
	Time                     int64  `json:"time"`
	UpdateTime               int64  `json:"updateTime"`
	UpdateTimeUnixMsFilled   int64  // prefers UpdateTime || Time for reporting
	AvgPx                    float64
	ExecQty                  float64
}

// TradeFill from user trades.
type TradeFill struct {
	ID              int64  `json:"id"`
	Symbol          string `json:"symbol"`
	OrderID         int64  `json:"orderId"`
	Side            string `json:"side"`
	Price           string `json:"price"`
	Qty             string `json:"qty"`
	RealizedProfit  string `json:"realizedPnl,omitempty"`
	Commission      string `json:"commission"`
	CommissionAsset string `json:"commissionAsset"`
	Time            int64  `json:"time"`
	IsBuyer         bool   `json:"buyer,omitempty"`
	IsMaker         bool   `json:"maker,omitempty"`
}

// PlaceMarketRequest parameters for MARKET execution.
type PlaceMarketRequest struct {
	Symbol           string
	Side             string // BUY / SELL
	QuantityBase     float64
	ReduceOnly       bool
	NewClientOrderID string // optional; uniqueness recommended
}

// PlaceMarket posts a MARKET order.
func (c *Client) PlaceMarket(ctx context.Context, req PlaceMarketRequest) (*OrderResult, error) {
	params := url.Values{}
	params.Set("symbol", strings.ToUpper(req.Symbol))
	params.Set("side", strings.ToUpper(req.Side))
	params.Set("type", "MARKET")
	params.Set("quantity", formatQty(req.QuantityBase))
	if req.ReduceOnly {
		params.Set("reduceOnly", "true")
	}
	if strings.TrimSpace(req.NewClientOrderID) != "" {
		params.Set("newClientOrderId", req.NewClientOrderID)
	}
	raw, _, err := c.doSignedPOST(ctx, "/fapi/v1/order", params)
	if err != nil {
		return nil, err
	}
	return parseOrderResult(raw)
}

// CancelByClientOrderID cancels using origClientOrderId.
func (c *Client) CancelByClientOrderID(ctx context.Context, symbol, clientOrderID string) (*OrderResult, error) {
	params := url.Values{}
	params.Set("symbol", strings.ToUpper(symbol))
	params.Set("origClientOrderId", clientOrderID)
	raw, _, err := c.doSignedDELETE(ctx, "/fapi/v1/order", params)
	if err != nil {
		return nil, err
	}
	return parseOrderResult(raw)
}

// CancelByOrderID cancels by venue order id.
func (c *Client) CancelByOrderID(ctx context.Context, symbol string, orderID int64) (*OrderResult, error) {
	params := url.Values{}
	params.Set("symbol", strings.ToUpper(symbol))
	params.Set("orderId", strconv.FormatInt(orderID, 10))
	raw, _, err := c.doSignedDELETE(ctx, "/fapi/v1/order", params)
	if err != nil {
		return nil, err
	}
	return parseOrderResult(raw)
}

// QueryOrderByClientOrderID fetches status for client order id.
func (c *Client) QueryOrderByClientOrderID(ctx context.Context, symbol, clientOrderID string) (*OrderResult, error) {
	params := url.Values{}
	params.Set("symbol", strings.ToUpper(symbol))
	params.Set("origClientOrderId", clientOrderID)
	raw, _, err := c.doSignedGET(ctx, "/fapi/v1/order", params)
	if err != nil {
		return nil, err
	}
	return parseOrderResult(raw)
}

// QueryOrderByID fetches by exchange order id.
func (c *Client) QueryOrderByID(ctx context.Context, symbol string, orderID int64) (*OrderResult, error) {
	params := url.Values{}
	params.Set("symbol", strings.ToUpper(symbol))
	params.Set("orderId", strconv.FormatInt(orderID, 10))
	raw, _, err := c.doSignedGET(ctx, "/fapi/v1/order", params)
	if err != nil {
		return nil, err
	}
	return parseOrderResult(raw)
}

// UserTradesForOrder pulls fills attributed to orderId for the symbol window.
func (c *Client) UserTradesForOrder(ctx context.Context, symbol string, orderID int64) ([]TradeFill, error) {
	params := url.Values{}
	params.Set("symbol", strings.ToUpper(symbol))
	params.Set("orderId", strconv.FormatInt(orderID, 10))
	params.Set("limit", "500")
	raw, _, err := c.doSignedGET(ctx, "/fapi/v1/userTrades", params)
	if err != nil {
		return nil, err
	}
	var out []TradeFill
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("binance: parse userTrades: %w", err)
	}
	return out, nil
}

func parseOrderResult(raw []byte) (*OrderResult, error) {
	var r OrderResult
	if err := json.Unmarshal(raw, &r); err != nil {
		return nil, fmt.Errorf("binance: parse order: %w", err)
	}
	r.AvgPx, _ = strconv.ParseFloat(r.AvgPrice, 64)
	r.ExecQty, _ = strconv.ParseFloat(r.ExecutedQty, 64)
	r.UpdateTimeUnixMsFilled = r.UpdateTime
	if r.UpdateTimeUnixMsFilled == 0 {
		r.UpdateTimeUnixMsFilled = r.Time
	}
	return &r, nil
}

func formatQty(q float64) string {
	// Avoid scientific notation surprises for Futures quantity.
	s := strconv.FormatFloat(q, 'f', -1, 64)
	s = strings.TrimRight(strings.TrimRight(s, "0"), ".")
	if s == "" {
		return "0"
	}
	return s
}
