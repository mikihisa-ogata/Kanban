package repository

import (
	"database/sql"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

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
	if _, err := repo.Create(domain.Todo{Title: "a", Deadline: "2026-10-01", Status: domain.StatusOpen, Description: description, EpicID: 2, SpaceID: 3}); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.Create(domain.Todo{Title: "b", Status: domain.StatusWaiting, IssueType: domain.IssueTypeBug, Priority: domain.PriorityHighest}); err != nil {
		t.Fatal(err)
	}

	todos, err := repo.FindAll()
	if err != nil {
		t.Fatal(err)
	}
	// スペース 3 は spaces にないのでキーの接頭辞がなく、課題キーは空になる。課題タイプを省くと Task、優先度を省くと Medium になる
	want := []domain.Todo{
		{ID: 1, Title: "a", Deadline: "2026-10-01", Status: domain.StatusOpen, IssueType: domain.IssueTypeTask, Priority: domain.PriorityMedium, Description: description, EpicID: 2, SpaceID: 3, Number: 1},
		{ID: 2, Title: "b", Status: domain.StatusWaiting, IssueType: domain.IssueTypeBug, Priority: domain.PriorityHighest, Number: 1, Key: "TASK-1"},
	}
	if len(todos) != len(want) {
		t.Fatalf("todos = %+v, want %+v", todos, want)
	}
	for i := range want {
		if !reflect.DeepEqual(todoWithoutTimes(todos[i]), want[i]) {
			t.Errorf("todos[%d] = %+v, want %+v", i, todos[i], want[i])
		}
	}

	updated := domain.Todo{Title: "a2", Done: true, Status: domain.StatusClosed, IssueType: domain.IssueTypeStory, Priority: domain.PriorityLow, Description: "更新後"}
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
	// 未割り当てへ移したので、未割り当ての続きの番号になる（2 回目の更新ではスペースが同じなので変わらない）
	updated.ID, updated.Number, updated.Key = 1, 2, "TASK-2"
	if !reflect.DeepEqual(todoWithoutTimes(todos[0]), updated) {
		t.Errorf("todos[0] = %+v, want %+v", todos[0], updated)
	}

	if err := repo.Update(99, updated); err == nil {
		t.Error("Update(99) error = nil")
	}
	if _, err := repo.Create(domain.Todo{Title: "c", Deadline: "not a date", Status: domain.StatusOpen}); err == nil {
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
		if _, err := repo.Create(domain.Todo{Title: title, Status: domain.StatusOpen}); err != nil {
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
	if _, err := repo.Create(domain.Todo{Title: "e", Status: domain.StatusOpen}); err != nil {
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
	if len(epics) != 2 || epicWithoutTimes(epics[0]) != want[0] || epicWithoutTimes(epics[1]) != want[1] {
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

// todoWithoutTimes は日時以外の内容を比べるため、作成日時・更新日時を空にしたタスクを返す
func todoWithoutTimes(todo domain.Todo) domain.Todo {
	todo.CreatedAt, todo.UpdatedAt = time.Time{}, time.Time{}
	// ラベル・修正バージョンのないタスクは空の配列で返るので、期待値（nil）と比べられるようにする
	if len(todo.Labels) == 0 {
		todo.Labels = nil
	}
	if len(todo.FixVersionIDs) == 0 {
		todo.FixVersionIDs = nil
	}
	return todo
}

// epicWithoutTimes は日時以外の内容を比べるため、作成日時・更新日時を空にしたエピックを返す
func epicWithoutTimes(epic domain.Epic) domain.Epic {
	epic.CreatedAt, epic.UpdatedAt = time.Time{}, time.Time{}
	return epic
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

func TestTodoMySQLRepository_Timestamps(t *testing.T) {
	db := openTestDB(t)
	repo := NewTodoMySQLRepository(db)

	before := currentTime()
	for _, title := range []string{"a", "b"} {
		if _, err := repo.Create(domain.Todo{Title: title, Status: domain.StatusOpen}); err != nil {
			t.Fatal(err)
		}
	}
	after := currentTime()
	todo := findTodo(t, repo, 1)
	if todo.CreatedAt.Before(before) || todo.CreatedAt.After(after) || !todo.UpdatedAt.Equal(todo.CreatedAt) {
		t.Fatalf("created_at = %v, updated_at = %v, want both between %v and %v", todo.CreatedAt, todo.UpdatedAt, before, after)
	}

	// 日時を過去にずらし、更新・並び替えで変わるかを確かめる
	old := time.Date(2020, 1, 2, 3, 4, 5, 0, time.UTC)
	if _, err := db.Exec(`UPDATE todos SET created_at = ?, updated_at = ?`, old, old); err != nil {
		t.Fatal(err)
	}

	// 並び替えでは更新日時は変わらない
	if err := repo.Move(1, 0); err != nil {
		t.Fatal(err)
	}
	if todo := findTodo(t, repo, 1); !todo.UpdatedAt.Equal(old) {
		t.Errorf("updated_at after Move = %v, want %v", todo.UpdatedAt, old)
	}

	// 更新すると更新日時だけが現在日時になる
	before = currentTime()
	if err := repo.Update(1, domain.Todo{Title: "a2", Status: domain.StatusClosed}); err != nil {
		t.Fatal(err)
	}
	todo = findTodo(t, repo, 1)
	if !todo.CreatedAt.Equal(old) {
		t.Errorf("created_at after Update = %v, want %v", todo.CreatedAt, old)
	}
	if todo.UpdatedAt.Before(before) {
		t.Errorf("updated_at after Update = %v, want >= %v", todo.UpdatedAt, before)
	}
	if other := findTodo(t, repo, 2); !other.UpdatedAt.Equal(old) {
		t.Errorf("updated_at of other todo = %v, want %v", other.UpdatedAt, old)
	}
}

func TestEpicMySQLRepository_Timestamps(t *testing.T) {
	db := openTestDB(t)
	repo := NewEpicMySQLRepository(db)

	before := currentTime()
	if err := repo.Create(domain.Epic{Title: "e"}); err != nil {
		t.Fatal(err)
	}
	after := currentTime()
	epics, err := repo.FindAll()
	if err != nil {
		t.Fatal(err)
	}
	if epics[0].CreatedAt.Before(before) || epics[0].CreatedAt.After(after) || !epics[0].UpdatedAt.Equal(epics[0].CreatedAt) {
		t.Fatalf("created_at = %v, updated_at = %v, want both between %v and %v", epics[0].CreatedAt, epics[0].UpdatedAt, before, after)
	}

	old := time.Date(2020, 1, 2, 3, 4, 5, 0, time.UTC)
	if _, err := db.Exec(`UPDATE epics SET created_at = ?, updated_at = ?`, old, old); err != nil {
		t.Fatal(err)
	}
	before = currentTime()
	if err := repo.Update(1, domain.Epic{Title: "e2"}); err != nil {
		t.Fatal(err)
	}
	epics, err = repo.FindAll()
	if err != nil {
		t.Fatal(err)
	}
	if !epics[0].CreatedAt.Equal(old) || epics[0].UpdatedAt.Before(before) {
		t.Errorf("created_at = %v, updated_at = %v, want created_at %v and updated_at >= %v", epics[0].CreatedAt, epics[0].UpdatedAt, old, before)
	}
}

func findTodo(t *testing.T, repo TodoRepository, id int) domain.Todo {
	t.Helper()
	todos, err := repo.FindAll()
	if err != nil {
		t.Fatal(err)
	}
	for _, todo := range todos {
		if todo.ID == id {
			return todo
		}
	}
	t.Fatalf("todo %d not found", id)
	return domain.Todo{}
}

func TestTodoMySQLRepository_IssueNumbers(t *testing.T) {
	db := openTestDB(t)
	repo := NewTodoMySQLRepository(db)
	spaceRepo := NewSpaceMySQLRepository(db)
	for _, space := range []domain.Space{{Title: "Kanban", Key: "KAN"}, {Title: "Money", Key: "MON"}} {
		if err := spaceRepo.Create(space); err != nil {
			t.Fatal(err)
		}
	}

	// スペースごとに 1 から採番する
	for _, todo := range []domain.Todo{
		{Title: "k1", SpaceID: 1}, {Title: "m1", SpaceID: 2}, {Title: "k2", SpaceID: 1}, {Title: "u1"},
	} {
		todo.Status = domain.StatusOpen
		if _, err := repo.Create(todo); err != nil {
			t.Fatal(err)
		}
	}
	if got := todoKeys(t, repo); got != "1:KAN-1 2:MON-1 3:KAN-2 4:TASK-1" {
		t.Fatalf("keys = %s", got)
	}

	// 削除した番号は再利用しない
	if err := repo.Delete(3); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.Create(domain.Todo{Title: "k3", Status: domain.StatusOpen, SpaceID: 1}); err != nil {
		t.Fatal(err)
	}
	if got := todoKeys(t, repo); got != "1:KAN-1 2:MON-1 4:TASK-1 5:KAN-3" {
		t.Fatalf("keys after delete = %s", got)
	}

	// スペースを移すと移動先で採番し直す。スペースが同じ更新では番号は変わらない
	if err := repo.Update(1, domain.Todo{Title: "k1", Status: domain.StatusOpen, SpaceID: 2}); err != nil {
		t.Fatal(err)
	}
	if err := repo.Update(1, domain.Todo{Title: "k1 改", Status: domain.StatusClosed, SpaceID: 2}); err != nil {
		t.Fatal(err)
	}
	if err := repo.Update(2, domain.Todo{Title: "m1", Status: domain.StatusOpen}); err != nil {
		t.Fatal(err)
	}
	if got := todoKeys(t, repo); got != "1:MON-2 2:TASK-2 4:TASK-1 5:KAN-3" {
		t.Fatalf("keys after move = %s", got)
	}

	// 並び替えでは番号は変わらない
	if err := repo.Move(5, 1); err != nil {
		t.Fatal(err)
	}
	if got := todoKeys(t, repo); got != "5:KAN-3 1:MON-2 2:TASK-2 4:TASK-1" {
		t.Fatalf("keys after Move = %s", got)
	}
}

func TestSpaceMySQLRepository_Keys(t *testing.T) {
	repo := NewSpaceMySQLRepository(openTestDB(t))
	if err := repo.Create(domain.Space{Title: "Kanban", Key: "KAN"}); err != nil {
		t.Fatal(err)
	}
	// キーなしは複数作れる（CSV からの取り込みなど。次の Migrate でキーが付く）
	for i := 0; i < 2; i++ {
		if err := repo.Create(domain.Space{Title: "no key"}); err != nil {
			t.Fatal(err)
		}
	}
	// 同じキーは DB の一意制約で弾く
	if err := repo.Create(domain.Space{Title: "dup", Key: "KAN"}); err == nil {
		t.Error("Create() with duplicate key error = nil")
	}
	spaces, err := repo.FindAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(spaces) != 3 || spaces[0].Key != "KAN" || spaces[1].Key != "" || spaces[2].Key != "" {
		t.Errorf("spaces = %+v", spaces)
	}

	if err := repo.Update(2, domain.Space{Title: "renamed", Key: "REN"}); err != nil {
		t.Fatal(err)
	}
	if err := repo.Update(3, domain.Space{Title: "dup", Key: "KAN"}); err == nil {
		t.Error("Update() with duplicate key error = nil")
	}
	if err := repo.Update(99, domain.Space{Title: "x", Key: "XYZ"}); err == nil {
		t.Error("Update(99) error = nil")
	}
	spaces, err = repo.FindAll()
	if err != nil {
		t.Fatal(err)
	}
	if spaces[1] != (domain.Space{ID: 2, Title: "renamed", Key: "REN"}) || spaces[2].Title != "no key" {
		t.Errorf("spaces = %+v", spaces)
	}
}

// todoKeys は表示順のタスクを「ID:課題キー」の空白区切りで返す
func todoKeys(t *testing.T, repo TodoRepository) string {
	t.Helper()
	todos, err := repo.FindAll()
	if err != nil {
		t.Fatal(err)
	}
	keys := []string{}
	for _, todo := range todos {
		keys = append(keys, fmt.Sprintf("%d:%s", todo.ID, todo.Key))
	}
	return strings.Join(keys, " ")
}

func TestTodoMySQLRepository_CreateReturnsIDAndFindByID(t *testing.T) {
	db := openTestDB(t)
	repo := NewTodoMySQLRepository(db)
	if err := NewSpaceMySQLRepository(db).Create(domain.Space{Title: "Kanban", Key: "KAN"}); err != nil {
		t.Fatal(err)
	}

	for i, want := range []int{1, 2} {
		id, err := repo.Create(domain.Todo{Title: fmt.Sprintf("t%d", i), Status: domain.StatusOpen, SpaceID: 1})
		if err != nil {
			t.Fatal(err)
		}
		if id != want {
			t.Errorf("Create() id = %d, want %d", id, want)
		}
	}

	todo, err := repo.FindByID(2)
	if err != nil {
		t.Fatal(err)
	}
	if todo.ID != 2 || todo.Title != "t1" || todo.Key != "KAN-2" || todo.CreatedAt.IsZero() {
		t.Errorf("FindByID(2) = %+v", todo)
	}
	if _, err := repo.FindByID(99); err == nil {
		t.Error("FindByID(99) error = nil")
	}
}

func TestTodoMySQLRepository_FindByKey(t *testing.T) {
	db := openTestDB(t)
	repo := NewTodoMySQLRepository(db)
	if err := NewSpaceMySQLRepository(db).Create(domain.Space{Title: "Kanban", Key: "KAN"}); err != nil {
		t.Fatal(err)
	}
	for _, todo := range []domain.Todo{
		{Title: "kan1", Status: domain.StatusOpen, SpaceID: 1},
		{Title: "task1", Status: domain.StatusOpen},
		{Title: "kan2", Status: domain.StatusOpen, SpaceID: 1},
	} {
		if _, err := repo.Create(todo); err != nil {
			t.Fatal(err)
		}
	}

	tests := []struct {
		prefix    string
		number    int
		wantTitle string // 空文字は見つからないこと
	}{
		{"KAN", 1, "kan1"},
		{"KAN", 2, "kan2"},
		{domain.UnassignedKeyPrefix, 1, "task1"},
		{domain.UnassignedKeyPrefix, 2, ""},
		{"KAN", 3, ""},
		{"OTH", 1, ""},
	}
	for _, tt := range tests {
		todo, err := repo.FindByKey(tt.prefix, tt.number)
		if err != nil {
			t.Fatal(err)
		}
		if tt.wantTitle == "" {
			if todo != nil {
				t.Errorf("FindByKey(%s, %d) = %+v, want nil", tt.prefix, tt.number, todo)
			}
			continue
		}
		if todo == nil || todo.Title != tt.wantTitle || todo.Key != domain.IssueKey(tt.prefix, tt.number) {
			t.Errorf("FindByKey(%s, %d) = %+v, want %s", tt.prefix, tt.number, todo, tt.wantTitle)
		}
	}
}

func TestTodoMySQLRepository_Labels(t *testing.T) {
	repo := NewTodoMySQLRepository(openTestDB(t))

	id1, err := repo.Create(domain.Todo{Title: "a", Status: domain.StatusOpen, Labels: []string{"ui", "backend"}})
	if err != nil {
		t.Fatal(err)
	}
	// 大文字・小文字だけ違うラベルは既存のラベルとして扱う
	id2, err := repo.Create(domain.Todo{Title: "b", Status: domain.StatusOpen, Labels: []string{"UI"}})
	if err != nil {
		t.Fatal(err)
	}
	id3, err := repo.Create(domain.Todo{Title: "c", Status: domain.StatusOpen})
	if err != nil {
		t.Fatal(err)
	}

	labelsOf := func(id int) []string {
		t.Helper()
		todo, err := repo.FindByID(id)
		if err != nil {
			t.Fatal(err)
		}
		return todo.Labels
	}
	if got := labelsOf(id1); !reflect.DeepEqual(got, []string{"backend", "ui"}) {
		t.Errorf("labels of a = %q", got)
	}
	if got := labelsOf(id2); !reflect.DeepEqual(got, []string{"ui"}) {
		t.Errorf("labels of b = %q, want [ui]", got)
	}
	if got := labelsOf(id3); got == nil || len(got) != 0 {
		t.Errorf("labels of c = %#v, want empty slice", got)
	}
	todos, err := repo.FindAll()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(todos[0].Labels, []string{"backend", "ui"}) || !reflect.DeepEqual(todos[1].Labels, []string{"ui"}) || len(todos[2].Labels) != 0 {
		t.Errorf("FindAll labels = %q, %q, %q", todos[0].Labels, todos[1].Labels, todos[2].Labels)
	}
	if todo, err := repo.FindByKey(domain.UnassignedKeyPrefix, 1); err != nil || !reflect.DeepEqual(todo.Labels, []string{"backend", "ui"}) {
		t.Errorf("FindByKey labels = %+v, %v", todo, err)
	}

	// 更新ではラベルを置き換える
	if err := repo.Update(id1, domain.Todo{Title: "a", Status: domain.StatusOpen, Labels: []string{"api"}}); err != nil {
		t.Fatal(err)
	}
	if got := labelsOf(id1); !reflect.DeepEqual(got, []string{"api"}) {
		t.Errorf("labels of a after update = %q", got)
	}
	// 補完候補はどれかのタスクに付いているラベルだけ（backend はもう使われていない）
	if got, err := repo.FindLabels(); err != nil || !reflect.DeepEqual(got, []string{"api", "ui"}) {
		t.Errorf("FindLabels() = %q, %v", got, err)
	}

	// 削除するとそのタスクへの付与も消える
	if err := repo.Delete(id2); err != nil {
		t.Fatal(err)
	}
	if got, err := repo.FindLabels(); err != nil || !reflect.DeepEqual(got, []string{"api"}) {
		t.Errorf("FindLabels() after delete = %q, %v", got, err)
	}
}

// ひらがなとカタカナ、全角と半角、アクセントの有無は別のラベルとして扱う
func TestTodoMySQLRepository_LabelsDistinguishKanaAndWidth(t *testing.T) {
	repo := NewTodoMySQLRepository(openTestDB(t))
	names := []string{"テスト", "てすと", "ＵＩ", "ui", "café", "cafe"}
	for _, name := range names {
		if _, err := repo.Create(domain.Todo{Title: name, Status: domain.StatusOpen, Labels: []string{name}}); err != nil {
			t.Fatal(err)
		}
	}
	got, err := repo.FindLabels()
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != len(names) {
		t.Errorf("FindLabels() = %q, want %d distinct labels", got, len(names))
	}
}

func TestTodoMySQLRepository_StoryPoints(t *testing.T) {
	repo := NewTodoMySQLRepository(openTestDB(t))
	points := 2.5
	id1, err := repo.Create(domain.Todo{Title: "a", Status: domain.StatusOpen, StoryPoints: &points})
	if err != nil {
		t.Fatal(err)
	}
	id2, err := repo.Create(domain.Todo{Title: "b", Status: domain.StatusOpen})
	if err != nil {
		t.Fatal(err)
	}
	a, err := repo.FindByID(id1)
	if err != nil || a.StoryPoints == nil || *a.StoryPoints != 2.5 {
		t.Errorf("FindByID(a) StoryPoints = %v, %v", a.StoryPoints, err)
	}
	b, err := repo.FindByID(id2)
	if err != nil || b.StoryPoints != nil {
		t.Errorf("FindByID(b) StoryPoints = %v, %v, want nil", b.StoryPoints, err)
	}

	// 更新で未設定に戻せる
	if err := repo.Update(id1, domain.Todo{Title: "a", Status: domain.StatusOpen}); err != nil {
		t.Fatal(err)
	}
	if a, err = repo.FindByID(id1); err != nil || a.StoryPoints != nil {
		t.Errorf("after update StoryPoints = %v, %v, want nil", a.StoryPoints, err)
	}
}
