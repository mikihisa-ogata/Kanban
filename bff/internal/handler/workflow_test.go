package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"todo-api/internal/domain"
)

// createStatus はステータスを追加して、追加したステータスを返す
func createStatus(t *testing.T, r *gin.Engine, spaceID int, body string) domain.WorkflowStatus {
	t.Helper()
	w := doRequest(t, r, http.MethodPost, fmt.Sprintf("/spaces/%d/statuses", spaceID), body)
	if w.Code != http.StatusCreated {
		t.Fatalf("POST statuses status = %d, body = %s", w.Code, w.Body)
	}
	var res struct{ Status domain.WorkflowStatus }
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		t.Fatal(err)
	}
	return res.Status
}

func TestWorkflow_CreateRenameReorder(t *testing.T) {
	r := setupSpaceRouter(t)
	mustStatus(t, r, http.MethodPost, "/spaces", `{"Title":"開発","Key":"DEV"}`, http.StatusCreated)

	review := createStatus(t, r, 1, `{"name":"  レビュー ","category":"InProgress"}`)
	if review.Name != "レビュー" || review.Category != domain.StatusCategoryInProgress || review.SpaceID != 1 || review.Position != 5 {
		t.Errorf("created = %+v", review)
	}
	if got := statusText(statusesOf(t, r, 1)); got != "オープン/ToDo,進行中/InProgress,待ち状況/InProgress,クローズ/Done,レビュー/InProgress" {
		t.Errorf("after create = %s", got)
	}

	// 名前の重複（大文字・小文字は区別しない）・空・長すぎる名前・知らないカテゴリ
	mustStatus(t, r, http.MethodPost, "/spaces/1/statuses", `{"name":"レビュー","category":"Done"}`, http.StatusConflict)
	createStatus(t, r, 1, `{"name":"QA","category":"InProgress"}`)
	mustStatus(t, r, http.MethodPost, "/spaces/1/statuses", `{"name":"qa","category":"InProgress"}`, http.StatusConflict)
	mustStatus(t, r, http.MethodPost, "/spaces/1/statuses", `{"name":"  ","category":"ToDo"}`, http.StatusBadRequest)
	mustStatus(t, r, http.MethodPost, "/spaces/1/statuses", fmt.Sprintf(`{"name":"%s","category":"ToDo"}`, strings.Repeat("あ", 51)), http.StatusBadRequest)
	mustStatus(t, r, http.MethodPost, "/spaces/1/statuses", `{"name":"x","category":"Later"}`, http.StatusBadRequest)
	// 未割り当てのワークフローは変えられない。存在しないスペースは 404
	mustStatus(t, r, http.MethodPost, "/spaces/0/statuses", `{"name":"x","category":"ToDo"}`, http.StatusBadRequest)
	mustStatus(t, r, http.MethodPost, "/spaces/99/statuses", `{"name":"x","category":"ToDo"}`, http.StatusNotFound)

	// 名前を変えると、課題のステータス名も変わる
	mustStatus(t, r, http.MethodPost, "/todos", fmt.Sprintf(`{"title":"a","spaceId":1,"statusId":%d}`, review.ID), http.StatusCreated)
	mustStatus(t, r, http.MethodPut, fmt.Sprintf("/statuses/%d", review.ID), `{"name":"コードレビュー"}`, http.StatusOK)
	if got := getJSON[domain.Todo](t, r, "/issues/DEV-1").StatusName; got != "コードレビュー" {
		t.Errorf("todo status name = %s", got)
	}
	mustStatus(t, r, http.MethodPut, fmt.Sprintf("/statuses/%d", review.ID), `{"name":"オープン"}`, http.StatusConflict)
	mustStatus(t, r, http.MethodPut, fmt.Sprintf("/statuses/%d", review.ID), `{"name":"コードレビュー"}`, http.StatusOK)
	unassigned := statusesOf(t, r, 0)
	mustStatus(t, r, http.MethodPut, fmt.Sprintf("/statuses/%d", unassigned[0].ID), `{"name":"x"}`, http.StatusBadRequest)
	mustStatus(t, r, http.MethodPut, "/statuses/9999", `{"name":"x"}`, http.StatusNotFound)

	// 並べ替え
	statuses := statusesOf(t, r, 1)
	order := []int{statuses[0].ID, statuses[1].ID, statuses[4].ID, statuses[5].ID, statuses[2].ID, statuses[3].ID}
	body, _ := json.Marshal(map[string][]int{"statusIds": order})
	mustStatus(t, r, http.MethodPut, "/spaces/1/statuses/order", string(body), http.StatusOK)
	if got := statusText(statusesOf(t, r, 1)); got != "オープン/ToDo,進行中/InProgress,コードレビュー/InProgress,QA/InProgress,待ち状況/InProgress,クローズ/Done" {
		t.Errorf("after reorder = %s", got)
	}
	for _, ids := range [][]int{order[:5], append(order[:5:5], order[0]), append(order[:5:5], unassigned[0].ID)} {
		body, _ := json.Marshal(map[string][]int{"statusIds": ids})
		mustStatus(t, r, http.MethodPut, "/spaces/1/statuses/order", string(body), http.StatusBadRequest)
	}
}

