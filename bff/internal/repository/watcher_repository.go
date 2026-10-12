package repository

import "todo-api/internal/domain"

type WatcherRepository interface {
	// Add はユーザーを課題のウォッチャーにする。既にウォッチャーなら何もしない。userID が 0 なら何もしない
	Add(todoID int, userID int) error
	// Remove はユーザーを課題のウォッチャーから外す。ウォッチャーでなくてもエラーにしない
	Remove(todoID int, userID int) error
	// FindByTodo は課題のウォッチャーを表示名の順に返す
	FindByTodo(todoID int) ([]domain.UserSummary, error)
}

type NotificationRepository interface {
	// Create は通知を作成する。作成日時は現在日時にし、未読にする
	Create(notifications []domain.Notification) error
	// FindByUser はユーザーへの通知を新しい順に最大 limit 件返す
	FindByUser(userID int, limit int) ([]domain.Notification, error)
	// CountUnread はユーザーへの未読の通知の数を返す
	CountUnread(userID int) (int, error)
	// MarkRead はユーザーへの通知を既読にする。ユーザーへの通知がなければ false を返す
	MarkRead(id int, userID int) (bool, error)
	// MarkAllRead はユーザーへの通知をすべて既読にする
	MarkAllRead(userID int) error
}
