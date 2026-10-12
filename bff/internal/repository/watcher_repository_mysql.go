package repository

import (
	"database/sql"

	"todo-api/internal/domain"
)

type watcherMySQLRepository struct {
	db *sql.DB
}

func NewWatcherMySQLRepository(db *sql.DB) WatcherRepository {
	return &watcherMySQLRepository{db: db}
}

func (r *watcherMySQLRepository) Add(todoID int, userID int) error {
	if userID == 0 {
		return nil
	}
	_, err := r.db.Exec(`INSERT IGNORE INTO watchers (todo_id, user_id, created_at) VALUES (?, ?, ?)`, todoID, userID, currentTime())
	return err
}

func (r *watcherMySQLRepository) Remove(todoID int, userID int) error {
	_, err := r.db.Exec(`DELETE FROM watchers WHERE todo_id = ? AND user_id = ?`, todoID, userID)
	return err
}

func (r *watcherMySQLRepository) FindByTodo(todoID int) ([]domain.UserSummary, error) {
	rows, err := r.db.Query(
		`SELECT u.id, u.display_name, u.avatar_color FROM watchers w JOIN users u ON u.id = w.user_id
		 WHERE w.todo_id = ? ORDER BY u.display_name, u.id`, todoID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	users := []domain.UserSummary{}
	for rows.Next() {
		var user domain.UserSummary
		if err := rows.Scan(&user.ID, &user.DisplayName, &user.AvatarColor); err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	return users, rows.Err()
}

type notificationMySQLRepository struct {
	db *sql.DB
}

func NewNotificationMySQLRepository(db *sql.DB) NotificationRepository {
	return &notificationMySQLRepository{db: db}
}

// Create は通知を 1 つのトランザクションで作成する（同じ操作の通知は同じ日時になる）
func (r *notificationMySQLRepository) Create(notifications []domain.Notification) error {
	if len(notifications) == 0 {
		return nil
	}
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	now := currentTime()
	for _, n := range notifications {
		if _, err := tx.Exec(
			`INSERT INTO notifications (user_id, todo_id, actor_id, kind, detail, created_at) VALUES (?, ?, ?, ?, ?, ?)`,
			n.UserID, n.TodoID, n.ActorID, string(n.Kind), n.Detail, now); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (r *notificationMySQLRepository) FindByUser(userID int, limit int) ([]domain.Notification, error) {
	rows, err := r.db.Query(
		`SELECT n.id, n.user_id, n.todo_id, n.actor_id, n.kind, n.detail, n.read_at IS NOT NULL, n.created_at,
		        t.title, t.space_id, s.space_key, t.number, u.display_name, u.avatar_color
		 FROM notifications n
		 JOIN todos t ON t.id = n.todo_id
		 LEFT JOIN spaces s ON s.id = t.space_id
		 LEFT JOIN users u ON u.id = n.actor_id
		 WHERE n.user_id = ? ORDER BY n.id DESC LIMIT ?`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	notifications := []domain.Notification{}
	for rows.Next() {
		var n domain.Notification
		var kind string
		var spaceID, number int
		var spaceKey sql.NullString
		var actor nullUser
		if err := rows.Scan(&n.ID, &n.UserID, &n.TodoID, &n.ActorID, &kind, &n.Detail, &n.Read, &n.CreatedAt,
			&n.IssueTitle, &spaceID, &spaceKey, &number, &actor.name, &actor.color); err != nil {
			return nil, err
		}
		n.Kind = domain.NotificationKind(kind)
		n.IssueKey = issueKey(spaceID, spaceKey.String, number)
		n.Actor = actor.summary(n.ActorID)
		notifications = append(notifications, n)
	}
	return notifications, rows.Err()
}

func (r *notificationMySQLRepository) CountUnread(userID int) (int, error) {
	var count int
	err := r.db.QueryRow(
		`SELECT COUNT(*) FROM notifications n JOIN todos t ON t.id = n.todo_id WHERE n.user_id = ? AND n.read_at IS NULL`,
		userID).Scan(&count)
	return count, err
}

func (r *notificationMySQLRepository) MarkRead(id int, userID int) (bool, error) {
	// 既読の通知を既読にしても日時は変えない。ClientFoundRows なので、既読でも一致した行として数える
	result, err := r.db.Exec(
		`UPDATE notifications SET read_at = COALESCE(read_at, ?) WHERE id = ? AND user_id = ?`, currentTime(), id, userID)
	if err != nil {
		return false, err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

func (r *notificationMySQLRepository) MarkAllRead(userID int) error {
	_, err := r.db.Exec(`UPDATE notifications SET read_at = ? WHERE user_id = ? AND read_at IS NULL`, currentTime(), userID)
	return err
}
