package repository

import (
	"database/sql"
	"testing"

	"todo-api/internal/database/dbtest"
	"todo-api/internal/domain"
)

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	return dbtest.Open(t, "repository")
}

func TestTodoMySQLRepository_CRUD(t *testing.T) {
	repo := NewTodoMySQLRepository(openTestDB(t))

	description := "1行目, カンマ入り\n2行目 \"引用符\""
	if err := repo.Create(domain.Todo{Title: "a", Deadline: "2026-10-01", Status: domain.StatusOpen, Description: description, EpicID: 2, SpaceID: 3}); err != nil {
		t.Fatal(err)
	}
	if err := repo.Create(domain.Todo{Title: "b", Status: domain.StatusWaiting}); err != nil {
		t.Fatal(err)
	}

	todos, err := repo.FindAll()
	if err != nil {
		t.Fatal(err)
	}
	want := []domain.Todo{
		{ID: 1, Title: "a", Deadline: "2026-10-01", Status: domain.StatusOpen, Description: description, EpicID: 2, SpaceID: 3},
		{ID: 2, Title: "b", Status: domain.StatusWaiting},
	}
	if len(todos) != len(want) {
		t.Fatalf("todos = %+v, want %+v", todos, want)
	}
	for i := range want {
		if todos[i] != want[i] {
			t.Errorf("todos[%d] = %+v, want %+v", i, todos[i], want[i])
		}
	}

	updated := domain.Todo{Title: "a2", Done: true, Status: domain.StatusClosed, Description: "更新後"}
	if err := repo.Update(1, updated); err != nil {
		t.Fatal(err)
	}
	// 値が変わらない更新もエラーにならない
	if err := repo.Update(1, updated); err != nil {
		t.Fatalf("Update() with same values error = %v", err)
	}
	todos, err = repo.FindAll()
	if err != nil {
		t.Fatal(err)
	}
	updated.ID = 1
	if todos[0] != updated {
		t.Errorf("todos[0] = %+v, want %+v", todos[0], updated)
	}

	if err := repo.Update(99, updated); err == nil {
		t.Error("Update(99) error = nil")
	}
	if err := repo.Create(domain.Todo{Title: "c", Deadline: "not a date", Status: domain.StatusOpen}); err == nil {
		t.Error("Create() with invalid deadline error = nil")
	}

	if err := repo.Delete(1); err != nil {
		t.Fatal(err)
	}
	if err := repo.Delete(1); err == nil {
		t.Error("Delete(1) twice error = nil")
	}
	if got := todoIDs(t, repo); len(got) != 1 || got[0] != 2 {
		t.Errorf("ids = %v, want [2]", got)
	}
}

func TestTodoMySQLRepository_Move(t *testing.T) {
	repo := NewTodoMySQLRepository(openTestDB(t))
	for _, title := range []string{"a", "b", "c", "d"} {
		if err := repo.Create(domain.Todo{Title: title, Status: domain.StatusOpen}); err != nil {
			t.Fatal(err)
		}
	}

	tests := []struct {
		name     string
		id       int
		beforeID int
		want     []int
	}{
		{"後ろのタスクを前へ", 4, 2, []int{1, 4, 2, 3}},
		{"前のタスクを後ろへ", 1, 3, []int{4, 2, 1, 3}},
		{"先頭へ", 3, 4, []int{3, 4, 2, 1}},
		{"末尾へ", 3, 0, []int{4, 2, 1, 3}},
		{"自分自身の直前は変化なし", 2, 2, []int{4, 2, 1, 3}},
	}
	for _, tt := range tests {
		if err := repo.Move(tt.id, tt.beforeID); err != nil {
			t.Fatalf("%s: %v", tt.name, err)
		}
		got := todoIDs(t, repo)
		if len(got) != len(tt.want) {
			t.Fatalf("%s: ids = %v, want %v", tt.name, got, tt.want)
		}
		for i := range got {
			if got[i] != tt.want[i] {
				t.Fatalf("%s: ids = %v, want %v", tt.name, got, tt.want)
			}
		}
	}

	// 移動後に追加したタスクは末尾に並ぶ
	if err := repo.Create(domain.Todo{Title: "e", Status: domain.StatusOpen}); err != nil {
		t.Fatal(err)
	}
	if got := todoIDs(t, repo); len(got) != 5 || got[4] != 5 {
		t.Errorf("ids = %v, want 5 at the end", got)
	}

	// 存在しないタスクはエラーになり、並び順は変わらない
	if err := repo.Move(99, 0); err == nil {
		t.Error("Move(99, 0) error = nil")
	}
	if err := repo.Move(1, 99); err == nil {
		t.Error("Move(1, 99) error = nil")
	}
	if got := todoIDs(t, repo); got[0] != 4 || got[1] != 2 || got[2] != 1 || got[3] != 3 || got[4] != 5 {
		t.Errorf("ids = %v, want [4 2 1 3 5]", got)
	}
}

func TestEpicMySQLRepository_CRUD(t *testing.T) {
	repo := NewEpicMySQLRepository(openTestDB(t))

	if err := repo.Create(domain.Epic{Title: "e1", SpaceID: 2}); err != nil {
		t.Fatal(err)
	}
	if err := repo.Create(domain.Epic{Title: "e2"}); err != nil {
		t.Fatal(err)
	}
	if err := repo.Update(1, domain.Epic{Title: "e1改", SpaceID: 0}); err != nil {
		t.Fatal(err)
	}
	if err := repo.Update(1, domain.Epic{Title: "e1改", SpaceID: 0}); err != nil {
		t.Fatalf("Update() with same values error = %v", err)
	}
	if err := repo.Update(99, domain.Epic{Title: "x"}); err == nil {
		t.Error("Update(99) error = nil")
	}

	epics, err := repo.FindAll()
	if err != nil {
		t.Fatal(err)
	}
	want := []domain.Epic{{ID: 1, Title: "e1改"}, {ID: 2, Title: "e2"}}
	if len(epics) != 2 || epics[0] != want[0] || epics[1] != want[1] {
		t.Fatalf("epics = %+v, want %+v", epics, want)
	}

	if err := repo.Delete(1); err != nil {
		t.Fatal(err)
	}
	if err := repo.Delete(1); err == nil {
		t.Error("Delete(1) twice error = nil")
	}
	epics, err = repo.FindAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(epics) != 1 || epics[0].ID != 2 {
		t.Errorf("epics = %+v, want only ID 2", epics)
	}
}

func TestSpaceMySQLRepository_CRUD(t *testing.T) {
	repo := NewSpaceMySQLRepository(openTestDB(t))

	spaces, err := repo.FindAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(spaces) != 0 {
		t.Fatalf("spaces = %+v, want empty", spaces)
	}

	for _, title := range []string{"s1", "s2"} {
		if err := repo.Create(domain.Space{Title: title}); err != nil {
			t.Fatal(err)
		}
	}
	if err := repo.Delete(1); err != nil {
		t.Fatal(err)
	}
	if err := repo.Delete(1); err == nil {
		t.Error("Delete(1) twice error = nil")
	}

	spaces, err = repo.FindAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(spaces) != 1 || spaces[0] != (domain.Space{ID: 2, Title: "s2"}) {
		t.Errorf("spaces = %+v, want [{2 s2}]", spaces)
	}
}

func todoIDs(t *testing.T, repo TodoRepository) []int {
	t.Helper()
	todos, err := repo.FindAll()
	if err != nil {
		t.Fatal(err)
	}
	ids := []int{}
	for _, todo := range todos {
		ids = append(ids, todo.ID)
	}
	return ids
}
