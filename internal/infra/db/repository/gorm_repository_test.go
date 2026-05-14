package repository_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/JerrrMi/quant/internal/infra"
	"github.com/JerrrMi/quant/internal/infra/db"
	"github.com/JerrrMi/quant/internal/infra/db/models"
	"github.com/JerrrMi/quant/internal/infra/db/repository"
)

// 约束：AutoMigrate + 基础仓储 CRUD 可复现；不依赖外部数据库。
func TestGormInstanceRepository_CreateGetRoundTrip(t *testing.T) {
	g, err := infra.OpenSQLite("file::memory:?cache=private")
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrateAll(g); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()

	u := models.User{Email: "repo-test-1@example.com"}
	if err := g.WithContext(ctx).Create(&u).Error; err != nil {
		t.Fatal(err)
	}
	s := models.Strategy{
		UserID:     u.ID,
		Name:       "minimal",
		Kind:       "stub",
		ConfigJSON: "{}",
	}
	if err := g.WithContext(ctx).Create(&s).Error; err != nil {
		t.Fatal(err)
	}

	inst := models.Instance{
		UserID:     u.ID,
		StrategyID: s.ID,
		AgentKey:   "agent-key-alpha",
		Status:     "active",
	}
	repo := repository.NewGormInstanceRepository(g)
	if err := repo.Create(ctx, &inst); err != nil {
		t.Fatal(err)
	}
	if inst.ID == 0 {
		t.Fatal("expected auto id")
	}

	got, err := repo.GetByID(ctx, inst.ID)
	if err != nil || got == nil || got.AgentKey != inst.AgentKey {
		t.Fatalf("get by id: %+v err %v", got, err)
	}
	byKey, err := repo.GetByAgentKey(ctx, "agent-key-alpha")
	if err != nil || byKey == nil || byKey.ID != inst.ID {
		t.Fatalf("get by agent key: %+v err %v", byKey, err)
	}
	active, err := repo.ListActive(ctx)
	if err != nil || len(active) != 1 {
		t.Fatalf("list active: %+v err %v", active, err)
	}
}

// 模拟进程重启：关闭连接后重新打开同一 SQLite 文件，数据仍在。
func TestSQLiteFile_survivesReconnect(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.sqlite3")
	dsn := "file:" + path + "?cache=private"

	g1, err := infra.OpenSQLite(dsn)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrateAll(g1); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	u := models.User{Email: "reconnect@example.com"}
	if err := g1.WithContext(ctx).Create(&u).Error; err != nil {
		t.Fatal(err)
	}
	sqlDB, err := g1.DB()
	if err != nil {
		t.Fatal(err)
	}
	_ = sqlDB.Close()

	g2, err := infra.OpenSQLite(dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		sqlDB, cerr := g2.DB()
		if cerr == nil {
			_ = sqlDB.Close()
		}
	})
	var count int64
	if err := g2.WithContext(ctx).Model(&models.User{}).Where("email = ?", "reconnect@example.com").Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("expected user after reopen, count=%d", count)
	}
}
