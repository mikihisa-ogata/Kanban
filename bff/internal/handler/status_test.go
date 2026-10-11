package handler

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"todo-api/internal/domain"
)

// statusesOf は GET /statuses のうち、スペースのステータスを「名前/カテゴリ」で並び順に返す
func statusesOf(t *testing.T, r *gin.Engine, spaceID int) []domain.WorkflowStatus {
	t.Helper()
	result := []domain.WorkflowStatus{}
	for _, s := range getJSON[[]domain.WorkflowStatus](t, r, "/statuses") {
		if s.SpaceID == spaceID {
			result = append(result, s)
		}
	}
	return result
}

func statusText(statuses []domain.WorkflowStatus) string {
	parts := []string{}
	for _, s := range statuses {
		parts = append(parts, s.Name+"/"+string(s.Category))
	}
	return strings.Join(parts, ",")
}

func todoStatus(t *testing.T, r *gin.Engine, key string) string {
	t.Helper()
	todo := getJSON[domain.Todo](t, r, "/issues/"+key)
	return fmt.Sprintf("%d:%s:%s:%s:%v", todo.StatusID, todo.StatusName, todo.StatusCategory, todo.Status, todo.Done)
}

func TestStatuses_DefaultWorkflows(t *testing.T) {
	r := setupSpaceRouter(t)
	mustStatus(t, r, http.MethodPost, "/spaces", `{"Title":"開発","Key":"DEV"}`, http.StatusCreated)

	want := "オープン/ToDo,進行中/InProgress,待ち状況/InProgress,クローズ/Done"
	// 未割り当て（0）と作ったスペースに、既定の 4 ステータスがある
	for _, spaceID := range []int{0, 1} {
		if got := statusText(statusesOf(t, r, spaceID)); got != want {
			t.Errorf("space %d statuses = %s", spaceID, got)
		}
	}
	// LegacyKey は API に出さない
	if w := doRequest(t, r, http.MethodGet, "/statuses", ""); strings.Contains(w.Body.String(), "LegacyKey") {
		t.Errorf("response exposes LegacyKey: %s", w.Body)
	}
}

func TestStatuses_TodoStatus(t *testing.T) {
	r := setupSpaceRouter(t)
	mustStatus(t, r, http.MethodPost, "/spaces", `{"Title":"開発","Key":"DEV"}`, http.StatusCreated)
	mustStatus(t, r, http.MethodPost, "/spaces", `{"Title":"運用","Key":"OPS"}`, http.StatusCreated)
	dev, ops := statusesOf(t, r, 1), statusesOf(t, r, 2)

	// 指定がなければスペースの To Do の先頭
	mustStatus(t, r, http.MethodPost, "/todos", `{"title":"a","spaceId":1}`, http.StatusCreated)
	if got, want := todoStatus(t, r, "DEV-1"), fmt.Sprintf("%d:オープン:ToDo:Open:false", dev[0].ID); got != want {
		t.Errorf("created = %s, want %s", got, want)
	}
	// statusId で指定する。Done カテゴリなら完了になる（done の指定は使わない）
	mustStatus(t, r, http.MethodPut, "/todos/1", fmt.Sprintf(`{"title":"a","spaceId":1,"statusId":%d,"done":false}`, dev[3].ID), http.StatusOK)
	if got, want := todoStatus(t, r, "DEV-1"), fmt.Sprintf("%d:クローズ:Done:Closed:true", dev[3].ID); got != want {
		t.Errorf("closed = %s, want %s", got, want)
	}
	// 導入前の status 文字列でも指定できる
	mustStatus(t, r, http.MethodPut, "/todos/1", `{"title":"a","spaceId":1,"status":"Waiting","done":true}`, http.StatusOK)
	if got, want := todoStatus(t, r, "DEV-1"), fmt.Sprintf("%d:待ち状況:InProgress:Waiting:false", dev[2].ID); got != want {
		t.Errorf("waiting = %s, want %s", got, want)
	}
	// ステータスを送らない更新では変わらない
	mustStatus(t, r, http.MethodPut, "/todos/1", `{"title":"b","spaceId":1}`, http.StatusOK)
	if got, want := todoStatus(t, r, "DEV-1"), fmt.Sprintf("%d:待ち状況:InProgress:Waiting:false", dev[2].ID); got != want {
		t.Errorf("without status = %s, want %s", got, want)
	}
	// スペースを移すと、移動先の同じ名前のステータスになる
	mustStatus(t, r, http.MethodPut, "/todos/1", `{"title":"b","spaceId":2}`, http.StatusOK)
	if got, want := todoStatus(t, r, "OPS-1"), fmt.Sprintf("%d:待ち状況:InProgress:Waiting:false", ops[2].ID); got != want {
		t.Errorf("moved = %s, want %s", got, want)
	}
	// 移動と同時に、移動元のスペースのステータスを指定しても、移動先の対応するステータスになる
	mustStatus(t, r, http.MethodPut, "/todos/1", fmt.Sprintf(`{"title":"b","spaceId":1,"statusId":%d}`, ops[3].ID), http.StatusOK)
	if got, want := todoStatus(t, r, "DEV-2"), fmt.Sprintf("%d:クローズ:Done:Closed:true", dev[3].ID); got != want {
		t.Errorf("moved with old status = %s, want %s", got, want)
	}

	mustStatus(t, r, http.MethodPut, "/todos/1", `{"title":"b","spaceId":1,"statusId":9999}`, http.StatusBadRequest)
	mustStatus(t, r, http.MethodPut, "/todos/1", `{"title":"b","spaceId":1,"status":"Someday"}`, http.StatusBadRequest)
	mustStatus(t, r, http.MethodPost, "/todos", `{"title":"c","status":"Someday"}`, http.StatusBadRequest)
}

// スペースを削除すると、課題は未割り当てのワークフローの同じ名前のステータスに移り、スペースのステータスは消える
func TestStatuses_DeleteSpace(t *testing.T) {
	r := setupSpaceRouter(t)
	mustStatus(t, r, http.MethodPost, "/spaces", `{"Title":"開発","Key":"DEV"}`, http.StatusCreated)
	mustStatus(t, r, http.MethodPost, "/todos", `{"title":"a","spaceId":1,"status":"InProgress"}`, http.StatusCreated)
	unassigned := statusesOf(t, r, 0)

	mustStatus(t, r, http.MethodDelete, "/spaces/1", "", http.StatusOK)
	if got, want := todoStatus(t, r, "TASK-1"), fmt.Sprintf("%d:進行中:InProgress:InProgress:false", unassigned[1].ID); got != want {
		t.Errorf("after delete = %s, want %s", got, want)
	}
	if got := statusesOf(t, r, 1); len(got) != 0 {
		t.Errorf("statuses of deleted space = %+v", got)
	}
}
