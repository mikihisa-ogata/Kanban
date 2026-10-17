package repository

import (
	"testing"

	"todo-api/internal/domain"
)

// 一括変更の途中で失敗したら（存在しない課題がある）、先に書いた課題の変更も履歴も残らない
func TestTodoMySQLRepository_UpdateManyRollsBack(t *testing.T) {
	db := openTestDB(t)
	repo := NewTodoMySQLRepository(db)
	history := NewHistoryMySQLRepository(db)
	id, err := repo.Create(domain.Todo{Title: "a", Status: domain.StatusOpen, Labels: []string{"old"}})
	if err != nil {
		t.Fatal(err)
	}
	before, err := repo.FindByID(id)
	if err != nil {
		t.Fatal(err)
	}

	changed := before
	changed.Priority, changed.Labels, changed.AssigneeID, changed.SprintID, changed.FixVersionIDs = domain.PriorityHigh, []string{"new"}, 5, 7, []int{3}
	newValue := "High"
	oldValue := "Medium"
	err = repo.UpdateMany([]domain.BulkIssueChange{
		{Todo: changed, History: []domain.HistoryEntry{{TodoID: id, Field: domain.HistoryFieldPriority, OldValue: &oldValue, NewValue: &newValue, UserID: 1}}, Watcher: 5},
		{Todo: domain.Todo{ID: 999, Priority: domain.PriorityLow}},
	})
	if err == nil {
		t.Fatal("UpdateMany() with a missing todo error = nil")
	}

	after, err := repo.FindByID(id)
	if err != nil {
		t.Fatal(err)
	}
	if after.Priority != before.Priority || len(after.Labels) != 1 || after.Labels[0] != "old" || after.AssigneeID != 0 || after.SprintID != 0 || len(after.FixVersionIDs) != 0 || !after.UpdatedAt.Equal(before.UpdatedAt) {
		t.Errorf("todo after failed UpdateMany = %+v, want %+v", after, before)
	}
	entries, err := history.FindByTodo(id)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Errorf("history after failed UpdateMany = %+v, want none", entries)
	}
	var watchers int
	if err := db.QueryRow(`SELECT COUNT(*) FROM watchers WHERE todo_id = ?`, id).Scan(&watchers); err != nil {
		t.Fatal(err)
	}
	if watchers != 0 {
		t.Errorf("watchers after failed UpdateMany = %d, want 0", watchers)
	}

	// 成功すれば全件の変更と履歴が残る
	if err := repo.UpdateMany([]domain.BulkIssueChange{{Todo: changed, History: []domain.HistoryEntry{{TodoID: id, Field: domain.HistoryFieldPriority, OldValue: &oldValue, NewValue: &newValue, UserID: 1}}, Watcher: 5}}); err != nil {
		t.Fatal(err)
	}
	after, err = repo.FindByID(id)
	if err != nil {
		t.Fatal(err)
	}
	if after.Priority != domain.PriorityHigh || len(after.Labels) != 1 || after.Labels[0] != "new" || after.AssigneeID != 5 || after.SprintID != 7 || len(after.FixVersionIDs) != 1 || after.FixVersionIDs[0] != 3 {
		t.Errorf("todo after UpdateMany = %+v", after)
	}
	if entries, err = history.FindByTodo(id); err != nil || len(entries) != 1 {
		t.Errorf("history after UpdateMany = %+v, %v, want 1 entry", entries, err)
	}
}

// 一括削除の途中で失敗したら（存在しない課題がある）、先に消した課題も残る
func TestTodoMySQLRepository_DeleteManyRollsBack(t *testing.T) {
	repo := NewTodoMySQLRepository(openTestDB(t))
	first, err := repo.Create(domain.Todo{Title: "a", Status: domain.StatusOpen})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repo.Create(domain.Todo{Title: "a のサブタスク", Status: domain.StatusOpen, IssueType: domain.IssueTypeSubtask, ParentID: first}); err != nil {
		t.Fatal(err)
	}
	second, err := repo.Create(domain.Todo{Title: "b", Status: domain.StatusOpen})
	if err != nil {
		t.Fatal(err)
	}

	if err := repo.DeleteMany([]int{first, 999}); err == nil {
		t.Fatal("DeleteMany() with a missing todo error = nil")
	}
	todos, err := repo.FindAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(todos) != 3 {
		t.Fatalf("todos after failed DeleteMany = %d, want 3", len(todos))
	}

	if err := repo.DeleteMany([]int{first, second}); err != nil {
		t.Fatal(err)
	}
	if todos, err = repo.FindAll(); err != nil || len(todos) != 0 {
		t.Errorf("todos after DeleteMany = %+v, %v, want none", todos, err)
	}
}
