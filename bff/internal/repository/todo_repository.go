package repository

import "todo-api/internal/domain"

type TodoRepository interface {
	FindAll() ([]domain.Todo, error)
	Create(todo domain.Todo) error
	Update(id int, todo domain.Todo) error
	Delete(id int) error
	// Move はタスクを beforeID のタスクの直前へ移動する。beforeID が 0 の場合は末尾へ移動する
	Move(id int, beforeID int) error
}
