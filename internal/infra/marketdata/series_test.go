package marketdata

import (
	"math"
	"testing"

	"github.com/JerrrMi/quant/internal/domain"
)

func TestLogReturnLast(t *testing.T) {
	w := []domain.Bar{
		{Close: 100, TimestampUnixMs: 1},
		{Close: 101, TimestampUnixMs: 2},
	}
	lr, err := LogReturnLast(w)
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(lr-math.Log(101.0/100.0)) > 1e-9 {
		t.Fatalf("lr %v", lr)
	}
}

func TestBuildPriceWindow(t *testing.T) {
	bars := []domain.Bar{{Close: 1}, {Close: 2}, {Close: 3}}
	got := BuildPriceWindow(bars, 2)
	if len(got) != 2 || got[0].Close != 2 || got[1].Close != 3 {
		t.Fatalf("%+v", got)
	}
}
