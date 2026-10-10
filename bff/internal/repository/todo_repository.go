package repository

import "todo-api/internal/domain"

type TodoRepository interface {
	FindAll() ([]domain.Todo, error)
	FindByID(id int) (domain.Todo, error)
	// FindByKey は課題キーの接頭辞と番号でタスクを探す。存在しなければ nil を返す
	FindByKey(prefix string, number int) (*domain.Todo, error)
	// Create はタスクを作成し、その ID を返す。todo.Labels のラベルを付ける（なければ作る）
	Create(todo domain.Todo) (int, error)
	// Update はタスクを更新する。ラベルは todo.Labels に置き換える
	Update(id int, todo domain.Todo) error
	Delete(id int) error
	// FindLabels はいずれかのタスクに付いているラベルの名前を名前順に返す
	FindLabels() ([]string, error)
	// Move はタスクを beforeID のタスクの直前へ移動する。beforeID が 0 の場合は末尾へ移動する
	Move(id int, beforeID int) error
}
