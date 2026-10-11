package database

import (
	"database/sql"
	"fmt"
	"os"
	"time"

	"github.com/go-sql-driver/mysql"

	"todo-api/internal/domain"
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
	for _, i := range addedIndexes() {
		if err := addIndexIfMissing(db, i.table, i.name, i.definition); err != nil {
			return fmt.Errorf("failed to migrate: %w", err)
		}
	}
	if err := assignMissingSpaceKeys(db); err != nil {
		return fmt.Errorf("failed to assign space keys: %w", err)
	}
	if err := assignMissingIssueNumbers(db); err != nil {
		return fmt.Errorf("failed to assign issue numbers: %w", err)
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
		// キーのないスペース（追加前のデータや CSV から取り込んだデータ）は assignMissingSpaceKeys で埋める
		{"spaces", "space_key", "VARCHAR(10) NULL"},
		// 0 は未採番。assignMissingIssueNumbers で埋める
		{"todos", "number", "INT NOT NULL DEFAULT 0"},
		// 課題タイプ（Task / Bug / Story）。追加前のタスクは Task になる
		{"todos", "issue_type", "VARCHAR(20) NOT NULL DEFAULT 'Task'"},
		// 優先度（Highest / High / Medium / Low / Lowest）。追加前のタスクは Medium になる
		{"todos", "priority", "VARCHAR(20) NOT NULL DEFAULT 'Medium'"},
		// ストーリーポイント。NULL は未設定
		{"todos", "story_points", "DECIMAL(8,2) NULL"},
		// サブタスクの親課題の ID。0 はサブタスクではない
		{"todos", "parent_id", "INT NOT NULL DEFAULT 0"},
	}
}

type index struct {
	table      string
	name       string
	definition string
}

// addedIndexes は既存のテーブルに後から追加したインデックスを返す
func addedIndexes() []index {
	return []index{
		{"spaces", "uq_spaces_space_key", "UNIQUE INDEX uq_spaces_space_key (space_key)"},
		{"todos", "idx_todos_space_number", "INDEX idx_todos_space_number (space_id, number)"},
		{"todos", "idx_todos_parent", "INDEX idx_todos_parent (parent_id)"},
	}
}

func addIndexIfMissing(db *sql.DB, table, name, definition string) error {
	var count int
	err := db.QueryRow(
		`SELECT COUNT(*) FROM information_schema.statistics
		 WHERE table_schema = DATABASE() AND table_name = ? AND index_name = ?`, table, name).Scan(&count)
	if err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	_, err = db.Exec("ALTER TABLE `" + table + "` ADD " + definition)
	return err
}

// assignMissingSpaceKeys はキーのないスペースに、名前から作った重複しないキーを ID 順に付ける
func assignMissingSpaceKeys(db *sql.DB) error {
	rows, err := db.Query(`SELECT id, title, space_key FROM spaces ORDER BY id`)
	if err != nil {
		return err
	}
	type space struct {
		id    int
		title string
	}
	taken := map[string]bool{}
	missing := []space{}
	for rows.Next() {
		var s space
		var key sql.NullString
		if err := rows.Scan(&s.id, &s.title, &key); err != nil {
			rows.Close()
			return err
		}
		if key.Valid {
			taken[key.String] = true
		} else {
			missing = append(missing, s)
		}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}

	for _, s := range missing {
		key := domain.SuggestSpaceKey(s.title, taken)
		if _, err := db.Exec(`UPDATE spaces SET space_key = ? WHERE id = ?`, key, s.id); err != nil {
			return err
		}
		taken[key] = true
	}
	return nil
}

