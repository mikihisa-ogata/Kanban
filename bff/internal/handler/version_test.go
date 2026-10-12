package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"todo-api/internal/domain"
	"todo-api/internal/service"
)

// setupVersionRouter はスペース・課題・バージョンの API を持つルーターを返す
func setupVersionRouter(t *testing.T) *gin.Engine {
	t.Helper()
	repos := newTestRepos(t)
	r := newTodoRouter(repos)
	sh := NewSpaceHandler(service.NewSpaceService(repos.space, repos.epic, repos.todo, repos.history, repos.status))
	r.POST("/spaces", sh.CreateSpace)
	r.DELETE("/spaces/:id", sh.DeleteSpace)
	h := NewVersionHandler(service.NewVersionService(repos.version, repos.todo, repos.space))
	r.GET("/versions", h.GetVersions)
	r.POST("/spaces/:id/versions", h.CreateVersion)
	r.PUT("/versions/:id", h.UpdateVersion)
	r.POST("/versions/:id/release", h.ReleaseVersion)
	r.DELETE("/versions/:id", h.DeleteVersion)
	r.PUT("/todos/:id/fix-versions", h.SetTodoVersions)
	return r
}

func createVersion(t *testing.T, r *gin.Engine, spaceID int, body string) domain.Version {
	t.Helper()
	w := doRequest(t, r, http.MethodPost, fmt.Sprintf("/spaces/%d/versions", spaceID), body)
	if w.Code != http.StatusCreated {
		t.Fatalf("POST versions status = %d, body = %s", w.Code, w.Body)
	}
	var res struct{ Version domain.Version }
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		t.Fatal(err)
	}
	return res.Version
}

func findVersion(t *testing.T, r *gin.Engine, id int) domain.Version {
	t.Helper()
	for _, v := range getJSON[[]domain.Version](t, r, "/versions") {
		if v.ID == id {
			return v
		}
	}
	t.Fatalf("version %d not found", id)
	return domain.Version{}
}

// fixVersionsOf は課題（ID）の修正バージョンを返す
func fixVersionsOf(t *testing.T, r *gin.Engine, todoID int) []int {
	t.Helper()
	for _, todo := range getTodos(t, r) {
		if todo.ID == todoID {
			return todo.FixVersionIDs
		}
	}
	t.Fatalf("todo %d not found", todoID)
	return nil
}

func setFixVersions(t *testing.T, r *gin.Engine, todoID int, ids string, want int) {
	t.Helper()
	mustStatus(t, r, http.MethodPut, fmt.Sprintf("/todos/%d/fix-versions", todoID), fmt.Sprintf(`{"versionIds":%s}`, ids), want)
}

