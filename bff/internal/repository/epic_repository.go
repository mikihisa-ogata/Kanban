package repository

import "todo-api/internal/domain"

type EpicRepository interface {
	FindAll() ([]domain.Epic, error)
	Create(epic domain.Epic) error
	Update(id int, epic domain.Epic) error
	Delete(id int) error
}
