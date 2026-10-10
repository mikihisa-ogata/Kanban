package database

import (
	"database/sql"
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
		"spaces":         {"id", "title", "space_key"},
		"issue_counters": {"space_id", "last_number"},
		"labels":         {"id", "name"},
		"todo_labels":    {"todo_id", "label_id"},
		"epics":          {"id", "title", "space_id", "created_at", "updated_at"},
		"todos":          {"id", "title", "done", "deadline", "status", "epic_id", "description", "space_id", "position", "created_at", "updated_at", "number", "issue_type", "priority"},
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
	db := openEmptyDatabase(t, "legacy")

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

	// 課題タイプ・優先度の追加前のタスクは Task・Medium になる
	var issueType, priority string
	if err := db.QueryRow(`SELECT issue_type, priority FROM todos`).Scan(&issueType, &priority); err != nil {
		t.Fatal(err)
	}
	if issueType != "Task" || priority != "Medium" {
		t.Errorf("issue_type = %q, priority = %q, want Task, Medium", issueType, priority)
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
}

// openEmptyDatabase は TEST_DB_DSN のデータベース名に "_<suffix>" を付けた空のデータベースを作って返す。
// テストの終わりにデータベースごと削除する
func openEmptyDatabase(t *testing.T, suffix string) *sql.DB {
	t.Helper()
	cfg, err := mysql.ParseDSN(testDSN(t))
	if err != nil {
		t.Fatal(err)
	}
	name := cfg.DBName + "_" + suffix
	cfg.DBName = ""
	admin, err := Open(cfg.FormatDSN(), 5*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { admin.Close() })
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
	// 後に登録した Cleanup が先に動くので、接続を閉じてからデータベースを消す
	t.Cleanup(func() {
		if _, err := admin.Exec("DROP DATABASE `" + name + "`"); err != nil {
			t.Error(err)
		}
	})
	t.Cleanup(func() { db.Close() })
	return db
}

// 課題キーの導入前のデータベースでも、スペースにキーが付き、タスクにスペースごとの連番が ID 順に振られる
func TestMigrate_AssignsSpaceKeysAndIssueNumbers(t *testing.T) {
	db := openEmptyDatabase(t, "issuekeys")

	legacy := []string{
		`CREATE TABLE spaces (
			id    INT AUTO_INCREMENT PRIMARY KEY,
			title VARCHAR(255) NOT NULL
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
		`INSERT INTO spaces (title) VALUES ('Kanban'), ('日本おすすめ地図'), ('お金')`,
		`INSERT INTO todos (title, status, description, space_id, position) VALUES
			('t1', 'Open', '', 3, 5), ('t2', 'Open', '', 1, 4), ('t3', 'Open', '', 0, 3),
			('t4', 'Open', '', 3, 2), ('t5', 'Open', '', 1, 1)`,
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
	if got := queryString(t, db, `SELECT GROUP_CONCAT(space_key ORDER BY id) FROM spaces`); got != "KAN,SP,SP2" {
		t.Errorf("space keys = %s, want KAN,SP,SP2", got)
	}
	// 並び順（position）ではなく ID 順に振る
	if got := queryString(t, db, `SELECT GROUP_CONCAT(CONCAT(id, ':', space_id, '-', number) ORDER BY id) FROM todos`); got != "1:3-1,2:1-1,3:0-1,4:3-2,5:1-2" {
		t.Errorf("numbers = %s", got)
	}
	if got := queryString(t, db, `SELECT GROUP_CONCAT(CONCAT(space_id, ':', last_number) ORDER BY space_id) FROM issue_counters`); got != "0:1,1:2,3:2" {
		t.Errorf("counters = %s", got)
	}

	// 削除済みの番号（カウンターの方が大きい）があれば、その続きから振る（CSV から取り込んだ未採番のタスクなど）
	if _, err := db.Exec(`UPDATE issue_counters SET last_number = 10 WHERE space_id = 1`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO todos (title, status, description, space_id, position) VALUES ('t6', 'Open', '', 1, 6), ('t7', 'Open', '', 2, 7)`); err != nil {
		t.Fatal(err)
	}
	if err := Migrate(db); err != nil {
		t.Fatal(err)
	}
	if got := queryString(t, db, `SELECT GROUP_CONCAT(CONCAT(id, ':', number) ORDER BY id) FROM todos WHERE id >= 6`); got != "6:11,7:1" {
		t.Errorf("numbers of new todos = %s, want 6:11,7:1", got)
	}
	if got := queryString(t, db, `SELECT GROUP_CONCAT(CONCAT(space_id, ':', last_number) ORDER BY space_id) FROM issue_counters`); got != "0:1,1:11,2:1,3:2" {
		t.Errorf("counters = %s", got)
	}
}

func queryString(t *testing.T, db *sql.DB, query string) string {
	t.Helper()
	var s sql.NullString
	if err := db.QueryRow(query).Scan(&s); err != nil {
		t.Fatal(err)
	}
	return s.String
}
