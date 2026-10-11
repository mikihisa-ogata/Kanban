package handler

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"

	"todo-api/internal/domain"
	"todo-api/internal/service"
)

func newWorklogRouter(t *testing.T) (*gin.Engine, testRepos) {
	t.Helper()
	repos := newTestRepos(t)
	r := newTodoRouter(repos)
	h := NewWorklogHandler(service.NewWorklogService(repos.worklog, repos.todo, repos.epic, repos.space, repos.history))
	r.GET("/todos/:id/worklogs", h.GetWorklogs)
	r.POST("/todos/:id/worklogs", h.AddWorklog)
	r.PUT("/worklogs/:id", h.UpdateWorklog)
	r.DELETE("/worklogs/:id", h.DeleteWorklog)
	return r, repos
}

// estimates は課題の「初期見積もり/残り時間/記録済み」を分で返す（未設定は -）
func estimates(t *testing.T, r *gin.Engine, key string) string {
	t.Helper()
	todo := getJSON[domain.Todo](t, r, "/issues/"+key)
	text := func(v *int) string {
		if v == nil {
			return "-"
		}
		return fmt.Sprint(*v)
	}
	return text(todo.OriginalEstimate) + "/" + text(todo.RemainingEstimate) + "/" + fmt.Sprint(todo.TimeSpent)
}

const startedAt = `"startedAt":"2026-10-11T10:00:00+09:00"`

func TestTodoHandler_Estimates(t *testing.T) {
	r, _ := newWorklogRouter(t)
	mustStatus(t, r, http.MethodPost, "/todos", `{"title":"a"}`, http.StatusCreated)
	if got := estimates(t, r, "TASK-1"); got != "-/-/0" {
		t.Fatalf("initial = %s", got)
	}

	// 初期見積もりを設定すると、残り時間も同じ値になる
	mustStatus(t, r, http.MethodPut, "/todos/1", `{"title":"a","status":"Open","originalEstimate":"1w 2d"}`, http.StatusOK)
	if got := estimates(t, r, "TASK-1"); got != "3360/3360/0" {
		t.Errorf("after original = %s", got)
	}
	// 見積もりを送らない更新（ドラッグでの移動など）では変わらない
	mustStatus(t, r, http.MethodPut, "/todos/1", `{"title":"a","status":"InProgress"}`, http.StatusOK)
	if got := estimates(t, r, "TASK-1"); got != "3360/3360/0" {
		t.Errorf("after update without estimates = %s", got)
	}
	// 残り時間は手で変えられる。初期見積もりを変えても、設定済みの残り時間はそのまま
	mustStatus(t, r, http.MethodPut, "/todos/1", `{"title":"a","status":"InProgress","remainingEstimate":"3h 30m"}`, http.StatusOK)
	mustStatus(t, r, http.MethodPut, "/todos/1", `{"title":"a","status":"InProgress","originalEstimate":"2d"}`, http.StatusOK)
	if got := estimates(t, r, "TASK-1"); got != "960/210/0" {
		t.Errorf("after manual remaining = %s", got)
	}
	// null と空文字は未設定に戻す
	mustStatus(t, r, http.MethodPut, "/todos/1", `{"title":"a","status":"InProgress","originalEstimate":null,"remainingEstimate":""}`, http.StatusOK)
	if got := estimates(t, r, "TASK-1"); got != "-/-/0" {
		t.Errorf("after clearing = %s", got)
	}

	for _, body := range []string{
		`{"title":"a","status":"Open","originalEstimate":"3"}`,
		`{"title":"a","status":"Open","originalEstimate":"2 days"}`,
		`{"title":"a","status":"Open","remainingEstimate":5}`,
	} {
		mustStatus(t, r, http.MethodPut, "/todos/1", body, http.StatusBadRequest)
	}

	// 見積もりの変更は分で履歴に残る
	got := historyOf(t, r, "/todos/1/history")
	want := "originalEstimate:->3360"
	found := false
	for _, entry := range got {
		found = found || entry == want
	}
	if !found {
		t.Errorf("history %q does not contain %q", got, want)
	}
}

