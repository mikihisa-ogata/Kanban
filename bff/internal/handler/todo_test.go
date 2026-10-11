package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"todo-api/internal/database/dbtest"
	"todo-api/internal/domain"
	"todo-api/internal/repository"
	"todo-api/internal/service"
)

type testRepos struct {
	todo       repository.TodoRepository
	epic       repository.EpicRepository
	space      repository.SpaceRepository
	link       repository.IssueLinkRepository
	comment    repository.CommentRepository
	history    repository.HistoryRepository
	attachment repository.AttachmentRepository
	// attachmentsDir は添付ファイルの保存先（テストごとの一時ディレクトリ）
	attachmentsDir string
}

// newTestRepos は空のテスト用 MySQL を使うリポジトリを返す（TEST_DB_DSN が未設定ならスキップ）
func newTestRepos(t *testing.T) testRepos {
	t.Helper()
	db := dbtest.Open(t, "handler")
	dir := t.TempDir()
	return testRepos{
		todo:           repository.NewTodoMySQLRepository(db),
		epic:           repository.NewEpicMySQLRepository(db),
		space:          repository.NewSpaceMySQLRepository(db),
		link:           repository.NewIssueLinkMySQLRepository(db),
		comment:        repository.NewCommentMySQLRepository(db),
		history:        repository.NewHistoryMySQLRepository(db),
		attachment:     repository.NewAttachmentMySQLRepository(db, dir),
		attachmentsDir: dir,
	}
}

func setupTodoRouter(t *testing.T) *gin.Engine {
	t.Helper()
	return newTodoRouter(newTestRepos(t))
}

func newTodoRouter(repos testRepos) *gin.Engine {
	gin.SetMode(gin.TestMode)
	h := NewTodoHandler(service.NewTodoService(repos.todo, repos.epic, repos.space, repos.link, repos.history, repos.attachment))

	r := gin.New()
	r.GET("/todos", h.GetTodos)
	r.POST("/todos", h.CreateTodo)
	r.PUT("/todos/:id", h.UpdateTodo)
	r.POST("/todos/:id/move", h.MoveTodo)
	r.GET("/issues/:key", h.GetIssue)
	r.GET("/labels", h.GetLabels)
	r.DELETE("/todos/:id", h.DeleteTodo)
	r.POST("/todos/:id/links", h.AddLink)
	r.DELETE("/links/:id", h.DeleteLink)
	r.GET("/todos/:id/history", h.GetHistory)
	return r
}

