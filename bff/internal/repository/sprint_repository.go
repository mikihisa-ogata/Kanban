package repository

import "todo-api/internal/domain"

type SprintRepository interface {
	// FindAll はすべてのスプリントを、スペース・作成順に返す。完了したスプリントには記録した件数とポイントを入れる
	FindAll() ([]domain.Sprint, error)
	// FindByID はスプリントを返す。存在しなければ nil を返す
	FindByID(id int) (*domain.Sprint, error)
	// Create は未開始（future）のスプリントを作成し、その ID を返す
	Create(sprint domain.Sprint) (int, error)
	// Update はスプリントの名前・ゴール・期間・状態を変える。存在しなければ not found のエラーを返す
	Update(sprint domain.Sprint) error
	// Complete はスプリントを完了（closed）にする。issues（スプリントの課題）を記録し、
	// moveIDs の課題とそのサブタスクを moveTo のスプリント（0 はバックログ）へ移す。まとめて 1 つのトランザクションで行う
	Complete(id int, issues []domain.SprintIssue, moveIDs []int, moveTo int) error
	// Delete はスプリントを削除し、その課題をバックログに戻す。存在しなければ not found のエラーを返す
	Delete(id int) error
	// SetTodoSprint は課題とそのサブタスクのスプリントを変える（0 はバックログ）。課題が存在しなければ not found のエラーを返す
	SetTodoSprint(todoID int, sprintID int) error
	// FindIssues は完了したときに記録したスプリントの課題を、課題の ID 順に返す
	FindIssues(sprintID int) ([]domain.SprintIssue, error)
}
