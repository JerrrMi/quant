package binance

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// AccountView aggregates wallet + crosses positions for reporting.
type AccountView struct {
	TotalWalletBalance   float64
	AvailableBalance     float64
	TotalMarginBalance   float64
	ExchangeTimeUnixMs   int64
	PositionsRisk        []PositionRisk
}

// PositionRisk from GET /fapi/v2/positionRisk.
type PositionRisk struct {
	Symbol           string  `json:"symbol"`
	PositionAmt      string  `json:"positionAmt"`
	EntryPrice       string  `json:"entryPrice"`
	UnrealizedProfit string  `json:"unRealizedProfit"`
	Leverage         string  `json:"leverage"`
	InitialMargin    string  `json:"initialMargin"`
	MaintMargin      string  `json:"maintMargin"`
	UpdateTime       int64   `json:"updateTime"` // millis
	Nominal          float64 // filled by parser
	Pnl              float64
	Lev              float64
	IniMg            float64
	MaintMg          float64
}

// OpenOrder futures order view (normalized subset).
type OpenOrder struct {
	OrderID                  int64  `json:"orderId"`
	Symbol                   string `json:"symbol"`
	ClientOrderID            string `json:"clientOrderId"`
	Price                    string `json:"price"`
	OrigQty                  string `json:"origQty"`
	ExecutedQty              string `json:"executedQty"`
	Status                   string `json:"status"`
	Side                     string `json:"side"`
	ReduceOnly               bool   `json:"reduceOnly"`
	Type                     string `json:"type"`
	UpdateTime               int64  `json:"updateTime"` // millis
	Time                     int64  `json:"time"`
}

// FuturesAccount retrieves signed account balances and cross margin summary from /fapi/v2/account.
func (c *Client) FuturesAccount(ctx context.Context) (*AccountView, error) {
	raw, _, err := c.doSignedGET(ctx, "/fapi/v2/account", nil)
	if err != nil {
		return nil, err
	}
	var a struct {
		TotalWalletBalance     string `json:"totalWalletBalance"`
		AvailableBalance       string `json:"availableBalance"`
		TotalMarginBalance     string `json:"totalMarginBalance"`
		UpdateTime             int64  `json:"updateTime"` // millis
		Positions              []struct {
			Symbol                   string `json:"symbol"`
			PositionAmt              string `json:"positionAmt"`
			UpdateTime               int64  `json:"updateTime"` // millis
			EntryPrice               string `json:"entryPrice"`
			UnrealizedProfit         string `json:"unRealizedProfit"`
			Leverage                 string `json:"leverage"`
			InitialMargin            string `json:"initialMargin"`
			MaintMargin              string `json:"maintMargin"`
		} `json:"positions"`
	}
	if err := json.Unmarshal(raw, &a); err != nil {
		return nil, fmt.Errorf("binance: parse account v2: %w", err)
	}
	view := &AccountView{
		ExchangeTimeUnixMs: a.UpdateTime,
	}
	view.TotalWalletBalance, _ = strconv.ParseFloat(a.TotalWalletBalance, 64)
	view.AvailableBalance, _ = strconv.ParseFloat(a.AvailableBalance, 64)
	view.TotalMarginBalance, _ = strconv.ParseFloat(a.TotalMarginBalance, 64)

	// Prefer positionRisk endpoint for authoritative margin fields; fallback to embedded positions snapshot.
	rawPR, _, errPR := c.doSignedGET(ctx, "/fapi/v2/positionRisk", nil)
	if errPR == nil {
		var risks []PositionRisk
		if err := json.Unmarshal(rawPR, &risks); err == nil {
			for i := range risks {
				r := &risks[i]
				r.PositionAmt = strings.TrimSpace(r.PositionAmt)
				r.Nominal, _ = strconv.ParseFloat(r.PositionAmt, 64)
				r.Pnl, _ = strconv.ParseFloat(r.UnrealizedProfit, 64)
				r.Lev, _ = strconv.ParseFloat(r.Leverage, 64)
				r.IniMg, _ = strconv.ParseFloat(r.InitialMargin, 64)
				r.MaintMg, _ = strconv.ParseFloat(r.MaintMargin, 64)
			}
			view.PositionsRisk = risks
		}
	}
	if len(view.PositionsRisk) == 0 {
		for _, p := range a.Positions {
			nom, _ := strconv.ParseFloat(p.PositionAmt, 64)
			if nom == 0 {
				continue
			}
			view.PositionsRisk = append(view.PositionsRisk, PositionRisk{
				Symbol:           p.Symbol,
				PositionAmt:      p.PositionAmt,
				UpdateTime:       p.UpdateTime,
				EntryPrice:       p.EntryPrice,
				UnrealizedProfit: p.UnrealizedProfit,
				Leverage:         p.Leverage,
				InitialMargin:    p.InitialMargin,
				MaintMargin:      p.MaintMargin,
				Nominal:          nom,
				Pnl:              parseSF(p.UnrealizedProfit),
				Lev:              parseSF(p.Leverage),
				IniMg:            parseSF(p.InitialMargin),
				MaintMg:          parseSF(p.MaintMargin),
			})
		}
	}
	return view, nil
}

func parseSF(s string) float64 {
	f, _ := strconv.ParseFloat(s, 64)
	return f
}

// OpenOrders lists open orders optionally filtered by symbol.
func (c *Client) OpenOrders(ctx context.Context, symbol string) ([]OpenOrder, error) {
	params := url.Values{}
	if strings.TrimSpace(symbol) != "" {
		params.Set("symbol", strings.ToUpper(symbol))
	}
	raw, _, err := c.doSignedGET(ctx, "/fapi/v1/openOrders", params)
	if err != nil {
		return nil, err
	}
	var out []OpenOrder
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("binance: parse openOrders: %w", err)
	}
	return out, nil
}