func doRequest(t *testing.T, r *gin.Engine, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func getTodos(t *testing.T, r *gin.Engine) []domain.Todo {
	t.Helper()
	w := doRequest(t, r, http.MethodGet, "/todos", "")
	if w.Code != http.StatusOK {
		t.Fatalf("GET /todos status = %d, body = %s", w.Code, w.Body)
	}
	var todos []domain.Todo
	if err := json.Unmarshal(w.Body.Bytes(), &todos); err != nil {
		t.Fatal(err)
	}
	return todos
}

func TestTodoHandler_Description(t *testing.T) {
	r := setupTodoRouter(t)

	w := doRequest(t, r, http.MethodPost, "/todos", `{"title":"a","deadline":"2026-10-01","description":"詳細\n2行目"}`)
	if w.Code != http.StatusCreated {
		t.Fatalf("POST status = %d, body = %s", w.Code, w.Body)
	}
	// description は省略可能
	w = doRequest(t, r, http.MethodPost, "/todos", `{"title":"b","deadline":"2026-10-02"}`)
	if w.Code != http.StatusCreated {
		t.Fatalf("POST status = %d, body = %s", w.Code, w.Body)
	}

	todos := getTodos(t, r)
	if len(todos) != 2 {
		t.Fatalf("len = %d, want 2", len(todos))
	}
	if todos[0].Description != "詳細\n2行目" {
		t.Errorf("Description = %q", todos[0].Description)
	}
	if todos[1].Description != "" {
		t.Errorf("Description = %q, want empty", todos[1].Description)
	}

	w = doRequest(t, r, http.MethodPut, "/todos/1", `{"title":"a","deadline":"2026-10-01","status":"InProgress","description":"更新後"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("PUT status = %d, body = %s", w.Code, w.Body)
	}
	todos = getTodos(t, r)
	if todos[0].Description != "更新後" || todos[0].Status != domain.StatusInProgress {
		t.Errorf("todo = %+v", todos[0])
	}
}

func TestTodoHandler_DeadlineOptional(t *testing.T) {
	r := setupTodoRouter(t)

	// deadline は省略可能
	w := doRequest(t, r, http.MethodPost, "/todos", `{"title":"a"}`)
	if w.Code != http.StatusCreated {
		t.Fatalf("POST status = %d, body = %s", w.Code, w.Body)
	}
	w = doRequest(t, r, http.MethodPost, "/todos", `{"title":"b","deadline":""}`)
	if w.Code != http.StatusCreated {
		t.Fatalf("POST status = %d, body = %s", w.Code, w.Body)
	}

	todos := getTodos(t, r)
	if len(todos) != 2 {
		t.Fatalf("len = %d, want 2", len(todos))
	}
	for _, todo := range todos {
		if todo.Deadline != "" {
			t.Errorf("Deadline = %q, want empty", todo.Deadline)
		}
	}

	// 期限なしのタスクをそのまま更新できる
	w = doRequest(t, r, http.MethodPut, "/todos/1", `{"title":"a","deadline":"","status":"InProgress"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("PUT status = %d, body = %s", w.Code, w.Body)
	}
	// 期限を設定したあとで外せる
	w = doRequest(t, r, http.MethodPut, "/todos/2", `{"title":"b","deadline":"2026-10-01","status":"Open"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("PUT status = %d, body = %s", w.Code, w.Body)
	}
	if todos = getTodos(t, r); todos[1].Deadline != "2026-10-01" {
		t.Errorf("Deadline = %q, want 2026-10-01", todos[1].Deadline)
	}
	w = doRequest(t, r, http.MethodPut, "/todos/2", `{"title":"b","status":"Open"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("PUT status = %d, body = %s", w.Code, w.Body)
	}

	todos = getTodos(t, r)
	if todos[0].Deadline != "" || todos[0].Status != domain.StatusInProgress {
		t.Errorf("todo = %+v", todos[0])
	}
	if todos[1].Deadline != "" {
		t.Errorf("Deadline = %q, want empty", todos[1].Deadline)
	}

	// title は引き続き必須
	w = doRequest(t, r, http.MethodPost, "/todos", `{"deadline":"2026-10-01"}`)
	if w.Code != http.StatusBadRequest {
		t.Errorf("POST without title status = %d, want 400", w.Code)
	}
}

func TestTodoHandler_Move(t *testing.T) {
	r := setupTodoRouter(t)
	for _, title := range []string{"a", "b", "c"} {
		if w := doRequest(t, r, http.MethodPost, "/todos", `{"title":"`+title+`"}`); w.Code != http.StatusCreated {
			t.Fatalf("POST status = %d, body = %s", w.Code, w.Body)
		}
	}

	titles := func() string {
		s := ""
		for _, todo := range getTodos(t, r) {
			s += todo.Title
		}
		return s
	}

	w := doRequest(t, r, http.MethodPost, "/todos/3/move", `{"beforeId":1}`)
	if w.Code != http.StatusOK {
		t.Fatalf("move status = %d, body = %s", w.Code, w.Body)
	}
	if got := titles(); got != "cab" {
		t.Errorf("order = %s, want cab", got)
	}

	// beforeId を省略すると末尾へ移動する
	w = doRequest(t, r, http.MethodPost, "/todos/3/move", `{}`)
	if w.Code != http.StatusOK {
		t.Fatalf("move status = %d, body = %s", w.Code, w.Body)
	}
	if got := titles(); got != "abc" {
		t.Errorf("order = %s, want abc", got)
	}

	if w := doRequest(t, r, http.MethodPost, "/todos/x/move", `{}`); w.Code != http.StatusBadRequest {
		t.Errorf("invalid id status = %d, want 400", w.Code)
	}
	if w := doRequest(t, r, http.MethodPost, "/todos/99/move", `{}`); w.Code != http.StatusInternalServerError {
		t.Errorf("unknown id status = %d, want 500", w.Code)
	}
}

// GET /todos と GET /epics は作成日時・更新日時を RFC 3339 形式（UTC）で返す
func TestTodoHandler_ReturnsTimestamps(t *testing.T) {
	r := setupSpaceRouter(t)
	mustStatus(t, r, http.MethodPost, "/epics", `{"title":"E"}`, http.StatusCreated)
	mustStatus(t, r, http.MethodPost, "/todos", `{"title":"t"}`, http.StatusCreated)

	for _, path := range []string{"/todos", "/epics"} {
		w := doRequest(t, r, http.MethodGet, path, "")
		var items []map[string]any
		if err := json.Unmarshal(w.Body.Bytes(), &items); err != nil {
			t.Fatal(err)
		}
		if len(items) != 1 {
			t.Fatalf("%s = %s", path, w.Body)
		}
		for _, key := range []string{"CreatedAt", "UpdatedAt"} {
			value, _ := items[0][key].(string)
			if _, err := time.Parse(time.RFC3339, value); err != nil || !strings.HasSuffix(value, "Z") {
				t.Errorf("%s %s = %v, want RFC 3339 in UTC", path, key, items[0][key])
			}
		}
	}
}

// POST /todos は作成したタスク（採番した課題キーを含む）を返す
func TestTodoHandler_CreateReturnsTodo(t *testing.T) {
	r := setupSpaceRouter(t)
	mustStatus(t, r, http.MethodPost, "/spaces", `{"title":"Kanban","key":"KAN"}`, http.StatusCreated)
	mustStatus(t, r, http.MethodPost, "/todos", `{"title":"a","spaceId":1}`, http.StatusCreated)

	w := doRequest(t, r, http.MethodPost, "/todos", `{"title":"b","spaceId":1,"status":"InProgress"}`)
	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body)
	}
	var body struct {
		Message string
		Todo    domain.Todo `json:"todo"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Todo.ID != 2 || body.Todo.Key != "KAN-2" || body.Todo.Title != "b" || body.Todo.Status != domain.StatusInProgress {
		t.Errorf("todo = %+v", body.Todo)
	}
}

// GET /issues/:key は課題キーのタスクを返し、存在しないキーや不正な形式には 404 を返す
func TestTodoHandler_GetIssue(t *testing.T) {
	r := setupSpaceRouter(t)
	mustStatus(t, r, http.MethodPost, "/spaces", `{"title":"Kanban","key":"KAN"}`, http.StatusCreated)
	mustStatus(t, r, http.MethodPost, "/spaces", `{"title":"Other","key":"OTH"}`, http.StatusCreated)
	mustStatus(t, r, http.MethodPost, "/todos", `{"title":"kan1","spaceId":1}`, http.StatusCreated)
	mustStatus(t, r, http.MethodPost, "/todos", `{"title":"oth1","spaceId":2}`, http.StatusCreated)
	mustStatus(t, r, http.MethodPost, "/todos", `{"title":"task1"}`, http.StatusCreated)

	tests := []struct {
		key       string
		wantTitle string
	}{
		{"KAN-1", "kan1"},
		{"OTH-1", "oth1"},
		{"TASK-1", "task1"},
		{"kan-1", "kan1"},
	}
	for _, tt := range tests {
		todo := getJSON[domain.Todo](t, r, "/issues/"+tt.key)
		if todo.Title != tt.wantTitle {
			t.Errorf("GET /issues/%s Title = %q, want %q", tt.key, todo.Title, tt.wantTitle)
		}
	}

	for _, key := range []string{"KAN-2", "XYZ-1", "TASK-2", "KAN-0", "KAN", "foo"} {
		mustStatus(t, r, http.MethodGet, "/issues/"+key, "", http.StatusNotFound)
	}
}

// PUT /todos/:id は更新後のタスクを返す。スペースを変えると新しい課題キーになる
func TestTodoHandler_UpdateReturnsTodo(t *testing.T) {
	r := setupSpaceRouter(t)
	mustStatus(t, r, http.MethodPost, "/spaces", `{"title":"Kanban","key":"KAN"}`, http.StatusCreated)
	mustStatus(t, r, http.MethodPost, "/spaces", `{"title":"Other","key":"OTH"}`, http.StatusCreated)
	mustStatus(t, r, http.MethodPost, "/todos", `{"title":"a","spaceId":1}`, http.StatusCreated)

	w := doRequest(t, r, http.MethodPut, "/todos/1", `{"title":"b","status":"Closed","spaceId":2}`)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body)
	}
	var body struct {
		Todo domain.Todo `json:"todo"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Todo.ID != 1 || body.Todo.Title != "b" || body.Todo.Status != domain.StatusClosed || body.Todo.Key != "OTH-1" {
		t.Errorf("todo = %+v", body.Todo)
	}
	mustStatus(t, r, http.MethodGet, "/issues/KAN-1", "", http.StatusNotFound)
	if todo := getJSON[domain.Todo](t, r, "/issues/OTH-1"); todo.ID != 1 {
		t.Errorf("GET /issues/OTH-1 = %+v", todo)
	}
}

// 課題タイプを指定して作成・変更できる。省略すると作成時は Task、更新時は今のタイプのまま。不正な値は 400
func TestTodoHandler_IssueType(t *testing.T) {
	r := setupTodoRouter(t)

	mustStatus(t, r, http.MethodPost, "/todos", `{"title":"a"}`, http.StatusCreated)
	mustStatus(t, r, http.MethodPost, "/todos", `{"title":"b","issueType":"Bug"}`, http.StatusCreated)
	mustStatus(t, r, http.MethodPost, "/todos", `{"title":"c","issueType":"Epic"}`, http.StatusBadRequest)
	todos := getTodos(t, r)
	if len(todos) != 2 || todos[0].IssueType != domain.IssueTypeTask || todos[1].IssueType != domain.IssueTypeBug {
		t.Fatalf("todos = %+v", todos)
	}

	mustStatus(t, r, http.MethodPut, "/todos/1", `{"title":"a","status":"Open","issueType":"Story"}`, http.StatusOK)
	// ドラッグでの移動のように課題タイプを送らない更新では、タイプは変わらない
	mustStatus(t, r, http.MethodPut, "/todos/2", `{"title":"b","status":"Closed"}`, http.StatusOK)
	mustStatus(t, r, http.MethodPut, "/todos/2", `{"title":"b","status":"Closed","issueType":"bug"}`, http.StatusBadRequest)
	todos = getTodos(t, r)
	if todos[0].IssueType != domain.IssueTypeStory || todos[1].IssueType != domain.IssueTypeBug || todos[1].Status != domain.StatusClosed {
		t.Errorf("todos = %+v", todos)
	}
}

// 優先度を指定して作成・変更できる。省略すると作成時は Medium、更新時は今の優先度のまま。不正な値は 400
func TestTodoHandler_Priority(t *testing.T) {
	r := setupTodoRouter(t)

	mustStatus(t, r, http.MethodPost, "/todos", `{"title":"a"}`, http.StatusCreated)
	mustStatus(t, r, http.MethodPost, "/todos", `{"title":"b","priority":"Highest"}`, http.StatusCreated)
	mustStatus(t, r, http.MethodPost, "/todos", `{"title":"c","priority":"Urgent"}`, http.StatusBadRequest)
	todos := getTodos(t, r)
	if len(todos) != 2 || todos[0].Priority != domain.PriorityMedium || todos[1].Priority != domain.PriorityHighest {
		t.Fatalf("todos = %+v", todos)
	}

	mustStatus(t, r, http.MethodPut, "/todos/1", `{"title":"a","status":"Open","priority":"Lowest"}`, http.StatusOK)
	// ドラッグでの移動のように優先度を送らない更新では、優先度は変わらない（課題タイプだけ変えても同じ）
	mustStatus(t, r, http.MethodPut, "/todos/2", `{"title":"b","status":"Closed","issueType":"Bug"}`, http.StatusOK)
	mustStatus(t, r, http.MethodPut, "/todos/2", `{"title":"b","status":"Closed","priority":"high"}`, http.StatusBadRequest)
	todos = getTodos(t, r)
	if todos[0].Priority != domain.PriorityLowest || todos[1].Priority != domain.PriorityHighest || todos[1].IssueType != domain.IssueTypeBug {
		t.Errorf("todos = %+v", todos)
	}
}

// ラベルを付けて作成し、付け外しできる。省略した更新では変わらない。不正なラベルは 400。GET /labels は使われているラベルを返す
func TestTodoHandler_Labels(t *testing.T) {
	r := setupTodoRouter(t)

	mustStatus(t, r, http.MethodPost, "/todos", `{"title":"a","labels":["ui"," backend ","UI"]}`, http.StatusCreated)
	mustStatus(t, r, http.MethodPost, "/todos", `{"title":"b"}`, http.StatusCreated)
	mustStatus(t, r, http.MethodPost, "/todos", `{"title":"c","labels":["two words"]}`, http.StatusBadRequest)
	todos := getTodos(t, r)
	if len(todos) != 2 || strings.Join(todos[0].Labels, ",") != "backend,ui" || todos[1].Labels == nil || len(todos[1].Labels) != 0 {
		t.Fatalf("todos = %+v", todos)
	}
	if labels := getJSON[[]string](t, r, "/labels"); strings.Join(labels, ",") != "backend,ui" {
		t.Errorf("GET /labels = %q", labels)
	}

	// 他のタスクで使ったラベルを付ける
	mustStatus(t, r, http.MethodPut, "/todos/2", `{"title":"b","status":"Open","labels":["ui","不具合"]}`, http.StatusOK)
	// ラベルを送らない更新では変わらない
	mustStatus(t, r, http.MethodPut, "/todos/2", `{"title":"b","status":"Closed"}`, http.StatusOK)
	// 空の配列で外せる
	mustStatus(t, r, http.MethodPut, "/todos/1", `{"title":"a","status":"Open","labels":[]}`, http.StatusOK)
	mustStatus(t, r, http.MethodPut, "/todos/1", `{"title":"a","status":"Open","labels":[""]}`, http.StatusBadRequest)
	todos = getTodos(t, r)
	if len(todos[0].Labels) != 0 || strings.Join(todos[1].Labels, ",") != "ui,不具合" || todos[1].Status != domain.StatusClosed {
		t.Errorf("todos = %+v", todos)
	}
	if labels := getJSON[[]string](t, r, "/labels"); strings.Join(labels, ",") != "ui,不具合" {
		t.Errorf("GET /labels = %q", labels)
	}
}

// ストーリーポイントを設定・解除できる。作成で省略すると未設定、更新で省略すると今のまま、null で未設定。不正な値は 400
func TestTodoHandler_StoryPoints(t *testing.T) {
	r := setupTodoRouter(t)

	mustStatus(t, r, http.MethodPost, "/todos", `{"title":"a"}`, http.StatusCreated)
	mustStatus(t, r, http.MethodPost, "/todos", `{"title":"b","storyPoints":0.5}`, http.StatusCreated)
	mustStatus(t, r, http.MethodPost, "/todos", `{"title":"c","storyPoints":-1}`, http.StatusBadRequest)
	mustStatus(t, r, http.MethodPost, "/todos", `{"title":"c","storyPoints":1.234}`, http.StatusBadRequest)
	todos := getTodos(t, r)
	if len(todos) != 2 || todos[0].StoryPoints != nil || todos[1].StoryPoints == nil || *todos[1].StoryPoints != 0.5 {
		t.Fatalf("todos = %+v", todos)
	}
	// 未設定は JSON で null になる
	if w := doRequest(t, r, http.MethodGet, "/todos", ""); !strings.Contains(w.Body.String(), `"StoryPoints":null`) || !strings.Contains(w.Body.String(), `"StoryPoints":0.5`) {
		t.Errorf("GET /todos body = %s", w.Body)
	}

	mustStatus(t, r, http.MethodPut, "/todos/1", `{"title":"a","status":"Open","storyPoints":8}`, http.StatusOK)
	mustStatus(t, r, http.MethodPut, "/todos/1", `{"title":"a","status":"Closed"}`, http.StatusOK)
	mustStatus(t, r, http.MethodPut, "/todos/2", `{"title":"b","status":"Open","storyPoints":null}`, http.StatusOK)
	mustStatus(t, r, http.MethodPut, "/todos/2", `{"title":"b","status":"Open","storyPoints":"3"}`, http.StatusBadRequest)
	mustStatus(t, r, http.MethodPut, "/todos/2", `{"title":"b","status":"Open","storyPoints":1000000}`, http.StatusBadRequest)
	todos = getTodos(t, r)
	if todos[0].StoryPoints == nil || *todos[0].StoryPoints != 8 || todos[0].Status != domain.StatusClosed || todos[1].StoryPoints != nil {
		t.Errorf("todos = %+v", todos)
	}
	// 0 は未設定と区別する
	mustStatus(t, r, http.MethodPut, "/todos/2", `{"title":"b","status":"Open","storyPoints":0}`, http.StatusOK)
	if todo := getJSON[domain.Todo](t, r, "/issues/TASK-2"); todo.StoryPoints == nil || *todo.StoryPoints != 0 {
		t.Errorf("TASK-2 StoryPoints = %v, want 0", todo.StoryPoints)
	}
}

// サブタスクは親課題を指定して作り、親と同じスペース・エピックに置かれる。サブタスクの下には作れない
func TestTodoHandler_Subtasks(t *testing.T) {
	r := setupSpaceRouter(t)
	mustStatus(t, r, http.MethodPost, "/spaces", `{"title":"Kanban","key":"KAN"}`, http.StatusCreated)
	mustStatus(t, r, http.MethodPost, "/spaces", `{"title":"Other","key":"OTH"}`, http.StatusCreated)
	mustStatus(t, r, http.MethodPost, "/epics", `{"title":"e1","spaceId":1}`, http.StatusCreated)
	mustStatus(t, r, http.MethodPost, "/epics", `{"title":"e2","spaceId":2}`, http.StatusCreated)
	mustStatus(t, r, http.MethodPost, "/todos", `{"title":"parent","spaceId":1,"epicId":1}`, http.StatusCreated)

	// スペース・エピックを送らなくても親と同じになる
	w := doRequest(t, r, http.MethodPost, "/todos", `{"title":"sub1","parentId":1}`)
	if w.Code != http.StatusCreated {
		t.Fatalf("POST subtask status = %d, body = %s", w.Code, w.Body)
	}
	var created struct {
		Todo domain.Todo `json:"todo"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	sub := created.Todo
	if sub.ParentID != 1 || sub.IssueType != domain.IssueTypeSubtask || sub.SpaceID != 1 || sub.EpicID != 1 || sub.Key != "KAN-2" {
		t.Errorf("subtask = %+v", sub)
	}
	mustStatus(t, r, http.MethodPost, "/todos", `{"title":"sub2","parentId":1,"issueType":"Subtask"}`, http.StatusCreated)

	mustStatus(t, r, http.MethodPost, "/todos", `{"title":"x","parentId":99}`, http.StatusBadRequest)
	mustStatus(t, r, http.MethodPost, "/todos", `{"title":"x","parentId":2}`, http.StatusBadRequest)
	mustStatus(t, r, http.MethodPost, "/todos", `{"title":"x","issueType":"Subtask"}`, http.StatusBadRequest)
	mustStatus(t, r, http.MethodPost, "/todos", `{"title":"x","parentId":1,"issueType":"Bug"}`, http.StatusBadRequest)

	// サブタスクのステータスを変えられる。スペース・エピックを送っても親と同じまま。タイプは変えられない
	mustStatus(t, r, http.MethodPut, "/todos/2", `{"title":"sub1","status":"Closed","spaceId":0,"epicId":0}`, http.StatusOK)
	mustStatus(t, r, http.MethodPut, "/todos/2", `{"title":"sub1","status":"Closed","issueType":"Task"}`, http.StatusBadRequest)
	mustStatus(t, r, http.MethodPut, "/todos/1", `{"title":"parent","status":"Open","spaceId":1,"epicId":1,"issueType":"Subtask"}`, http.StatusBadRequest)
	got := getJSON[domain.Todo](t, r, "/issues/KAN-2")
	if got.Status != domain.StatusClosed || got.SpaceID != 1 || got.EpicID != 1 || got.ParentID != 1 {
		t.Errorf("subtask after update = %+v", got)
	}

	// 親を別のスペース・エピックへ移すと、サブタスクも移って課題キーが変わる
	mustStatus(t, r, http.MethodPut, "/todos/1", `{"title":"parent","status":"Open","spaceId":2,"epicId":2}`, http.StatusOK)
	todos := getTodos(t, r)
	for _, todo := range todos {
		if todo.SpaceID != 2 || todo.EpicID != 2 {
			t.Errorf("after moving parent: %s space = %d, epic = %d, want 2, 2", todo.Title, todo.SpaceID, todo.EpicID)
		}
	}
	if keys := todos[0].Key + "," + todos[1].Key + "," + todos[2].Key; keys != "OTH-1,OTH-2,OTH-3" {
		t.Errorf("keys = %s", keys)
	}
}

// 親課題を削除すると、サブタスクも削除される。サブタスクだけを削除しても親は残る
func TestTodoHandler_DeleteParentDeletesSubtasks(t *testing.T) {
	r := setupTodoRouter(t)

	mustStatus(t, r, http.MethodPost, "/todos", `{"title":"parent"}`, http.StatusCreated)
	mustStatus(t, r, http.MethodPost, "/todos", `{"title":"sub1","parentId":1,"labels":["ui"]}`, http.StatusCreated)
	mustStatus(t, r, http.MethodPost, "/todos", `{"title":"sub2","parentId":1}`, http.StatusCreated)
	mustStatus(t, r, http.MethodPost, "/todos", `{"title":"other"}`, http.StatusCreated)

	mustStatus(t, r, http.MethodDelete, "/todos/3", "", http.StatusOK)
	if todos := getTodos(t, r); len(todos) != 3 {
		t.Fatalf("after deleting a subtask: %d todos, want 3", len(todos))
	}
	mustStatus(t, r, http.MethodDelete, "/todos/1", "", http.StatusOK)
	todos := getTodos(t, r)
	if len(todos) != 1 || todos[0].Title != "other" {
		t.Errorf("after deleting the parent: todos = %+v", todos)
	}
	// 消したサブタスクのラベルは候補に残らない
	if labels := getJSON[[]string](t, r, "/labels"); len(labels) != 0 {
		t.Errorf("GET /labels = %q, want empty", labels)
	}
}

// リンクを追加すると両方の課題に、それぞれから見た名前で表示される
func TestTodoHandler_Links(t *testing.T) {
	r := setupTodoRouter(t)

	mustStatus(t, r, http.MethodPost, "/todos", `{"title":"a"}`, http.StatusCreated)
	mustStatus(t, r, http.MethodPost, "/todos", `{"title":"b"}`, http.StatusCreated)
	mustStatus(t, r, http.MethodPost, "/todos", `{"title":"c"}`, http.StatusCreated)

	// a is blocked by b（保存は b blocks a）
	w := doRequest(t, r, http.MethodPost, "/todos/1/links", `{"relation":"is blocked by","targetId":2}`)
	if w.Code != http.StatusCreated {
		t.Fatalf("POST links status = %d, body = %s", w.Code, w.Body)
	}
	var created struct{ Todo domain.Todo }
	if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	if len(created.Todo.Links) != 1 || created.Todo.Links[0].Relation != "is blocked by" || created.Todo.Links[0].Issue.Key != "TASK-2" {
		t.Errorf("created todo links = %+v", created.Todo.Links)
	}
	mustStatus(t, r, http.MethodPost, "/todos/1/links", `{"relation":"relates to","targetId":3}`, http.StatusCreated)
	mustStatus(t, r, http.MethodPost, "/todos/1/links", `{"relation":"duplicates","targetId":3}`, http.StatusCreated)

	todos := getTodos(t, r)
	relations := func(todo domain.Todo) string {
		s := []string{}
		for _, link := range todo.Links {
			s = append(s, link.Relation+":"+link.Issue.Title)
		}
		return strings.Join(s, ",")
	}
	// 種類の順（is blocked by → relates to → duplicates）に並ぶ
	if got := relations(todos[0]); got != "is blocked by:b,relates to:c,duplicates:c" {
		t.Errorf("a links = %s", got)
	}
	if got := relations(todos[1]); got != "blocks:a" {
		t.Errorf("b links = %s", got)
	}
	if got := relations(todos[2]); got != "relates to:a,is duplicated by:a" {
		t.Errorf("c links = %s", got)
	}
	// GET /issues/:key でもリンクが返る
	if issue := getJSON[domain.Todo](t, r, "/issues/TASK-2"); relations(issue) != "blocks:a" {
		t.Errorf("GET /issues/TASK-2 links = %s", relations(issue))
	}

	// 同じリンクは作れない。relates to は向きを区別しない
	mustStatus(t, r, http.MethodPost, "/todos/2/links", `{"relation":"blocks","targetId":1}`, http.StatusConflict)
	mustStatus(t, r, http.MethodPost, "/todos/3/links", `{"relation":"relates to","targetId":1}`, http.StatusConflict)
	// 逆向きの blocks は別のリンク
	mustStatus(t, r, http.MethodPost, "/todos/2/links", `{"relation":"is blocked by","targetId":1}`, http.StatusCreated)

	mustStatus(t, r, http.MethodPost, "/todos/1/links", `{"relation":"fixes","targetId":2}`, http.StatusBadRequest)
	mustStatus(t, r, http.MethodPost, "/todos/1/links", `{"relation":"blocks","targetId":1}`, http.StatusBadRequest)
	mustStatus(t, r, http.MethodPost, "/todos/1/links", `{"relation":"blocks","targetId":99}`, http.StatusBadRequest)
	mustStatus(t, r, http.MethodPost, "/todos/99/links", `{"relation":"blocks","targetId":1}`, http.StatusNotFound)

	// リンクを削除すると両方から消える
	linkID := todos[1].Links[0].ID
	mustStatus(t, r, http.MethodDelete, "/links/"+strconv.Itoa(linkID), "", http.StatusOK)
	mustStatus(t, r, http.MethodDelete, "/links/"+strconv.Itoa(linkID), "", http.StatusNotFound)
	todos = getTodos(t, r)
	if got := relations(todos[0]); got != "blocks:b,relates to:c,duplicates:c" {
		t.Errorf("a links after delete = %s", got)
	}
}

// 課題を削除すると、その課題（とサブタスク）のリンクも消える
func TestTodoHandler_DeleteTodoDeletesLinks(t *testing.T) {
	repos := newTestRepos(t)
	r := newTodoRouter(repos)

	mustStatus(t, r, http.MethodPost, "/todos", `{"title":"a"}`, http.StatusCreated)
	mustStatus(t, r, http.MethodPost, "/todos", `{"title":"b"}`, http.StatusCreated)
	mustStatus(t, r, http.MethodPost, "/todos", `{"title":"sub","parentId":2}`, http.StatusCreated)
	mustStatus(t, r, http.MethodPost, "/todos", `{"title":"c"}`, http.StatusCreated)
	mustStatus(t, r, http.MethodPost, "/todos/1/links", `{"relation":"blocks","targetId":2}`, http.StatusCreated)
	mustStatus(t, r, http.MethodPost, "/todos/3/links", `{"relation":"clones","targetId":4}`, http.StatusCreated)
	mustStatus(t, r, http.MethodPost, "/todos/1/links", `{"relation":"relates to","targetId":4}`, http.StatusCreated)

	mustStatus(t, r, http.MethodDelete, "/todos/2", "", http.StatusOK)
	todos := getTodos(t, r)
	if len(todos) != 2 {
		t.Fatalf("todos = %+v", todos)
	}
	for _, todo := range todos {
		if len(todo.Links) != 1 || todo.Links[0].Relation != "relates to" {
			t.Errorf("%s links = %+v", todo.Title, todo.Links)
		}
	}
	// 削除した課題のリンクはテーブルにも残らない
	links, err := repos.link.FindAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(links) != 1 || links[0].SourceID != 1 || links[0].TargetID != 4 {
		t.Errorf("stored links = %+v", links)
	}
}
