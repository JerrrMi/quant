package binance

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// MarkPrice returns the mark price for a linear symbol (e.g. BTCUSDT).
func (c *Client) MarkPrice(ctx context.Context, symbol string) (float64, error) {
	q := url.Values{}
	q.Set("symbol", strings.ToUpper(symbol))
	raw, _, err := c.doPublicGET(ctx, "/fapi/v1/premiumIndex", q)
	if err != nil {
		return 0, err
	}
	var out struct {
		MarkPrice string `json:"markPrice"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return 0, fmt.Errorf("binance: parse mark price: %w", err)
	}
	return strconv.ParseFloat(out.MarkPrice, 64)
}

// LotStepSize returns the LOT_SIZE step for quantity rounding.
func (c *Client) LotStepSize(ctx context.Context, symbol string) (float64, error) {
	sym := strings.ToUpper(symbol)
	c.symMu.Lock()
	if v, ok := c.lot[sym]; ok {
		c.symMu.Unlock()
		return v, nil
	}
	c.symMu.Unlock()

	q := url.Values{}
	q.Set("symbol", sym)
	raw, _, err := c.doPublicGET(ctx, "/fapi/v1/exchangeInfo", q)
	if err != nil {
		return 0, err
	}
	var info struct {
		Symbols []struct {
			Symbol  string `json:"symbol"`
			Filters []struct {
				FilterType string `json:"filterType"`
				StepSize   string `json:"stepSize"`
			} `json:"filters"`
		} `json:"symbols"`
	}
	if err := json.Unmarshal(raw, &info); err != nil {
		return 0, fmt.Errorf("binance: parse exchangeInfo: %w", err)
	}
	if len(info.Symbols) != 1 {
		return 0, fmt.Errorf("binance: exchangeInfo: symbol %q not found", sym)
	}
	step := 0.0
	for _, f := range info.Symbols[0].Filters {
		if strings.EqualFold(f.FilterType, "LOT_SIZE") {
			step, err = strconv.ParseFloat(f.StepSize, 64)
			if err != nil {
				return 0, fmt.Errorf("binance: parse stepSize: %w", err)
			}
			break
		}
	}
	if step <= 0 {
		return 0, fmt.Errorf("binance: LOT_SIZE missing for %q", sym)
	}
	c.symMu.Lock()
	c.lot[sym] = step
	c.symMu.Unlock()
	return step, nil
}
