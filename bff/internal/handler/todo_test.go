package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"todo-api/internal/domain"
	"todo-api/internal/repository"
	"todo-api/internal/service"
)

func setupTodoRouter(t *testing.T) *gin.Engine {
	t.Helper()
	dir := t.TempDir()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(wd); err != nil {
			t.Fatal(err)
		}
	})

	gin.SetMode(gin.TestMode)
	todoRepo := repository.NewTodoRepository()
	epicRepo := repository.NewEpicRepository()
	h := NewTodoHandler(service.NewTodoService(todoRepo, epicRepo))

	r := gin.New()
	r.GET("/todos", h.GetTodos)
	r.POST("/todos", h.CreateTodo)
	r.PUT("/todos/:id", h.UpdateTodo)
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
