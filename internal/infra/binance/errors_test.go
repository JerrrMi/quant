package binance_test

import (
	"errors"
	"testing"

	"github.com/JerrrMi/quant/internal/infra/binance"
)

func TestIsDuplicateClientOrder(t *testing.T) {
	err := binance.ParseAPIErrorFromResponse(400, []byte(`{"code":1,"msg":"Duplicate order sent"}`), nil)
	if !binance.IsDuplicateClientOrder(err) {
		t.Fatal("expected duplicate match")
	}
	if binance.IsDuplicateClientOrder(errors.New("other")) {
		t.Fatal("false positive")
	}
}
