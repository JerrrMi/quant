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

// 重复 AutoMigrate 不得破坏已有数据（GORM 演进底座可多次调用）。
func TestAutoMigrateAll_IdempotentOnSameConnection(t *testing.T) {
	gormDB, err := infra.OpenSQLite("file::memory:?cache=private")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrateAll(gormDB); err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrateAll(gormDB); err != nil {
		t.Fatalf("second automigrate: %v", err)
	}
}