func TestVersion_CreateAssignRelease(t *testing.T) {
	r := setupVersionRouter(t)
	mustStatus(t, r, http.MethodPost, "/spaces", `{"Title":"開発","Key":"DEV"}`, http.StatusCreated)
	mustStatus(t, r, http.MethodPost, "/spaces", `{"Title":"運用","Key":"OPS"}`, http.StatusCreated)

	v1 := createVersion(t, r, 1, `{"name":" v1.0 ","description":"最初のリリース","startDate":"2026-10-01","releaseDate":"2026-11-01"}`)
	if v1.Name != "v1.0" || v1.Status != domain.VersionUnreleased || v1.StartDate != "2026-10-01" || v1.ReleaseDate != "2026-11-01" || v1.Description != "最初のリリース" {
		t.Errorf("created = %+v", v1)
	}
	v2 := createVersion(t, r, 1, `{"name":"v1.1"}`)
	if v2.StartDate != "" || v2.ReleaseDate != "" {
		t.Errorf("dates should be empty: %+v", v2)
	}
	ops := createVersion(t, r, 2, `{"name":"v1.0"}`) // 別のスペースなら同じ名前でよい
	// 同じスペースの同じ名前（大文字・小文字は区別しない）は 409。空の名前・日付の誤り・未割り当て・存在しないスペース
	mustStatus(t, r, http.MethodPost, "/spaces/1/versions", `{"name":"V1.0"}`, http.StatusConflict)
	mustStatus(t, r, http.MethodPost, "/spaces/1/versions", `{"name":"  "}`, http.StatusBadRequest)
	mustStatus(t, r, http.MethodPost, "/spaces/1/versions", `{"name":"x","startDate":"2026-11-02","releaseDate":"2026-11-01"}`, http.StatusBadRequest)
	mustStatus(t, r, http.MethodPost, "/spaces/0/versions", `{"name":"x"}`, http.StatusBadRequest)
	mustStatus(t, r, http.MethodPost, "/spaces/99/versions", `{"name":"x"}`, http.StatusNotFound)

	// 課題: A・B・C（DEV）、X（OPS）
	for _, body := range []string{`{"title":"A","spaceId":1}`, `{"title":"B","spaceId":1}`, `{"title":"C","spaceId":1}`, `{"title":"X","spaceId":2}`} {
		mustStatus(t, r, http.MethodPost, "/todos", body, http.StatusCreated)
	}
	if got := fixVersionsOf(t, r, 1); len(got) != 0 {
		t.Errorf("new todo versions = %v", got)
	}
	// 複数のバージョンを付けられる（重複は 1 つにまとめ、ID 順）
	setFixVersions(t, r, 1, fmt.Sprintf("[%d,%d,%d]", v2.ID, v1.ID, v2.ID), http.StatusOK)
	if got := fixVersionsOf(t, r, 1); !reflect.DeepEqual(got, []int{v1.ID, v2.ID}) {
		t.Errorf("todo 1 versions = %v", got)
	}
	if todo := getJSON[domain.Todo](t, r, "/issues/DEV-1"); !reflect.DeepEqual(todo.FixVersionIDs, []int{v1.ID, v2.ID}) {
		t.Errorf("GET /issues versions = %v", todo.FixVersionIDs)
	}
	setFixVersions(t, r, 2, fmt.Sprintf("[%d]", v1.ID), http.StatusOK)
	setFixVersions(t, r, 3, fmt.Sprintf("[%d]", v1.ID), http.StatusOK)
	// 別スペースのバージョンは 400、存在しないバージョン・課題は 404。空の配列で外せる
	setFixVersions(t, r, 4, fmt.Sprintf("[%d]", v1.ID), http.StatusBadRequest)
	setFixVersions(t, r, 1, "[99]", http.StatusNotFound)
	mustStatus(t, r, http.MethodPut, "/todos/99/fix-versions", `{"versionIds":[]}`, http.StatusNotFound)
	setFixVersions(t, r, 4, fmt.Sprintf("[%d]", ops.ID), http.StatusOK)
	setFixVersions(t, r, 4, "[]", http.StatusOK)
	if got := fixVersionsOf(t, r, 4); len(got) != 0 {
		t.Errorf("todo 4 versions after clear = %v", got)
	}

	// B を完了にしてから v1.0 をリリースする。移し先は同じスペースの別の未リリースのバージョンだけ
	mustStatus(t, r, http.MethodPut, "/todos/2", `{"title":"B","status":"Closed","spaceId":1}`, http.StatusOK)
	release := fmt.Sprintf("/versions/%d/release", v1.ID)
	mustStatus(t, r, http.MethodPost, release, fmt.Sprintf(`{"moveTo":%d}`, ops.ID), http.StatusBadRequest)
	mustStatus(t, r, http.MethodPost, release, fmt.Sprintf(`{"moveTo":%d}`, v1.ID), http.StatusBadRequest)
	mustStatus(t, r, http.MethodPost, release, `{"releaseDate":"2026-09-01"}`, http.StatusBadRequest) // 開始日より前
	mustStatus(t, r, http.MethodPost, release, fmt.Sprintf(`{"moveTo":%d}`, v2.ID), http.StatusOK)
	released := findVersion(t, r, v1.ID)
	if released.Status != domain.VersionReleased || released.ReleaseDate != "2026-11-01" {
		t.Errorf("released = %+v", released)
	}
	// 未完了の A・C は v1.1 へ付け替え（A は元から v1.1 にもある）、完了した B は v1.0 のまま
	for todoID, want := range map[int][]int{1: {v2.ID}, 2: {v1.ID}, 3: {v2.ID}} {
		if got := fixVersionsOf(t, r, todoID); !reflect.DeepEqual(got, want) {
			t.Errorf("todo %d versions = %v, want %v", todoID, got, want)
		}
	}
	mustStatus(t, r, http.MethodPost, release, `{}`, http.StatusBadRequest) // リリース済み

	// リリース日を省略して日付もなければ今日。移し先なしなら未完了の課題もこのバージョンのまま（Jira の「無視してリリース」）
	mustStatus(t, r, http.MethodPost, fmt.Sprintf("/versions/%d/release", v2.ID), `{}`, http.StatusOK)
	if got := findVersion(t, r, v2.ID); got.ReleaseDate != time.Now().Format("2006-01-02") {
		t.Errorf("default release date = %s", got.ReleaseDate)
	}
	if got := fixVersionsOf(t, r, 1); !reflect.DeepEqual(got, []int{v2.ID}) {
		t.Errorf("todo 1 versions after release without moveTo = %v", got)
	}

	// 編集で状態を変えられる（アーカイブ）。アーカイブしたバージョンは新しく付けられないが、付いたままなら残せる
	mustStatus(t, r, http.MethodPut, fmt.Sprintf("/versions/%d", v1.ID), `{"name":"v1.0","description":"","startDate":"2026-10-01","releaseDate":"2026-11-01","status":"archived"}`, http.StatusOK)
	if got := findVersion(t, r, v1.ID); got.Status != domain.VersionArchived || got.Description != "" {
		t.Errorf("archived = %+v", got)
	}
	setFixVersions(t, r, 3, fmt.Sprintf("[%d]", v1.ID), http.StatusBadRequest)
	setFixVersions(t, r, 2, fmt.Sprintf("[%d]", v1.ID), http.StatusOK)
	mustStatus(t, r, http.MethodPut, fmt.Sprintf("/versions/%d", v1.ID), `{"name":"v1.0","status":"done"}`, http.StatusBadRequest)
	mustStatus(t, r, http.MethodPut, fmt.Sprintf("/versions/%d", v1.ID), `{"name":"v1.1","status":"archived"}`, http.StatusConflict)
	mustStatus(t, r, http.MethodPut, "/versions/99", `{"name":"x","status":"unreleased"}`, http.StatusNotFound)
}

