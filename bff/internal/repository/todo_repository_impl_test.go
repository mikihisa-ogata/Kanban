package repository

import (
	"os"
	"testing"

	"todo-api/internal/domain"
)

// chdirTemp はカレントディレクトリを一時ディレクトリに切り替え、終了時に元へ戻す
func chdirTemp(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(wd); err != nil {
			t.Fatal(err)
		}
	})
	return dir
}

func TestTodoRepository_DescriptionRoundTrip(t *testing.T) {
	chdirTemp(t)
	repo := NewTodoRepository()

	description := "1行目, カンマ入り\n2行目 \"引用符\""
	if err := repo.Create(domain.Todo{Title: "a", Deadline: "2026-10-01", Status: domain.StatusOpen, Description: description}); err != nil {
		t.Fatal(err)
	}
	if err := repo.Create(domain.Todo{Title: "b", Deadline: "2026-10-02", Status: domain.StatusOpen}); err != nil {
		t.Fatal(err)
	}

	todos, err := repo.FindAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(todos) != 2 {
		t.Fatalf("len = %d, want 2", len(todos))
	}
	if todos[0].Description != description {
		t.Errorf("Description = %q, want %q", todos[0].Description, description)
	}
	if todos[1].Description != "" {
		t.Errorf("Description = %q, want empty", todos[1].Description)
	}

	updated := todos[0]
	updated.Description = "更新後"
	if err := repo.Update(updated.ID, updated); err != nil {
		t.Fatal(err)
	}
	todos, err = repo.FindAll()
	if err != nil {
		t.Fatal(err)
	}
	if todos[0].Description != "更新後" {
		t.Errorf("Description = %q, want %q", todos[0].Description, "更新後")
	}
}

// Description 列がない旧形式の CSV も読み込め、書き戻すと列が追加される
func TestTodoRepository_ReadsLegacyCSVWithoutDescription(t *testing.T) {
	chdirTemp(t)
	legacy := "ID,Title,Done,Deadline,Status,EpicID\n1,old,false,2026-10-01,Open,0\n"
	if err := os.WriteFile(csvFilePath, []byte(legacy), 0o644); err != nil {
		t.Fatal(err)
	}
	repo := NewTodoRepository()

	todos, err := repo.FindAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(todos) != 1 || todos[0].Title != "old" || todos[0].Description != "" {
		t.Fatalf("todos = %+v", todos)
	}

	if err := repo.Create(domain.Todo{Title: "new", Deadline: "2026-10-02", Status: domain.StatusOpen, Description: "説明"}); err != nil {
		t.Fatal(err)
	}
	todos, err = repo.FindAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(todos) != 2 || todos[0].Title != "old" || todos[1].Description != "説明" {
		t.Fatalf("todos = %+v", todos)
	}

	data, err := os.ReadFile(csvFilePath)
	if err != nil {
		t.Fatal(err)
	}
	want := "ID,Title,Done,Deadline,Status,EpicID,Description,SpaceID\n1,old,false,2026-10-01,Open,0,,0\n2,new,false,2026-10-02,Open,0,説明,0\n"
	if string(data) != want {
		t.Errorf("csv = %q, want %q", data, want)
	}
}

// SpaceID 列がない旧形式の CSV はスペース未割り当てとして読み込め、書き戻すと列が追加される
func TestTodoRepository_SpaceID(t *testing.T) {
	chdirTemp(t)
	legacy := "ID,Title,Done,Deadline,Status,EpicID,Description\n1,old,false,,Open,0,説明\n"
	if err := os.WriteFile(csvFilePath, []byte(legacy), 0o644); err != nil {
		t.Fatal(err)
	}
	repo := NewTodoRepository()

	if err := repo.Create(domain.Todo{Title: "new", Status: domain.StatusOpen, SpaceID: 3}); err != nil {
		t.Fatal(err)
	}
	todos, err := repo.FindAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(todos) != 2 || todos[0].SpaceID != 0 || todos[0].Description != "説明" || todos[1].SpaceID != 3 {
		t.Fatalf("todos = %+v", todos)
	}

	data, err := os.ReadFile(csvFilePath)
	if err != nil {
		t.Fatal(err)
	}
	want := "ID,Title,Done,Deadline,Status,EpicID,Description,SpaceID\n1,old,false,,Open,0,説明,0\n2,new,false,,Open,0,,3\n"
	if string(data) != want {
		t.Errorf("csv = %q, want %q", data, want)
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

func TestTodoRepository_Move(t *testing.T) {
	chdirTemp(t)
	repo := NewTodoRepository()
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

	// 移動しても内容は変わらない
	todos, err := repo.FindAll()
	if err != nil {
		t.Fatal(err)
	}
	if todos[0].ID != 4 || todos[0].Title != "d" {
		t.Errorf("todo = %+v", todos[0])
	}

	// 存在しないタスクはエラーになり、並び順は変わらない
	if err := repo.Move(99, 0); err == nil {
		t.Error("Move(99, 0) error = nil")
	}
	if err := repo.Move(1, 99); err == nil {
		t.Error("Move(1, 99) error = nil")
	}
	if got := todoIDs(t, repo); got[0] != 4 || got[1] != 2 || got[2] != 1 || got[3] != 3 {
		t.Errorf("ids = %v, want [4 2 1 3]", got)
	}
}
