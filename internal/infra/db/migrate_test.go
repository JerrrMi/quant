package db_test

import (
	"testing"

	"github.com/JerrrMi/quant/internal/infra"
	"github.com/JerrrMi/quant/internal/infra/db"
)

func TestAutoMigrateAll_OK(t *testing.T) {
	gormDB, err := infra.OpenSQLite("file::memory:?cache=private")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrateAll(gormDB); err != nil {
		t.Fatalf("automigrate: %v", err)
	}
}

func TestAutoMigrateAll_NilDB(t *testing.T) {
	err := db.AutoMigrateAll(nil)
	if err == nil {
		t.Fatal("expected error for nil db")
	}
}