func TestWorklogHandler_AdjustsRemaining(t *testing.T) {
	r, _ := newWorklogRouter(t)
	mustStatus(t, r, http.MethodPost, "/todos", `{"title":"a"}`, http.StatusCreated)
	mustStatus(t, r, http.MethodPut, "/todos/1", `{"title":"a","status":"Open","originalEstimate":"1d"}`, http.StatusOK)

	// 作業を記録すると残り時間が自動で減る
	mustStatus(t, r, http.MethodPost, "/todos/1/worklogs", `{"timeSpent":"2h",`+startedAt+`,"note":"調査"}`, http.StatusCreated)
	if got := estimates(t, r, "TASK-1"); got != "480/360/120" {
		t.Errorf("after 2h = %s", got)
	}
	// 残り時間を指定して記録できる
	mustStatus(t, r, http.MethodPost, "/todos/1/worklogs", `{"timeSpent":"30m",`+startedAt+`,"remainingEstimate":"4h"}`, http.StatusCreated)
	if got := estimates(t, r, "TASK-1"); got != "480/240/150" {
		t.Errorf("after 30m with remaining 4h = %s", got)
	}

	worklogs := getJSON[[]domain.Worklog](t, r, "/todos/1/worklogs")
	if len(worklogs) != 2 || worklogs[0].Minutes != 120 || worklogs[0].Note != "調査" || worklogs[0].StartedAt.UTC().Hour() != 1 || worklogs[0].UpdatedAt != nil {
		t.Fatalf("worklogs = %+v", worklogs)
	}

	// 作業時間を 2h → 3h に直すと、増えた 1h だけ残り時間が減る
	mustStatus(t, r, http.MethodPut, "/worklogs/1", `{"timeSpent":"3h",`+startedAt+`,"note":"調査と修正"}`, http.StatusOK)
	if got := estimates(t, r, "TASK-1"); got != "480/180/210" {
		t.Errorf("after edit = %s", got)
	}
	if w := getJSON[[]domain.Worklog](t, r, "/todos/1/worklogs"); w[0].Note != "調査と修正" || w[0].UpdatedAt == nil {
		t.Errorf("edited worklog = %+v", w[0])
	}
	// 削除すると、その作業時間だけ残り時間が戻る
	mustStatus(t, r, http.MethodDelete, "/worklogs/1", "", http.StatusOK)
	if got := estimates(t, r, "TASK-1"); got != "480/360/30" {
		t.Errorf("after delete = %s", got)
	}
	// 残り時間は 0 未満にならない
	mustStatus(t, r, http.MethodPost, "/todos/1/worklogs", `{"timeSpent":"1w",`+startedAt+`}`, http.StatusCreated)
	if got := estimates(t, r, "TASK-1"); got != "480/0/2430" {
		t.Errorf("after overrun = %s", got)
	}
	// 残り時間の変化は履歴に残る
	got := historyOf(t, r, "/todos/1/history")
	if last := got[len(got)-1]; last != "remainingEstimate:360>0" {
		t.Errorf("last history = %q", last)
	}
}

func TestWorklogHandler_WithoutEstimates(t *testing.T) {
	r, _ := newWorklogRouter(t)
	mustStatus(t, r, http.MethodPost, "/todos", `{"title":"a"}`, http.StatusCreated)
	mustStatus(t, r, http.MethodPost, "/todos", `{"title":"b"}`, http.StatusCreated)

	// 見積もりがなければ残り時間は未設定のまま
	mustStatus(t, r, http.MethodPost, "/todos/1/worklogs", `{"timeSpent":"1h",`+startedAt+`}`, http.StatusCreated)
	if got := estimates(t, r, "TASK-1"); got != "-/-/60" {
		t.Errorf("no estimates = %s", got)
	}
	// 初期見積もりだけあれば「初期見積もり − 記録済み」になる
	mustStatus(t, r, http.MethodPut, "/todos/2", `{"title":"b","status":"Open","originalEstimate":"5h","remainingEstimate":null}`, http.StatusOK)
	mustStatus(t, r, http.MethodPost, "/todos/2/worklogs", `{"timeSpent":"2h",`+startedAt+`}`, http.StatusCreated)
	if got := estimates(t, r, "TASK-2"); got != "300/180/120" {
		t.Errorf("original only = %s", got)
	}
}

func TestWorklogHandler_Errors(t *testing.T) {
	r, repos := newWorklogRouter(t)
	mustStatus(t, r, http.MethodPost, "/todos", `{"title":"a"}`, http.StatusCreated)
	mustStatus(t, r, http.MethodPost, "/todos/1/worklogs", `{"timeSpent":"1h",`+startedAt+`}`, http.StatusCreated)

	for _, body := range []string{
		`{"timeSpent":"0m",` + startedAt + `}`,
		`{"timeSpent":"abc",` + startedAt + `}`,
		`{"timeSpent":"1h"}`,
		`{"timeSpent":"1h","startedAt":"2026-10-11 10:00"}`,
		`{"timeSpent":"1h",` + startedAt + `,"remainingEstimate":"x"}`,
	} {
		mustStatus(t, r, http.MethodPost, "/todos/1/worklogs", body, http.StatusBadRequest)
	}
	mustStatus(t, r, http.MethodPost, "/todos/99/worklogs", `{"timeSpent":"1h",`+startedAt+`}`, http.StatusNotFound)
	mustStatus(t, r, http.MethodGet, "/todos/99/worklogs", "", http.StatusNotFound)
	mustStatus(t, r, http.MethodPut, "/worklogs/99", `{"timeSpent":"1h",`+startedAt+`}`, http.StatusNotFound)
	mustStatus(t, r, http.MethodDelete, "/worklogs/99", "", http.StatusNotFound)

	// 課題を削除すると作業ログも消える
	mustStatus(t, r, http.MethodDelete, "/todos/1", "", http.StatusOK)
	if worklogs, err := repos.worklog.FindByTodo(1); err != nil || len(worklogs) != 0 {
		t.Errorf("worklogs after delete = %+v, %v", worklogs, err)
	}
}
