package repository

import (
	"os"
	"testing"

	"todo-api/internal/domain"
)

// SpaceID 列がない旧形式の CSV はスペース未割り当てとして読み込め、更新すると列が追加される
func TestEpicRepository_SpaceIDAndUpdate(t *testing.T) {
	chdirTemp(t)
	legacy := "ID,Title\n1,old\n"
	if err := os.WriteFile(epicCSVFilePath, []byte(legacy), 0o644); err != nil {
		t.Fatal(err)
	}
	repo := NewEpicRepository()

	if err := repo.Create(domain.Epic{Title: "new", SpaceID: 2}); err != nil {
		t.Fatal(err)
	}
	if err := repo.Update(1, domain.Epic{Title: "renamed", SpaceID: 5}); err != nil {
		t.Fatal(err)
	}
	if err := repo.Update(99, domain.Epic{Title: "x"}); err == nil {
		t.Error("Update of missing epic: want error")
	}

	epics, err := repo.FindAll()
	if err != nil {
		t.Fatal(err)
	}
	want := []domain.Epic{{ID: 1, Title: "renamed", SpaceID: 5}, {ID: 2, Title: "new", SpaceID: 2}}
	if len(epics) != len(want) || epics[0] != want[0] || epics[1] != want[1] {
		t.Fatalf("epics = %+v, want %+v", epics, want)
	}

	data, err := os.ReadFile(epicCSVFilePath)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "ID,Title,SpaceID\n1,renamed,5\n2,new,2\n" {
		t.Errorf("csv = %q", data)
	}
}

func TestSpaceRepository_CRUD(t *testing.T) {
	chdirTemp(t)
	repo := NewSpaceRepository()

	spaces, err := repo.FindAll()
	if err != nil || len(spaces) != 0 {
		t.Fatalf("FindAll on missing file = %+v, %v", spaces, err)
	}

	for _, title := range []string{"A", "B, カンマ入り"} {
		if err := repo.Create(domain.Space{Title: title}); err != nil {
			t.Fatal(err)
		}
	}
	if err := repo.Delete(1); err != nil {
		t.Fatal(err)
	}
	if err := repo.Delete(1); err == nil {
		t.Error("Delete of missing space: want error")
	}
	if err := repo.Create(domain.Space{Title: "C"}); err != nil {
		t.Fatal(err)
	}

	spaces, err = repo.FindAll()
	if err != nil {
		t.Fatal(err)
	}
	want := []domain.Space{{ID: 2, Title: "B, カンマ入り"}, {ID: 3, Title: "C"}}
	if len(spaces) != len(want) || spaces[0] != want[0] || spaces[1] != want[1] {
		t.Fatalf("spaces = %+v, want %+v", spaces, want)
	}
}
