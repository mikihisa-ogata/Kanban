package repository

import (
	"os"
	"path/filepath"
	"testing"

	"todo-api/internal/domain"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// ID・並び順・全項目を保ったまま取り込み、以降に追加した行は続きの ID になる
func TestImportCSV(t *testing.T) {
	db := openTestDB(t)
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "todos.csv"),
		"ID,Title,Done,Deadline,Status,EpicID,Description,SpaceID\n"+
			"5,後ろに作ったタスク,true,2026-10-01,Closed,3,\"1行目, カンマ\n2行目\",2\n"+
			"2,前に作ったタスク,false,,InProgress,0,,0\n")
	writeFile(t, filepath.Join(dir, "epics.csv"), "ID,Title,SpaceID\n3,エピック,2\n")
	writeFile(t, filepath.Join(dir, "spaces.csv"), "ID,Title\n2,スペース\n")

	result, err := ImportCSV(db, dir)
	if err != nil {
		t.Fatal(err)
	}
	if result != (ImportResult{Todos: 2, Epics: 1, Spaces: 1}) {
		t.Errorf("result = %+v", result)
	}

	todoRepo := NewTodoMySQLRepository(db)
	todos, err := todoRepo.FindAll()
	if err != nil {
		t.Fatal(err)
	}
	want := []domain.Todo{
		{ID: 5, Title: "後ろに作ったタスク", Done: true, Deadline: "2026-10-01", Status: domain.StatusClosed, EpicID: 3, Description: "1行目, カンマ\n2行目", SpaceID: 2},
		{ID: 2, Title: "前に作ったタスク", Status: domain.StatusInProgress},
	}
	if len(todos) != len(want) {
		t.Fatalf("todos = %+v, want %+v", todos, want)
	}
	for i := range want {
		if todos[i] != want[i] {
			t.Errorf("todos[%d] = %+v, want %+v", i, todos[i], want[i])
		}
	}

	epics, err := NewEpicMySQLRepository(db).FindAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(epics) != 1 || epics[0] != (domain.Epic{ID: 3, Title: "エピック", SpaceID: 2}) {
		t.Errorf("epics = %+v", epics)
	}
	spaces, err := NewSpaceMySQLRepository(db).FindAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(spaces) != 1 || spaces[0] != (domain.Space{ID: 2, Title: "スペース"}) {
		t.Errorf("spaces = %+v", spaces)
	}

	// 取り込み後に追加したタスクは続きの ID で末尾に並ぶ
	if err := todoRepo.Create(domain.Todo{Title: "new", Status: domain.StatusOpen}); err != nil {
		t.Fatal(err)
	}
	if got := todoIDs(t, todoRepo); len(got) != 3 || got[0] != 5 || got[1] != 2 || got[2] != 6 {
		t.Errorf("ids = %v, want [5 2 6]", got)
	}

	// 二重に取り込まない
	if _, err := ImportCSV(db, dir); err == nil {
		t.Error("second ImportCSV() error = nil")
	}
	if got := todoIDs(t, todoRepo); len(got) != 3 {
		t.Errorf("ids after second import = %v", got)
	}

	// CSV は変更しない
	data, err := os.ReadFile(filepath.Join(dir, "spaces.csv"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "ID,Title\n2,スペース\n" {
		t.Errorf("spaces.csv = %q", data)
	}
}

// 列が足りない旧形式の CSV も取り込め、ファイルがなければ 0 件になる
func TestImportCSV_LegacyAndMissingFiles(t *testing.T) {
	db := openTestDB(t)
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "todos.csv"),
		"ID,Title,Done,Deadline,Status,EpicID\n1,old,false,2026-10-01,Open,0\n")
	writeFile(t, filepath.Join(dir, "epics.csv"), "ID,Title\n1,old epic\n")

	result, err := ImportCSV(db, dir)
	if err != nil {
		t.Fatal(err)
	}
	if result != (ImportResult{Todos: 1, Epics: 1, Spaces: 0}) {
		t.Errorf("result = %+v", result)
	}
	todos, err := NewTodoMySQLRepository(db).FindAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(todos) != 1 || todos[0] != (domain.Todo{ID: 1, Title: "old", Deadline: "2026-10-01", Status: domain.StatusOpen}) {
		t.Errorf("todos = %+v", todos)
	}
	epics, err := NewEpicMySQLRepository(db).FindAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(epics) != 1 || epics[0] != (domain.Epic{ID: 1, Title: "old epic"}) {
		t.Errorf("epics = %+v", epics)
	}
}

// 不正な行があれば何も取り込まない
func TestImportCSV_RollbackOnError(t *testing.T) {
	db := openTestDB(t)
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "spaces.csv"), "ID,Title\n1,s\n")
	writeFile(t, filepath.Join(dir, "todos.csv"),
		"ID,Title,Done,Deadline,Status,EpicID,Description,SpaceID\n1,ok,false,,Open,0,,0\n2,bad,false,2026/10/01,Open,0,,0\n")

	if _, err := ImportCSV(db, dir); err == nil {
		t.Fatal("ImportCSV() error = nil, want error for invalid deadline")
	}
	spaces, err := NewSpaceMySQLRepository(db).FindAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(spaces) != 0 {
		t.Errorf("spaces = %+v, want empty after rollback", spaces)
	}
}
