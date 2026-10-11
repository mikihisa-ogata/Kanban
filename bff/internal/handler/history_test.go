package handler

import (
	"net/http"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"todo-api/internal/domain"
)

// historyOf は課題の変更履歴を「項目:変更前>変更後」の形で返す（作成は created）
func historyOf(t *testing.T, r *gin.Engine, path string) []string {
	t.Helper()
	entries := getJSON[[]domain.HistoryEntry](t, r, path)
	result := []string{}
	for _, e := range entries {
		if e.CreatedAt.IsZero() {
			t.Errorf("entry %+v has no CreatedAt", e)
		}
		if e.Field == domain.HistoryFieldCreated {
			result = append(result, e.Field)
			continue
		}
		result = append(result, e.Field+":"+historyValue(e.OldValue)+">"+historyValue(e.NewValue))
	}
	return result
}

func historyValue(s *string) string {
	if s == nil {
		return "-"
	}
	return *s
}

// 作成と、項目の変更が 1 項目 1 件で記録される。変わらない更新では記録されない
func TestTodoHandler_History(t *testing.T) {
	r := setupSpaceRouter(t)
	mustStatus(t, r, http.MethodPost, "/spaces", `{"Title":"開発","Key":"DEV"}`, http.StatusCreated)
	mustStatus(t, r, http.MethodPost, "/todos", `{"title":"a","spaceId":1,"storyPoints":3}`, http.StatusCreated)

	if got := historyOf(t, r, "/todos/1/history"); strings.Join(got, "|") != "created" {
		t.Fatalf("after create: %q", got)
	}

	mustStatus(t, r, http.MethodPut, "/todos/1", `{"title":"a","status":"InProgress","spaceId":1}`, http.StatusOK)
	mustStatus(t, r, http.MethodPut, "/todos/1", `{"title":"b","status":"InProgress","spaceId":1,"storyPoints":5,"labels":["ui"],"priority":"High"}`, http.StatusOK)
	// 同じ内容の更新（ドラッグで同じ列に戻すなど）は記録しない
	mustStatus(t, r, http.MethodPut, "/todos/1", `{"title":"b","status":"InProgress","spaceId":1}`, http.StatusOK)
	// スペースを外すとキーも変わる
	mustStatus(t, r, http.MethodPut, "/todos/1", `{"title":"b","status":"Closed","spaceId":0}`, http.StatusOK)

	want := []string{
		"created",
		// ステータスは名前で残り、カテゴリが変われば一緒に残る
		"status:オープン>進行中",
		"statusCategory:ToDo>InProgress",
		"title:a>b",
		"priority:Medium>High",
		"labels:->ui",
		"storyPoints:3>5",
		"status:進行中>クローズ",
		"statusCategory:InProgress>Done",
		// 未割り当てのワークフローにも同じ名前のステータスがあるので、ステータスは変わらない
		"space:開発>-",
		"key:DEV-1>TASK-1",
	}
	if got := historyOf(t, r, "/todos/1/history"); strings.Join(got, "|") != strings.Join(want, "|") {
		t.Errorf("history =\n%q\nwant\n%q", got, want)
	}

	mustStatus(t, r, http.MethodGet, "/todos/99/history", "", http.StatusNotFound)
}

// エピックのスペース変更・スペースの削除・親課題の移動で変わった課題にも履歴が残る
func TestTodoHandler_HistoryOfCascadedChanges(t *testing.T) {
	r := setupSpaceRouter(t)
	mustStatus(t, r, http.MethodPost, "/spaces", `{"Title":"開発","Key":"DEV"}`, http.StatusCreated)
	mustStatus(t, r, http.MethodPost, "/spaces", `{"Title":"運用","Key":"OPS"}`, http.StatusCreated)
	mustStatus(t, r, http.MethodPost, "/epics", `{"title":"認証","spaceId":1}`, http.StatusCreated)
	mustStatus(t, r, http.MethodPost, "/todos", `{"title":"in epic","spaceId":1,"epicId":1}`, http.StatusCreated)
	mustStatus(t, r, http.MethodPost, "/todos", `{"title":"parent","spaceId":2}`, http.StatusCreated)
	mustStatus(t, r, http.MethodPost, "/todos", `{"title":"sub","parentId":2}`, http.StatusCreated)

	// エピックを運用へ移すと、子タスクのスペースとキーが変わる
	mustStatus(t, r, http.MethodPut, "/epics/1", `{"title":"認証","spaceId":2}`, http.StatusOK)
	if got := historyOf(t, r, "/todos/1/history"); strings.Join(got, "|") != "created|space:開発>運用|key:DEV-1>OPS-3" {
		t.Errorf("todo in moved epic: %q", got)
	}

	// 親課題を開発へ移すと、サブタスクも移って記録される
	mustStatus(t, r, http.MethodPut, "/todos/2", `{"title":"parent","status":"Open","spaceId":1}`, http.StatusOK)
	if got := historyOf(t, r, "/todos/3/history"); strings.Join(got, "|") != "created|space:運用>開発|key:OPS-2>DEV-3" {
		t.Errorf("subtask of moved parent: %q", got)
	}

	// スペースを削除すると、削除したスペースの名前で記録される
	mustStatus(t, r, http.MethodDelete, "/spaces/2", "", http.StatusOK)
	got := historyOf(t, r, "/todos/1/history")
	if last := strings.Join(got[len(got)-2:], "|"); last != "space:運用>-|key:OPS-3>TASK-1" {
		t.Errorf("todo in deleted space: %q", got)
	}
}

// 課題を削除すると、その変更履歴も消える
func TestTodoHandler_DeleteTodoDeletesHistory(t *testing.T) {
	repos := newTestRepos(t)
	r := newTodoRouter(repos)
	mustStatus(t, r, http.MethodPost, "/todos", `{"title":"a"}`, http.StatusCreated)
	mustStatus(t, r, http.MethodPut, "/todos/1", `{"title":"b","status":"Open"}`, http.StatusOK)
	mustStatus(t, r, http.MethodDelete, "/todos/1", "", http.StatusOK)
	entries, err := repos.history.FindByTodo(1)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Errorf("history after delete = %+v", entries)
	}
}
