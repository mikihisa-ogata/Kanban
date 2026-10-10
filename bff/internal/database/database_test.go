package database

import (
	"os"
	"testing"
	"time"

	"github.com/go-sql-driver/mysql"
)

// testDSN は環境変数 TEST_DB_DSN の接続文字列を返す。未設定ならテストをスキップする。
// 実データの DB を壊さないよう、テスト専用のデータベースを指定すること
func testDSN(t *testing.T) string {
	t.Helper()
	dsn := os.Getenv("TEST_DB_DSN")
	if dsn == "" {
		t.Skip("TEST_DB_DSN が未設定のため MySQL を使うテストをスキップする")
	}
	return dsn
}

func TestDSNFromEnv(t *testing.T) {
	t.Setenv("DB_DSN", "")
	if got := DSNFromEnv(); got != DefaultDSN {
		t.Errorf("DSNFromEnv() = %q, want %q", got, DefaultDSN)
	}

	t.Setenv("DB_DSN", "user:pass@tcp(db:3306)/app")
	if got := DSNFromEnv(); got != "user:pass@tcp(db:3306)/app" {
		t.Errorf("DSNFromEnv() = %q, want env value", got)
	}
}

func TestOpen_InvalidDSN(t *testing.T) {
	if _, err := Open("not a dsn", 0); err == nil {
		t.Error("Open() error = nil, want error for invalid DSN")
	}
}

func TestOpen_Unreachable(t *testing.T) {
	if _, err := Open("u:p@tcp(127.0.0.1:1)/db", 0); err == nil {
		t.Error("Open() error = nil, want error for unreachable server")
	}
}

func TestMigrate(t *testing.T) {
	dsn := testDSN(t)
	db, err := Open(dsn, 5*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	// 2回実行してもエラーにならない
	for i := 0; i < 2; i++ {
		if err := Migrate(db); err != nil {
			t.Fatalf("Migrate() #%d error = %v", i+1, err)
		}
	}

	want := map[string][]string{
		"spaces": {"id", "title"},
		"epics":  {"id", "title", "space_id", "created_at", "updated_at"},
		"todos":  {"id", "title", "done", "deadline", "status", "epic_id", "description", "space_id", "position", "created_at", "updated_at"},
	}
	for table, columns := range want {
		rows, err := db.Query(
			`SELECT column_name FROM information_schema.columns
			 WHERE table_schema = DATABASE() AND table_name = ? ORDER BY ordinal_position`, table)
		if err != nil {
			t.Fatal(err)
		}
		var got []string
		for rows.Next() {
			var name string
			if err := rows.Scan(&name); err != nil {
				t.Fatal(err)
			}
			got = append(got, name)
		}
		rows.Close()
		if len(got) != len(columns) {
			t.Fatalf("%s columns = %v, want %v", table, got, columns)
		}
		for i := range columns {
			if got[i] != columns[i] {
				t.Errorf("%s columns = %v, want %v", table, got, columns)
				break
			}
		}
	}
}

// 作成日時・更新日時の列がない、追加前のテーブルでも Migrate で列が追加され、既存の行も残る
func TestMigrate_AddsTimestampColumnsToExistingTables(t *testing.T) {
	cfg, err := mysql.ParseDSN(testDSN(t))
	if err != nil {
		t.Fatal(err)
	}
	name := cfg.DBName + "_legacy"
	cfg.DBName = ""
	admin, err := Open(cfg.FormatDSN(), 5*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	if _, err := admin.Exec("DROP DATABASE IF EXISTS `" + name + "`"); err != nil {
		t.Fatal(err)
	}
	if _, err := admin.Exec("CREATE DATABASE `" + name + "` DEFAULT CHARSET utf8mb4"); err != nil {
		t.Fatal(err)
	}

	cfg.DBName = name
	db, err := Open(cfg.FormatDSN(), 5*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	legacy := []string{
		`CREATE TABLE epics (
			id       INT AUTO_INCREMENT PRIMARY KEY,
			title    VARCHAR(255) NOT NULL,
			space_id INT NOT NULL DEFAULT 0
		) DEFAULT CHARSET = utf8mb4`,
		`CREATE TABLE todos (
			id          INT AUTO_INCREMENT PRIMARY KEY,
			title       VARCHAR(255) NOT NULL,
			done        BOOLEAN NOT NULL DEFAULT FALSE,
			deadline    DATE NULL,
			status      VARCHAR(20) NOT NULL,
			epic_id     INT NOT NULL DEFAULT 0,
			description TEXT NOT NULL,
			space_id    INT NOT NULL DEFAULT 0,
			position    INT NOT NULL
		) DEFAULT CHARSET = utf8mb4`,
		`INSERT INTO epics (title) VALUES ('既存のエピック')`,
		`INSERT INTO todos (title, status, description, position) VALUES ('既存のタスク', 'Open', '', 1)`,
	}
	for _, stmt := range legacy {
		if _, err := db.Exec(stmt); err != nil {
			t.Fatal(err)
		}
	}

	for i := 0; i < 2; i++ {
		if err := Migrate(db); err != nil {
			t.Fatalf("Migrate() #%d error = %v", i+1, err)
		}
	}

	for _, table := range []string{"epics", "todos"} {
		var title string
		var createdAt, updatedAt time.Time
		err := db.QueryRow("SELECT title, created_at, updated_at FROM `"+table+"`").Scan(&title, &createdAt, &updatedAt)
		if err != nil {
			t.Fatalf("%s: %v", table, err)
		}
		if title == "" || createdAt.IsZero() || updatedAt.IsZero() {
			t.Errorf("%s: title = %q, created_at = %v, updated_at = %v, want existing row with timestamps", table, title, createdAt, updatedAt)
		}
	}

	if _, err := admin.Exec("DROP DATABASE `" + name + "`"); err != nil {
		t.Fatal(err)
	}
}
