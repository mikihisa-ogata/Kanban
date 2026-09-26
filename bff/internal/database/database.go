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
	return nil
}

// schemaStatements はテーブル定義を返す。
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