func TestWorkflow_DeleteStatus(t *testing.T) {
	r := setupSpaceRouter(t)
	mustStatus(t, r, http.MethodPost, "/spaces", `{"Title":"開発","Key":"DEV"}`, http.StatusCreated)
	mustStatus(t, r, http.MethodPost, "/spaces", `{"Title":"運用","Key":"OPS"}`, http.StatusCreated)
	review := createStatus(t, r, 1, `{"name":"レビュー","category":"InProgress"}`)
	unused := createStatus(t, r, 1, `{"name":"未使用","category":"ToDo"}`)
	dev, ops := statusesOf(t, r, 1), statusesOf(t, r, 2)

	mustStatus(t, r, http.MethodPost, "/todos", fmt.Sprintf(`{"title":"a","spaceId":1,"statusId":%d}`, review.ID), http.StatusCreated)
	mustStatus(t, r, http.MethodPost, "/todos", fmt.Sprintf(`{"title":"b","spaceId":1,"statusId":%d}`, review.ID), http.StatusCreated)

	// 使われていなければそのまま消せる
	mustStatus(t, r, http.MethodDelete, fmt.Sprintf("/statuses/%d", unused.ID), "", http.StatusOK)
	// 使われていれば移し先が要る。移し先は同じスペースの別のステータス
	mustStatus(t, r, http.MethodDelete, fmt.Sprintf("/statuses/%d", review.ID), "", http.StatusConflict)
	mustStatus(t, r, http.MethodDelete, fmt.Sprintf("/statuses/%d?moveTo=%d", review.ID, review.ID), "", http.StatusBadRequest)
	mustStatus(t, r, http.MethodDelete, fmt.Sprintf("/statuses/%d?moveTo=%d", review.ID, ops[0].ID), "", http.StatusBadRequest)
	mustStatus(t, r, http.MethodDelete, fmt.Sprintf("/statuses/%d?moveTo=abc", review.ID), "", http.StatusBadRequest)

	// クローズ（Done）へ移すと完了になり、履歴に残る
	mustStatus(t, r, http.MethodDelete, fmt.Sprintf("/statuses/%d?moveTo=%d", review.ID, dev[3].ID), "", http.StatusOK)
	for _, key := range []string{"DEV-1", "DEV-2"} {
		if got, want := todoStatus(t, r, key), fmt.Sprintf("%d:クローズ:Done:Closed:true", dev[3].ID); got != want {
			t.Errorf("%s = %s, want %s", key, got, want)
		}
	}
	got := historyOf(t, r, "/todos/1/history")
	if last := got[len(got)-2:]; last[0] != "status:レビュー>クローズ" || last[1] != "statusCategory:InProgress>Done" {
		t.Errorf("history = %q", got)
	}
	mustStatus(t, r, http.MethodDelete, fmt.Sprintf("/statuses/%d", review.ID), "", http.StatusNotFound)

	// 最後のステータスは消せない
	for _, s := range ops[:3] {
		mustStatus(t, r, http.MethodDelete, fmt.Sprintf("/statuses/%d", s.ID), "", http.StatusOK)
	}
	mustStatus(t, r, http.MethodDelete, fmt.Sprintf("/statuses/%d", ops[3].ID), "", http.StatusBadRequest)
	// 未割り当てのワークフローは消せない
	mustStatus(t, r, http.MethodDelete, fmt.Sprintf("/statuses/%d", statusesOf(t, r, 0)[0].ID), "", http.StatusBadRequest)
}

// 追加したステータスの課題に、画面が互換用の Status をそのまま送り返しても、ステータスは変わらない
func TestWorkflow_LegacyStatusRoundTrip(t *testing.T) {
	r := setupSpaceRouter(t)
	mustStatus(t, r, http.MethodPost, "/spaces", `{"Title":"開発","Key":"DEV"}`, http.StatusCreated)
	review := createStatus(t, r, 1, `{"name":"レビュー","category":"InProgress"}`)
	mustStatus(t, r, http.MethodPost, "/todos", fmt.Sprintf(`{"title":"a","spaceId":1,"statusId":%d}`, review.ID), http.StatusCreated)

	todo := getJSON[domain.Todo](t, r, "/issues/DEV-1")
	if todo.Status != domain.StatusInProgress {
		t.Fatalf("legacy status = %s", todo.Status)
	}
	mustStatus(t, r, http.MethodPut, "/todos/1", `{"title":"b","spaceId":1,"status":"InProgress"}`, http.StatusOK)
	if got := getJSON[domain.Todo](t, r, "/issues/DEV-1"); got.StatusID != review.ID {
		t.Errorf("status after round trip = %d %s, want %d", got.StatusID, got.StatusName, review.ID)
	}
	// 違う値を送れば、その値の既定のステータスになる
	mustStatus(t, r, http.MethodPut, "/todos/1", `{"title":"b","spaceId":1,"status":"Waiting"}`, http.StatusOK)
	if got := getJSON[domain.Todo](t, r, "/issues/DEV-1").StatusName; got != "待ち状況" {
		t.Errorf("status after legacy change = %s", got)
	}
}
