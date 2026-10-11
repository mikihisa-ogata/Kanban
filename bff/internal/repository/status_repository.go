package repository

import "todo-api/internal/domain"

type StatusRepository interface {
	// FindAll はすべてのワークフローのステータスを、スペース・並び順の順に返す
	FindAll() ([]domain.WorkflowStatus, error)
	// FindBySpace はスペース（0 は未割り当て）のステータスを並び順に返す
	FindBySpace(spaceID int) ([]domain.WorkflowStatus, error)
	// EnsureDefaults はスペースにステータスが 1 つもなければ、既定の 4 ステータスを作る
	EnsureDefaults(spaceID int) error
	// DeleteBySpace はスペースのステータスをすべて削除する
	DeleteBySpace(spaceID int) error
}