func TestVersion_DeleteAndSpaceChange(t *testing.T) {
	r := setupVersionRouter(t)
	mustStatus(t, r, http.MethodPost, "/spaces", `{"Title":"開発","Key":"DEV"}`, http.StatusCreated)
	mustStatus(t, r, http.MethodPost, "/spaces", `{"Title":"運用","Key":"OPS"}`, http.StatusCreated)
	v1 := createVersion(t, r, 1, `{"name":"v1"}`)
	v2 := createVersion(t, r, 1, `{"name":"v2"}`)
	mustStatus(t, r, http.MethodPost, "/todos", `{"title":"A","spaceId":1}`, http.StatusCreated)
	mustStatus(t, r, http.MethodPost, "/todos", `{"title":"B","spaceId":1}`, http.StatusCreated)
	setFixVersions(t, r, 1, fmt.Sprintf("[%d,%d]", v1.ID, v2.ID), http.StatusOK)
	setFixVersions(t, r, 2, fmt.Sprintf("[%d]", v1.ID), http.StatusOK)

	// 削除すると課題からも外れる
	mustStatus(t, r, http.MethodDelete, fmt.Sprintf("/versions/%d", v1.ID), "", http.StatusOK)
	if got := fixVersionsOf(t, r, 1); !reflect.DeepEqual(got, []int{v2.ID}) {
		t.Errorf("todo 1 versions after delete = %v", got)
	}
	mustStatus(t, r, http.MethodDelete, fmt.Sprintf("/versions/%d", v1.ID), "", http.StatusNotFound)

	// スペースが変わらない更新では残り、スペースを移すと外れる
	mustStatus(t, r, http.MethodPut, "/todos/1", `{"title":"A2","spaceId":1}`, http.StatusOK)
	if got := fixVersionsOf(t, r, 1); !reflect.DeepEqual(got, []int{v2.ID}) {
		t.Errorf("todo 1 versions after update = %v", got)
	}
	mustStatus(t, r, http.MethodPut, "/todos/1", `{"title":"A2","spaceId":2}`, http.StatusOK)
	if got := fixVersionsOf(t, r, 1); len(got) != 0 {
		t.Errorf("todo 1 versions after space change = %v", got)
	}

	// 課題を削除しても、スペースを削除してもバージョンの付与が残らない
	setFixVersions(t, r, 2, fmt.Sprintf("[%d]", v2.ID), http.StatusOK)
	mustStatus(t, r, http.MethodDelete, "/todos/2", "", http.StatusOK)
	mustStatus(t, r, http.MethodPost, "/todos", `{"title":"C","spaceId":1}`, http.StatusCreated)
	setFixVersions(t, r, 3, fmt.Sprintf("[%d]", v2.ID), http.StatusOK)
	mustStatus(t, r, http.MethodDelete, "/spaces/1", "", http.StatusOK)
	if versions := getJSON[[]domain.Version](t, r, "/versions"); len(versions) != 0 {
		t.Errorf("versions after space delete = %+v", versions)
	}
	if got := fixVersionsOf(t, r, 3); len(got) != 0 {
		t.Errorf("todo 3 versions after space delete = %v", got)
	}
}
