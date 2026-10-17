package handler

import (
	"fmt"
	"net/http"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"todo-api/internal/domain"
	"todo-api/internal/service"
)

// newBulkRouter はログインが必要な一括変更・一括削除と、確認に使う課題・スペース・スプリント・バージョン・エピックの API を持つルーターを返す。
// 添付ファイルのアップロード（upload で送る）だけはログインなしで使える
func newBulkRouter(t *testing.T) (*gin.Engine, testRepos) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	repos := newTestRepos(t)
	auth := NewAuthHandler(service.NewAuthService(repos.user, repos.session))
	todo := NewTodoHandler(service.NewTodoService(repos.todo, repos.epic, repos.space, repos.link, repos.history, repos.attachment, repos.status, repos.user, repos.watcher, repos.notification))
	space := NewSpaceHandler(service.NewSpaceService(repos.space, repos.epic, repos.todo, repos.history, repos.status))
	sprint := NewSprintHandler(service.NewSprintService(repos.sprint, repos.todo, repos.space, repos.history))
	version := NewVersionHandler(service.NewVersionService(repos.version, repos.todo, repos.space))
	epic := NewEpicHandler(service.NewEpicService(repos.epic, repos.todo, repos.space, repos.history, repos.status))
	attachment := NewAttachmentHandler(service.NewAttachmentService(repos.attachment, repos.todo))
	bulk := NewBulkHandler(service.NewBulkService(repos.todo, repos.epic, repos.space, repos.sprint, repos.version, repos.user, repos.attachment, repos.status, repos.watcher, repos.notification))

	r := gin.New()
	r.POST("/auth/signup", auth.Signup)
	r.POST("/todos/:id/attachments", attachment.AddAttachment)
	api := r.Group("/", auth.RequireLogin)
	api.GET("/todos", todo.GetTodos)
	api.POST("/todos", todo.CreateTodo)
	api.GET("/todos/:id/history", todo.GetHistory)
	api.POST("/spaces", space.CreateSpace)
	api.GET("/statuses", space.GetStatuses)
	api.POST("/spaces/:id/sprints", sprint.CreateSprint)
	api.PUT("/todos/:id/sprint", sprint.SetTodoSprint)
	api.POST("/spaces/:id/versions", version.CreateVersion)
	api.PUT("/todos/:id/fix-versions", version.SetTodoVersions)
	api.POST("/epics", epic.CreateEpic)
	api.POST("/issues/bulk-update", bulk.UpdateIssues)
	api.POST("/issues/bulk-delete", bulk.DeleteIssues)
	return r, repos
}

// todosByID は token のユーザーとして課題一覧を取り、ID から課題への対応にして返す
func todosByID(t *testing.T, r *gin.Engine, token string) map[int]domain.Todo {
	t.Helper()
	result := map[int]domain.Todo{}
	for _, todo := range decode[[]domain.Todo](t, sendAs(t, r, token, http.MethodGet, "/todos", "", http.StatusOK)) {
		result[todo.ID] = todo
	}
	return result
}

// statusID はスペースの名前のステータスの ID を返す
func statusID(t *testing.T, r *gin.Engine, token string, spaceID int, name string) int {
	t.Helper()
	for _, s := range decode[[]domain.WorkflowStatus](t, sendAs(t, r, token, http.MethodGet, "/statuses", "", http.StatusOK)) {
		if s.SpaceID == spaceID && s.Name == name {
			return s.ID
		}
	}
	t.Fatalf("status %q of space %d not found", name, spaceID)
	return 0
}

