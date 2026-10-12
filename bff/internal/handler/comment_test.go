package handler

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"todo-api/internal/domain"
	"todo-api/internal/service"
)

// newCommentRouter は課題とコメントの API を持つルーターを返す
func newCommentRouter(t *testing.T) *gin.Engine {
	t.Helper()
	db := newTestRepos(t)
	r := newTodoRouter(db)
	h := NewCommentHandler(service.NewCommentService(db.comment, db.todo, db.watcher, db.notification))
	r.GET("/todos/:id/comments", h.GetComments)
	r.POST("/todos/:id/comments", h.CreateComment)
	r.PUT("/comments/:id", h.UpdateComment)
	r.DELETE("/comments/:id", h.DeleteComment)
	return r
}

func getComments(t *testing.T, r *gin.Engine, path string) []domain.Comment {
	t.Helper()
	return getJSON[[]domain.Comment](t, r, path)
}

func TestCommentHandler_CRUD(t *testing.T) {
	r := newCommentRouter(t)
	mustStatus(t, r, http.MethodPost, "/todos", `{"title":"a"}`, http.StatusCreated)
	mustStatus(t, r, http.MethodPost, "/todos", `{"title":"b"}`, http.StatusCreated)

	if comments := getComments(t, r, "/todos/1/comments"); len(comments) != 0 {
		t.Fatalf("comments = %+v, want empty", comments)
	}

	w := doRequest(t, r, http.MethodPost, "/todos/1/comments", `{"body":"**最初**のコメント"}`)
	if w.Code != http.StatusCreated {
		t.Fatalf("POST comment status = %d, body = %s", w.Code, w.Body)
	}
	var created struct{ Comment domain.Comment }
	if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	if created.Comment.ID == 0 || created.Comment.TodoID != 1 || created.Comment.Body != "**最初**のコメント" ||
		created.Comment.CreatedAt.IsZero() || created.Comment.UpdatedAt != nil {
		t.Errorf("created comment = %+v", created.Comment)
	}
	mustStatus(t, r, http.MethodPost, "/todos/1/comments", `{"body":"2つ目"}`, http.StatusCreated)
	mustStatus(t, r, http.MethodPost, "/todos/2/comments", `{"body":"別の課題"}`, http.StatusCreated)

	comments := getComments(t, r, "/todos/1/comments")
	if len(comments) != 2 || comments[0].Body != "**最初**のコメント" || comments[1].Body != "2つ目" {
		t.Fatalf("comments = %+v", comments)
	}

	// 課題にコメント数が付く
	todos := getTodos(t, r)
	if todos[0].CommentCount != 2 || todos[1].CommentCount != 1 {
		t.Errorf("comment counts = %d, %d", todos[0].CommentCount, todos[1].CommentCount)
	}
	if issue := getJSON[domain.Todo](t, r, "/issues/TASK-1"); issue.CommentCount != 2 {
		t.Errorf("GET /issues comment count = %d", issue.CommentCount)
	}

	// 編集すると編集日時が付く
	w = doRequest(t, r, http.MethodPut, "/comments/1", `{"body":"編集後"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("PUT comment status = %d, body = %s", w.Code, w.Body)
	}
	var updated struct{ Comment domain.Comment }
	if err := json.Unmarshal(w.Body.Bytes(), &updated); err != nil {
		t.Fatal(err)
	}
	if updated.Comment.Body != "編集後" || updated.Comment.UpdatedAt == nil || !updated.Comment.CreatedAt.Equal(created.Comment.CreatedAt) {
		t.Errorf("updated comment = %+v", updated.Comment)
	}

	mustStatus(t, r, http.MethodDelete, "/comments/1", "", http.StatusOK)
	if comments := getComments(t, r, "/todos/1/comments"); len(comments) != 1 || comments[0].Body != "2つ目" {
		t.Errorf("comments after delete = %+v", comments)
	}
	if todos := getTodos(t, r); todos[0].CommentCount != 1 {
		t.Errorf("comment count after delete = %d", todos[0].CommentCount)
	}
}

func TestCommentHandler_Errors(t *testing.T) {
	r := newCommentRouter(t)
	mustStatus(t, r, http.MethodPost, "/todos", `{"title":"a"}`, http.StatusCreated)
	mustStatus(t, r, http.MethodPost, "/todos/1/comments", `{"body":"ok"}`, http.StatusCreated)

	mustStatus(t, r, http.MethodPost, "/todos/1/comments", `{"body":"  \n "}`, http.StatusBadRequest)
	mustStatus(t, r, http.MethodPost, "/todos/1/comments", `{}`, http.StatusBadRequest)
	mustStatus(t, r, http.MethodPost, "/todos/1/comments", `{"body":"`+strings.Repeat("a", 10001)+`"}`, http.StatusBadRequest)
	mustStatus(t, r, http.MethodPost, "/todos/99/comments", `{"body":"x"}`, http.StatusNotFound)
	mustStatus(t, r, http.MethodGet, "/todos/99/comments", "", http.StatusNotFound)
	mustStatus(t, r, http.MethodGet, "/todos/x/comments", "", http.StatusBadRequest)
	mustStatus(t, r, http.MethodPut, "/comments/1", `{"body":""}`, http.StatusBadRequest)
	mustStatus(t, r, http.MethodPut, "/comments/99", `{"body":"x"}`, http.StatusNotFound)
	mustStatus(t, r, http.MethodDelete, "/comments/99", "", http.StatusNotFound)
}

// 課題を削除すると、その課題とサブタスクのコメントも消える
func TestCommentHandler_DeleteTodoDeletesComments(t *testing.T) {
	db := newTestRepos(t)
	r := newTodoRouter(db)
	h := NewCommentHandler(service.NewCommentService(db.comment, db.todo, db.watcher, db.notification))
	r.POST("/todos/:id/comments", h.CreateComment)

	mustStatus(t, r, http.MethodPost, "/todos", `{"title":"parent"}`, http.StatusCreated)
	mustStatus(t, r, http.MethodPost, "/todos", `{"title":"sub","parentId":1}`, http.StatusCreated)
	mustStatus(t, r, http.MethodPost, "/todos", `{"title":"other"}`, http.StatusCreated)
	for _, id := range []string{"1", "2", "3"} {
		mustStatus(t, r, http.MethodPost, "/todos/"+id+"/comments", `{"body":"c"}`, http.StatusCreated)
	}

	mustStatus(t, r, http.MethodDelete, "/todos/1", "", http.StatusOK)
	for id, want := range map[int]int{1: 0, 2: 0, 3: 1} {
		comments, err := db.comment.FindByTodo(id)
		if err != nil {
			t.Fatal(err)
		}
		if len(comments) != want {
			t.Errorf("todo %d: %d comments, want %d", id, len(comments), want)
		}
	}
}
