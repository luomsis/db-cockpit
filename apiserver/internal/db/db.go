package db

import (
	"fmt"
	"log"
	"strings"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"db-cockpit/apiserver/internal/model"
)

// Open：连接既有 PostgreSQL 实例；若目标库不存在则先自动创建（幂等）。
func Open(dsn string) (*gorm.DB, error) {
	name := dbnameOf(dsn)
	if name == "" {
		return nil, fmt.Errorf("DB_DSN 缺少 dbname")
	}
	if err := ensureDatabase(dsn, name); err != nil {
		return nil, fmt.Errorf("ensure database %s: %w", name, err)
	}
	gdb, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		return nil, fmt.Errorf("connect %s: %w", name, err)
	}
	if err := gdb.AutoMigrate(model.AllModels()...); err != nil {
		return nil, fmt.Errorf("automigrate: %w", err)
	}
	log.Printf("[db] connected, database=%s", name)
	return gdb, nil
}

// ensureDatabase：连 postgres 维护库检查 pg_database，缺则创建。
func ensureDatabase(dsn, name string) error {
	admin, err := gorm.Open(postgres.Open(withDBName(dsn, "postgres")), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return err
	}
	var n int64
	if err := admin.Raw("SELECT count(*) FROM pg_database WHERE datname = ?", name).Scan(&n).Error; err != nil {
		return err
	}
	if n == 0 {
		log.Printf("[db] creating database %s", name)
		if err := admin.Exec(fmt.Sprintf(`CREATE DATABASE %q`, name)).Error; err != nil {
			return err
		}
	}
	sqlDB, _ := admin.DB()
	_ = sqlDB.Close()
	return nil
}

// dbnameOf / withDBName：解析 key=value 形式的 DSN 中的 dbname。
func dbnameOf(dsn string) string {
	for _, kv := range strings.Fields(dsn) {
		if strings.HasPrefix(kv, "dbname=") {
			return strings.Trim(strings.TrimPrefix(kv, "dbname="), `"'`)
		}
	}
	return ""
}

func withDBName(dsn, name string) string {
	parts := strings.Fields(dsn)
	out := make([]string, 0, len(parts))
	replaced := false
	for _, kv := range parts {
		if strings.HasPrefix(kv, "dbname=") {
			kv = "dbname=" + name
			replaced = true
		}
		out = append(out, kv)
	}
	if !replaced {
		out = append(out, "dbname="+name)
	}
	return strings.Join(out, " ")
}