// bulkHistory は課題の変更履歴のうち作成以外を「項目:変更前>変更後」にし、変更者がすべて userID であることを確かめて返す
func bulkHistory(t *testing.T, r *gin.Engine, token string, todoID int, userID int) []string {
	t.Helper()
	result := []string{}
	for _, e := range decode[[]domain.HistoryEntry](t, sendAs(t, r, token, http.MethodGet, fmt.Sprintf("/todos/%d/history", todoID), "", http.StatusOK)) {
		if e.Field == domain.HistoryFieldCreated {
			continue
		}
		if e.UserID != userID {
			t.Errorf("history %+v UserID = %d, want %d", e, e.UserID, userID)
		}
		result = append(result, e.Field+":"+historyValue(e.OldValue)+">"+historyValue(e.NewValue))
	}
	return result
}

func TestBulk_UpdateIssues(t *testing.T) {
	r, repos := newBulkRouter(t)
	actor, token := signup(t, r, "操作 太郎", "actor@example.com")
	assignee, _ := signup(t, r, "担当 花子", "assignee@example.com")

	sendAs(t, r, token, http.MethodPost, "/spaces", `{"Title":"開発","Key":"DEV"}`, http.StatusCreated)
	sendAs(t, r, token, http.MethodPost, "/spaces", `{"Title":"運用","Key":"OPS"}`, http.StatusCreated)
	sendAs(t, r, token, http.MethodPost, "/todos", `{"title":"a","spaceId":1}`, http.StatusCreated)
	sendAs(t, r, token, http.MethodPost, "/todos", `{"title":"b","spaceId":1,"labels":["old","keep"]}`, http.StatusCreated)
	sendAs(t, r, token, http.MethodPost, "/todos", `{"title":"a のサブタスク","parentId":1}`, http.StatusCreated)
	sendAs(t, r, token, http.MethodPost, "/todos", `{"title":"c","spaceId":2}`, http.StatusCreated)
	sprint := decode[struct{ Sprint domain.Sprint }](t, sendAs(t, r, token, http.MethodPost, "/spaces/1/sprints", `{"name":"DEV スプリント 1"}`, http.StatusCreated)).Sprint
	version := decode[struct{ Version domain.Version }](t, sendAs(t, r, token, http.MethodPost, "/spaces/1/versions", `{"name":"v1.0"}`, http.StatusCreated)).Version
	opsVersion := decode[struct{ Version domain.Version }](t, sendAs(t, r, token, http.MethodPost, "/spaces/2/versions", `{"name":"ops-1"}`, http.StatusCreated)).Version
	sendAs(t, r, token, http.MethodPost, "/epics", `{"title":"ログイン","spaceId":1}`, http.StatusCreated)
	inProgress := statusID(t, r, token, 1, "進行中")

	body := fmt.Sprintf(`{"ids":[1,2],"statusId":%d,"assigneeId":%d,"priority":"High","addLabels":["new"],"removeLabels":["OLD"],"sprintId":%d,"epicId":1,"addFixVersionIds":[%d]}`,
		inProgress, assignee.ID, sprint.ID, version.ID)
	res := decode[struct{ Count int }](t, sendAs(t, r, token, http.MethodPost, "/issues/bulk-update", body, http.StatusOK))
	// 親課題のスプリント・エピックを変えたので、サブタスク（ID 3）も変わる
	if res.Count != 3 {
		t.Errorf("count = %d, want 3", res.Count)
	}

	todos := todosByID(t, r, token)
	for _, id := range []int{1, 2} {
		got := todos[id]
		if got.StatusID != inProgress || got.AssigneeID != assignee.ID || got.Priority != domain.PriorityHigh ||
			got.SprintID != sprint.ID || got.EpicID != 1 || !reflect.DeepEqual(got.FixVersionIDs, []int{version.ID}) {
			t.Errorf("todo %d = %+v", id, got)
		}
	}
	if want := []string{"keep", "new"}; !reflect.DeepEqual(todos[2].Labels, want) {
		t.Errorf("todo 2 labels = %q, want %q", todos[2].Labels, want)
	}
	if want := []string{"new"}; !reflect.DeepEqual(todos[1].Labels, want) {
		t.Errorf("todo 1 labels = %q, want %q", todos[1].Labels, want)
	}
	// サブタスクはスプリント・エピックだけ親に合わせ、ほかの項目は変えない
	if sub := todos[3]; sub.SprintID != sprint.ID || sub.EpicID != 1 || sub.StatusID == inProgress || sub.Priority != domain.PriorityMedium {
		t.Errorf("subtask = %+v", sub)
	}
	if c := todos[4]; c.Priority != domain.PriorityMedium || c.AssigneeID != 0 {
		t.Errorf("unselected todo changed: %+v", c)
	}

	want := []string{
		"status:オープン>進行中",
		"statusCategory:ToDo>InProgress",
		"priority:Medium>High",
		"labels:keep, old>keep, new",
		"epic:->ログイン",
		"assignee:->担当 花子",
		"sprint:->DEV スプリント 1",
		"fixVersions:->v1.0",
	}
	if got := bulkHistory(t, r, token, 2, actor.ID); strings.Join(got, "|") != strings.Join(want, "|") {
		t.Errorf("history of todo 2 =\n%q\nwant\n%q", got, want)
	}
	if got := bulkHistory(t, r, token, 3, actor.ID); strings.Join(got, "|") != "epic:->ログイン|sprint:->DEV スプリント 1" {
		t.Errorf("history of subtask = %q", got)
	}

	// 新しい担当者はウォッチャーになり、変更の通知が届く
	watchers, err := repos.watcher.FindByTodo(2)
	if err != nil {
		t.Fatal(err)
	}
	if !containsWatcher(watchers, assignee.ID) {
		t.Errorf("watchers of todo 2 = %+v, want to include %d", watchers, assignee.ID)
	}
	notifications, err := repos.notification.FindByUser(assignee.ID, 50)
	if err != nil {
		t.Fatal(err)
	}
	notified := []int{}
	for _, n := range notifications {
		notified = append(notified, n.TodoID)
	}
	sort.Ints(notified)
	if !reflect.DeepEqual(notified, []int{1, 2}) {
		t.Errorf("notified todos = %v, want [1 2]", notified)
	}

	// 別のスペースの課題には、同じ名前のステータスを当てる。変わらない課題は数えず、履歴も残さない
	res = decode[struct{ Count int }](t, sendAs(t, r, token, http.MethodPost, "/issues/bulk-update",
		fmt.Sprintf(`{"ids":[1,4],"statusId":%d}`, inProgress), http.StatusOK))
	if res.Count != 1 {
		t.Errorf("count = %d, want 1", res.Count)
	}
	if got := todosByID(t, r, token)[4].StatusID; got != statusID(t, r, token, 2, "進行中") {
		t.Errorf("OPS todo status = %d", got)
	}
	if got := bulkHistory(t, r, token, 1, actor.ID); len(got) != 8 {
		t.Errorf("history of todo 1 has %d entries, want 8: %q", len(got), got)
	}

	// 修正バージョンを外し、ラベルを外し、担当者・スプリント・エピックをなしにする
	sendAs(t, r, token, http.MethodPost, "/issues/bulk-update",
		fmt.Sprintf(`{"ids":[2],"removeFixVersionIds":[%d],"removeLabels":["keep","new"],"assigneeId":null,"sprintId":null,"epicId":0}`, version.ID), http.StatusOK)
	if got := todosByID(t, r, token)[2]; got.AssigneeID != 0 || got.SprintID != 0 || got.EpicID != 0 || len(got.FixVersionIDs) != 0 || len(got.Labels) != 0 {
		t.Errorf("todo 2 after clearing = %+v", got)
	}

	// 1 件でも当てられない変更があれば、どの課題も変えない
	before := todosByID(t, r, token)
	for _, tc := range []struct {
		name string
		body string
		want int
	}{
		{"エピックが別のスペース", `{"ids":[1,4],"priority":"Lowest","epicId":1}`, http.StatusBadRequest},
		{"スプリントが別のスペース", fmt.Sprintf(`{"ids":[1,4],"priority":"Lowest","sprintId":%d}`, sprint.ID), http.StatusBadRequest},
		{"バージョンが別のスペース", fmt.Sprintf(`{"ids":[1,4],"priority":"Lowest","addFixVersionIds":[%d]}`, opsVersion.ID), http.StatusBadRequest},
		{"存在しない課題", `{"ids":[1,99],"priority":"Lowest"}`, http.StatusNotFound},
		{"存在しない担当者", `{"ids":[1],"assigneeId":999}`, http.StatusBadRequest},
		{"不正な優先度", `{"ids":[1],"priority":"Urgent"}`, http.StatusBadRequest},
		{"不正なラベル", `{"ids":[1],"addLabels":["a b"]}`, http.StatusBadRequest},
		{"課題がない", `{"ids":[],"priority":"Low"}`, http.StatusBadRequest},
		{"変更する項目がない", `{"ids":[1]}`, http.StatusBadRequest},
		{"不正なスプリントの指定", `{"ids":[1],"sprintId":"x"}`, http.StatusBadRequest},
	} {
		t.Run(tc.name, func(t *testing.T) {
			sendAs(t, r, token, http.MethodPost, "/issues/bulk-update", tc.body, tc.want)
			if after := todosByID(t, r, token); !reflect.DeepEqual(after, before) {
				t.Errorf("todos changed:\nbefore %+v\nafter  %+v", before, after)
			}
		})
	}
}

