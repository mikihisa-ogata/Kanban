package database

import (
	"os"
	"testing"
	"time"
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
		"epics":  {"id", "title", "space_id"},
		"todos":  {"id", "title", "done", "deadline", "status", "epic_id", "description", "space_id", "position"},
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
