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
	// Create はスペースの末尾にステータスを追加し、その ID を返す
	Create(status domain.WorkflowStatus) (int, error)
	// Rename はステータスの名前を変える。存在しなければ not found のエラーを返す
	Rename(id int, name string) error
	// SetWIPLimit はステータスの列の最大件数を変える（nil は制限なし）。存在しなければ not found のエラーを返す
	SetWIPLimit(id int, limit *int) error
	// Reorder はスペースのステータスを ids の順に並べ直す（ids はそのスペースのステータスの ID をすべて含む）
	Reorder(spaceID int, ids []int) error
	// Delete はステータスを削除する。存在しなければ not found のエラーを返す
	Delete(id int) error
}
