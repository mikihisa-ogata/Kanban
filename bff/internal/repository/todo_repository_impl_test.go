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
	want := "ID,Title,Done,Deadline,Status,EpicID,Description\n1,old,false,2026-10-01,Open,0,\n2,new,false,2026-10-02,Open,0,説明\n"
	if string(data) != want {
		t.Errorf("csv = %q, want %q", data, want)
	}
}
