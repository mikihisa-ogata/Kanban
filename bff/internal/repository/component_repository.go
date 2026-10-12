package repository

import "todo-api/internal/domain"

type ComponentRepository interface {
	// FindAll はすべてのコンポーネントを、スペース・名前順に返す
	FindAll() ([]domain.Component, error)
	// FindByID はコンポーネントを返す。存在しなければ nil を返す
	FindByID(id int) (*domain.Component, error)
	// Create はコンポーネントを作成し、その ID を返す
	Create(component domain.Component) (int, error)
	// Update はコンポーネントの名前・説明・既定の担当者を変える。存在しなければ not found のエラーを返す
	Update(component domain.Component) error
	// Delete はコンポーネントと、課題へのその付与を削除する。存在しなければ not found のエラーを返す
	Delete(id int) error
	// SetTodoComponents は課題のコンポーネントを componentIDs に置き換える
	SetTodoComponents(todoID int, componentIDs []int) error
}
