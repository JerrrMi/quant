package backtest

import (
	"testing"

	"github.com/JerrrMi/quant/internal/domain"
)

func TestDecimateBars_stride(t *testing.T) {
	bars := []domain.Bar{
		{TimestampUnixMs: 1, Close: 1},
		{TimestampUnixMs: 2, Close: 2},
		{TimestampUnixMs: 3, Close: 3},
		{TimestampUnixMs: 4, Close: 4},
	}
	out := DecimateBars(bars, 2)
	if len(out) != 2 || out[0].Close != 1 || out[1].Close != 3 {
		t.Fatalf("got %+v", out)
	}
	if DecimateBars(bars, 1) == nil || len(DecimateBars(bars, 1)) != 4 {
		t.Fatal("stride 1")
	}
	if len(DecimateBars(nil, 3)) != 0 {
		t.Fatal("nil")
	}
}
