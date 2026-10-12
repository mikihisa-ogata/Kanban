package repository

import "todo-api/internal/domain"

type CommentRepository interface {
	// FindByTodo は課題のコメントを投稿順（古い順）に返す
	FindByTodo(todoID int) ([]domain.Comment, error)
	// FindByID はコメントを返す。存在しなければ nil を返す
	FindByID(id int) (*domain.Comment, error)
	// Create はコメントを作成し、その ID を返す。投稿者は comment.AuthorID、投稿日時は現在日時にする
	Create(comment domain.Comment) (int, error)
	// Update はコメントの本文を変え、編集日時を現在日時にする。存在しなければ not found のエラーを返す
	Update(id int, body string) error
	// Delete はコメントを削除する。存在しなければ not found のエラーを返す
	Delete(id int) error
}
