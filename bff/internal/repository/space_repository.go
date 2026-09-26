package repository

import "todo-api/internal/domain"

type SpaceRepository interface {
	FindAll() ([]domain.Space, error)
	Create(space domain.Space) error
	Delete(id int) error
}