// assignMissingIssueNumbers は未採番（number = 0）のタスクに、スペースごとに ID 順で連番を振る。
// 番号はそのスペースの採番済みの最大値と issue_counters の続きから振り、最後に issue_counters を追いつかせる
func assignMissingIssueNumbers(db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(
		`UPDATE todos t
		 JOIN (
			SELECT u.id,
			       GREATEST(COALESCE(m.max_number, 0), COALESCE(c.last_number, 0))
			         + ROW_NUMBER() OVER (PARTITION BY u.space_id ORDER BY u.id) AS new_number
			FROM todos u
			LEFT JOIN (SELECT space_id, MAX(number) AS max_number FROM todos GROUP BY space_id) m
			  ON m.space_id = u.space_id
			LEFT JOIN issue_counters c ON c.space_id = u.space_id
			WHERE u.number = 0
		 ) x ON x.id = t.id
		 SET t.number = x.new_number`); err != nil {
		return err
	}
	if _, err := tx.Exec(
		`INSERT INTO issue_counters (space_id, last_number)
		 SELECT * FROM (SELECT space_id, MAX(number) AS max_number FROM todos GROUP BY space_id) AS src
		 ON DUPLICATE KEY UPDATE last_number = GREATEST(last_number, src.max_number)`); err != nil {
		return err
	}
	return tx.Commit()
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
		// space_id ごとの採番済みの最大の番号。0 はスペース未割り当て
		`CREATE TABLE IF NOT EXISTS issue_counters (
			space_id    INT PRIMARY KEY,
			last_number INT NOT NULL
		) DEFAULT CHARSET = utf8mb4`,
		// ラベル（Jira の自由入力のタグ）。既定の照合順序はひらがなとカタカナ、全角と半角などを同じとみなすので、
		// 名前は utf8mb4_bin で完全一致にする（大文字・小文字の違いだけのラベルは repository の setLabels でまとめる）
		`CREATE TABLE IF NOT EXISTS labels (
			id   INT AUTO_INCREMENT PRIMARY KEY,
			name VARCHAR(255) COLLATE utf8mb4_bin NOT NULL,
			UNIQUE INDEX uq_labels_name (name)
		) DEFAULT CHARSET = utf8mb4`,
		// タスクとラベルの多対多
		`CREATE TABLE IF NOT EXISTS todo_labels (
			todo_id  INT NOT NULL,
			label_id INT NOT NULL,
			PRIMARY KEY (todo_id, label_id),
			INDEX idx_todo_labels_label (label_id)
		) DEFAULT CHARSET = utf8mb4`,
		// 課題リンク。source_id の課題が target_id の課題を link_type する（例: source blocks target）
		`CREATE TABLE IF NOT EXISTS issue_links (
			id         INT AUTO_INCREMENT PRIMARY KEY,
			link_type  VARCHAR(20) NOT NULL,
			source_id  INT NOT NULL,
			target_id  INT NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			UNIQUE INDEX uq_issue_links (link_type, source_id, target_id),
			INDEX idx_issue_links_target (target_id)
		) DEFAULT CHARSET = utf8mb4`,
		// 課題へのコメント。updated_at は最後に編集した日時（NULL は未編集）。
		// 投稿者はユーザー機能の追加後に列を足す
		`CREATE TABLE IF NOT EXISTS comments (
			id         INT AUTO_INCREMENT PRIMARY KEY,
			todo_id    INT NOT NULL,
			body       TEXT NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NULL,
			INDEX idx_comments_todo (todo_id)
		) DEFAULT CHARSET = utf8mb4`,
		// 課題の変更履歴。field は項目名（作成は created）、old_value・new_value は変更前・変更後（NULL はなし）
		`CREATE TABLE IF NOT EXISTS issue_history (
			id         INT AUTO_INCREMENT PRIMARY KEY,
			todo_id    INT NOT NULL,
			field      VARCHAR(40) NOT NULL,
			old_value  TEXT NULL,
			new_value  TEXT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			INDEX idx_issue_history_todo (todo_id)
		) DEFAULT CHARSET = utf8mb4`,
		// 課題の添付ファイルの情報。ファイル本体は添付ファイルのディレクトリに storage_name の名前で置く
		`CREATE TABLE IF NOT EXISTS attachments (
			id           INT AUTO_INCREMENT PRIMARY KEY,
			todo_id      INT NOT NULL,
			file_name    VARCHAR(255) NOT NULL,
			content_type VARCHAR(255) NOT NULL,
			size         BIGINT NOT NULL,
			storage_name VARCHAR(64) NOT NULL,
			created_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			INDEX idx_attachments_todo (todo_id)
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
