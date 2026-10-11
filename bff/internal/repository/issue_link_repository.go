package repository

import "todo-api/internal/domain"

type IssueLinkRepository interface {
	// FindAll はすべての課題リンクを作成順（ID 順）に返す
	FindAll() ([]domain.Link, error)
	// Create は課題リンクを作成し、その ID を返す
	Create(link domain.Link) (int, error)
	// Delete は課題リンクを削除する。存在しなければ not found のエラーを返す
	Delete(id int) error
}
