package repository

import "todo-api/internal/domain"

type TodoRepository interface {
	FindAll() ([]domain.Todo, error)
	Create(todo domain.Todo) error
	Update(id int, todo domain.Todo) error
	Delete(id int) error
}