func TestBulk_DeleteIssues(t *testing.T) {
	r, repos := newBulkRouter(t)
	_, token := signup(t, r, "操作 太郎", "actor@example.com")

	sendAs(t, r, token, http.MethodPost, "/spaces", `{"Title":"開発","Key":"DEV"}`, http.StatusCreated)
	sendAs(t, r, token, http.MethodPost, "/todos", `{"title":"a","spaceId":1}`, http.StatusCreated)
	sendAs(t, r, token, http.MethodPost, "/todos", `{"title":"b","spaceId":1}`, http.StatusCreated)
	sendAs(t, r, token, http.MethodPost, "/todos", `{"title":"a のサブタスク","parentId":1}`, http.StatusCreated)
	sendAs(t, r, token, http.MethodPost, "/todos", `{"title":"c","spaceId":1}`, http.StatusCreated)
	for _, id := range []int{1, 3, 4} {
		if w := upload(t, r, fmt.Sprintf("/todos/%d/attachments", id), "memo.txt", []byte("memo")); w.Code != http.StatusCreated {
			t.Fatalf("upload status = %d, body = %s", w.Code, w.Body)
		}
	}

	// 存在しない課題が混じっていれば、どれも消さない
	sendAs(t, r, token, http.MethodPost, "/issues/bulk-delete", `{"ids":[2,99]}`, http.StatusNotFound)
	if got := len(todosByID(t, r, token)); got != 4 {
		t.Fatalf("todos = %d, want 4", got)
	}
	sendAs(t, r, token, http.MethodPost, "/issues/bulk-delete", `{"ids":[]}`, http.StatusBadRequest)

	// 親課題とそのサブタスクを両方選んでも消せる。サブタスクの数も含めて返す
	res := decode[struct{ Count int }](t, sendAs(t, r, token, http.MethodPost, "/issues/bulk-delete", `{"ids":[1,3,2]}`, http.StatusOK))
	if res.Count != 3 {
		t.Errorf("count = %d, want 3", res.Count)
	}
	todos := todosByID(t, r, token)
	if _, ok := todos[4]; len(todos) != 1 || !ok {
		t.Errorf("todos after delete = %+v, want only 4", todos)
	}
	// 消した課題の添付ファイルは本体も消え、残った課題の添付ファイルは残る
	if got := storedFiles(t, repos.attachmentsDir); got != 1 {
		t.Errorf("stored files = %d, want 1", got)
	}
}
