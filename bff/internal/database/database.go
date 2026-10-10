package database

import (
	"database/sql"
	"fmt"
	"os"
	"time"

	"github.com/go-sql-driver/mysql"
)

// DefaultDSN は環境変数 DB_DSN が未設定のときの接続先（ローカルの MySQL を想定）
const DefaultDSN = "kanban:kanban@tcp(localhost:3306)/kanban"

// DSNFromEnv は環境変数 DB_DSN の接続文字列を返す。未設定なら DefaultDSN を返す
func DSNFromEnv() string {
	if dsn := os.Getenv("DB_DSN"); dsn != "" {
		return dsn
	}
	return DefaultDSN
}

// Open は MySQL に接続する。起動直後の MySQL を待てるよう、timeout の間は接続を再試行する
func Open(dsn string, timeout time.Duration) (*sql.DB, error) {
	cfg, err := mysql.ParseDSN(dsn)
	if err != nil {
		return nil, fmt.Errorf("invalid DB_DSN: %w", err)
	}
	cfg.ParseTime = true
	// UPDATE の RowsAffected を「変更した行数」ではなく「一致した行数」にする。
	// 値が変わらない更新を「存在しない」と誤判定しないため
	cfg.ClientFoundRows = true

	db, err := sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		return nil, err
	}

	deadline := time.Now().Add(timeout)
	for {
		err = db.Ping()
		if err == nil {
			return db, nil
		}
		if time.Now().After(deadline) {
			db.Close()
			return nil, fmt.Errorf("failed to connect to MySQL: %w", err)
		}
		time.Sleep(time.Second)
	}
}

// Migrate はテーブルがなければ作成する。何度実行してもよい
func Migrate(db *sql.DB) error {
	for _, stmt := range schemaStatements() {
		if _, err := db.Exec(stmt); err != nil {
			return fmt.Errorf("failed to migrate: %w", err)
		}
	}
	for _, c := range addedColumns() {
		if err := addColumnIfMissing(db, c.table, c.column, c.definition); err != nil {
			return fmt.Errorf("failed to migrate: %w", err)
		}
	}
	return nil
}

type column struct {
	table      string
	column     string
	definition string
}

// addedColumns は既存のテーブルに後から追加した列を返す。
// 追加前に作られたデータベースでも同じ列になるよう、なければ ALTER TABLE で追加する。
// 既存の行の作成日時・更新日時は、列を追加した時点の日時になる
func addedColumns() []column {
	return []column{
		{"epics", "created_at", "DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP"},
		{"epics", "updated_at", "DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP"},
		{"todos", "created_at", "DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP"},
		{"todos", "updated_at", "DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP"},
	}
}

func addColumnIfMissing(db *sql.DB, table, column, definition string) error {
	var count int
	err := db.QueryRow(
		`SELECT COUNT(*) FROM information_schema.columns
		 WHERE table_schema = DATABASE() AND table_name = ? AND column_name = ?`, table, column).Scan(&count)
	if err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	_, err = db.Exec("ALTER TABLE `" + table + "` ADD COLUMN `" + column + "` " + definition)
	return err
}

// schemaStatements はテーブル定義を返す。後から追加した列は addedColumns にある。
// EpicID・SpaceID の 0 は「未割り当て」を表すため、外部キー制約は付けない
func schemaStatements() []string {
	return []string{
		`CREATE TABLE IF NOT EXISTS spaces (
			id    INT AUTO_INCREMENT PRIMARY KEY,
			title VARCHAR(255) NOT NULL
		) DEFAULT CHARSET = utf8mb4`,
		`CREATE TABLE IF NOT EXISTS epics (
			id       INT AUTO_INCREMENT PRIMARY KEY,
			title    VARCHAR(255) NOT NULL,
			space_id INT NOT NULL DEFAULT 0
		) DEFAULT CHARSET = utf8mb4`,
		`CREATE TABLE IF NOT EXISTS todos (
			id          INT AUTO_INCREMENT PRIMARY KEY,
			title       VARCHAR(255) NOT NULL,
			done        BOOLEAN NOT NULL DEFAULT FALSE,
			deadline    DATE NULL,
			status      VARCHAR(20) NOT NULL,
			epic_id     INT NOT NULL DEFAULT 0,
			description TEXT NOT NULL,
			space_id    INT NOT NULL DEFAULT 0,
			position    INT NOT NULL,
			INDEX idx_todos_position (position)
		) DEFAULT CHARSET = utf8mb4`,
	}
}
