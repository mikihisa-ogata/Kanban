// Package dbtest は MySQL を使うテストの準備をする
package dbtest

import (
	"database/sql"
	"os"
	"testing"
	"time"

	"github.com/go-sql-driver/mysql"

	"todo-api/internal/database"
)

// Open は環境変数 TEST_DB_DSN のデータベース名に "_<suffix>" を付けたデータベースを作り、
// テーブルを作成・全削除して返す。TEST_DB_DSN が未設定ならテストをスキップする。
// go test はパッケージを並列に実行するため、パッケージごとに suffix を変えてデータベースを分ける
func Open(t *testing.T, suffix string) *sql.DB {
	t.Helper()
	dsn := os.Getenv("TEST_DB_DSN")
	if dsn == "" {
		t.Skip("TEST_DB_DSN が未設定のため MySQL を使うテストをスキップする")
	}
	cfg, err := mysql.ParseDSN(dsn)
	if err != nil {
		t.Fatal(err)
	}
	cfg.DBName += "_" + suffix
	name := cfg.DBName

	// データベースを作るため、データベースを指定せずに接続する
	cfg.DBName = ""
	admin, err := database.Open(cfg.FormatDSN(), 5*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	if _, err := admin.Exec("CREATE DATABASE IF NOT EXISTS `" + name + "` DEFAULT CHARSET utf8mb4"); err != nil {
		t.Fatal(err)
	}

	cfg.DBName = name
	db, err := database.Open(cfg.FormatDSN(), 5*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if err := database.Migrate(db); err != nil {
		t.Fatal(err)
	}
	for _, table := range []string{"todos", "epics", "spaces"} {
		if _, err := db.Exec("TRUNCATE TABLE " + table); err != nil {
			t.Fatal(err)
		}
	}
	return db
}
