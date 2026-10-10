package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
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
	todo  repository.TodoRepository
	epic  repository.EpicRepository
	space repository.SpaceRepository
}

// newTestRepos は空のテスト用 MySQL を使うリポジトリを返す（TEST_DB_DSN が未設定ならスキップ）
func newTestRepos(t *testing.T) testRepos {
	t.Helper()
	db := dbtest.Open(t, "handler")
	return testRepos{
		todo:  repository.NewTodoMySQLRepository(db),
		epic:  repository.NewEpicMySQLRepository(db),
		space: repository.NewSpaceMySQLRepository(db),
	}
}

func setupTodoRouter(t *testing.T) *gin.Engine {
	t.Helper()
	return newTodoRouter(newTestRepos(t))
}

func newTodoRouter(repos testRepos) *gin.Engine {
	gin.SetMode(gin.TestMode)
	h := NewTodoHandler(service.NewTodoService(repos.todo, repos.epic, repos.space))

	r := gin.New()
	r.GET("/todos", h.GetTodos)
	r.POST("/todos", h.CreateTodo)
	r.PUT("/todos/:id", h.UpdateTodo)
	r.POST("/todos/:id/move", h.MoveTodo)
	r.GET("/issues/:key", h.GetIssue)
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
