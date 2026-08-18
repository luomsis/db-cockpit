package seed

import (
	"testing"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"db-cockpit/apiserver/internal/model"
)

// 集成测试：连本地既有 PG（postgres-age:55432）验证种子幂等。
// 不可达时跳过（不阻塞纯单元环境）。
func openTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "host=localhost port=55432 user=graphiti password=graphiti dbname=db_cockpit sslmode=disable"
	gdb, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Skipf("local postgres unreachable: %v", err)
	}
	return gdb
}

func TestSeedIdempotent(t *testing.T) {
	gdb := openTestDB(t)
	if err := Run(gdb); err != nil {
		t.Fatalf("seed run 1: %v", err)
	}
	var n1 int64
	gdb.Model(&model.Cluster{}).Count(&n1)
	if n1 != 4 {
		t.Fatalf("clusters after seed: got %d want 4", n1)
	}
	if err := Run(gdb); err != nil {
		t.Fatalf("seed run 2 (idempotency): %v", err)
	}
	var n2 int64
	gdb.Model(&model.Cluster{}).Count(&n2)
	if n2 != 4 {
		t.Fatalf("second seed must not duplicate: got %d clusters", n2)
	}
}
