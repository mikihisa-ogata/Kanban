package handler

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"

	"todo-api/internal/domain"
	"todo-api/internal/repository"
	"todo-api/internal/service"
)

func setupSpaceRouter(t *testing.T) *gin.Engine {
	t.Helper()
	r := setupTodoRouter(t)

	todoRepo := repository.NewTodoRepository()
	epicRepo := repository.NewEpicRepository()
	spaceRepo := repository.NewSpaceRepository()
	eh := NewEpicHandler(service.NewEpicService(epicRepo, todoRepo, spaceRepo))
	sh := NewSpaceHandler(service.NewSpaceService(spaceRepo, epicRepo, todoRepo))

	r.GET("/epics", eh.GetEpics)
	r.POST("/epics", eh.CreateEpic)
	r.PUT("/epics/:id", eh.UpdateEpic)
	r.GET("/spaces", sh.GetSpaces)
	r.POST("/spaces", sh.CreateSpace)
	r.DELETE("/spaces/:id", sh.DeleteSpace)
	return r
}

func mustStatus(t *testing.T, r *gin.Engine, method, path, body string, want int) {
	t.Helper()
	if w := doRequest(t, r, method, path, body); w.Code != want {
		t.Fatalf("%s %s %s status = %d, want %d, body = %s", method, path, body, w.Code, want, w.Body)
	}
}

func getJSON[T any](t *testing.T, r *gin.Engine, path string) T {
	t.Helper()
	w := doRequest(t, r, http.MethodGet, path, "")
	if w.Code != http.StatusOK {
		t.Fatalf("GET %s status = %d, body = %s", path, w.Code, w.Body)
	}
	var v T
	if err := json.Unmarshal(w.Body.Bytes(), &v); err != nil {
		t.Fatal(err)
	}
	return v
}

func TestSpace_CreateAssignAndValidate(t *testing.T) {
	r := setupSpaceRouter(t)

	if spaces := getJSON[[]domain.Space](t, r, "/spaces"); len(spaces) != 0 {
		t.Fatalf("spaces = %+v, want empty", spaces)
	}
	mustStatus(t, r, http.MethodPost, "/spaces", `{"title":"プロダクトA"}`, http.StatusCreated)
	mustStatus(t, r, http.MethodPost, "/spaces", `{"title":"プロダクトB"}`, http.StatusCreated)
	mustStatus(t, r, http.MethodPost, "/spaces", `{}`, http.StatusBadRequest)
	spaces := getJSON[[]domain.Space](t, r, "/spaces")
	if len(spaces) != 2 || spaces[0] != (domain.Space{ID: 1, Title: "プロダクトA"}) || spaces[1].ID != 2 {
		t.Fatalf("spaces = %+v", spaces)
	}

	// エピックはスペースに属する（省略時は未割り当て）
	mustStatus(t, r, http.MethodPost, "/epics", `{"title":"E1","spaceId":1}`, http.StatusCreated)
	mustStatus(t, r, http.MethodPost, "/epics", `{"title":"E0"}`, http.StatusCreated)
	mustStatus(t, r, http.MethodPost, "/epics", `{"title":"X","spaceId":99}`, http.StatusBadRequest)
	epics := getJSON[[]domain.Epic](t, r, "/epics")
	if len(epics) != 2 || epics[0].SpaceID != 1 || epics[1].SpaceID != 0 {
		t.Fatalf("epics = %+v", epics)
	}

	// タスクはスペースに属し、エピックと同じスペースでなければならない
	mustStatus(t, r, http.MethodPost, "/todos", `{"title":"t1","spaceId":1,"epicId":1}`, http.StatusCreated)
	mustStatus(t, r, http.MethodPost, "/todos", `{"title":"t2","spaceId":2}`, http.StatusCreated)
	mustStatus(t, r, http.MethodPost, "/todos", `{"title":"t3"}`, http.StatusCreated)
	mustStatus(t, r, http.MethodPost, "/todos", `{"title":"x","spaceId":99}`, http.StatusBadRequest)
	mustStatus(t, r, http.MethodPost, "/todos", `{"title":"x","spaceId":2,"epicId":1}`, http.StatusBadRequest)
	mustStatus(t, r, http.MethodPost, "/todos", `{"title":"x","epicId":1}`, http.StatusBadRequest)
	mustStatus(t, r, http.MethodPost, "/todos", `{"title":"x","spaceId":1,"epicId":99}`, http.StatusBadRequest)
	mustStatus(t, r, http.MethodPut, "/todos/2", `{"title":"t2","status":"Open","spaceId":2,"epicId":1}`, http.StatusBadRequest)
	// タスクを別のスペースへ移す
	mustStatus(t, r, http.MethodPut, "/todos/3", `{"title":"t3","status":"Open","spaceId":2}`, http.StatusOK)

	todos := getTodos(t, r)
	if len(todos) != 3 {
		t.Fatalf("todos = %+v", todos)
	}
	got := [3]int{todos[0].SpaceID, todos[1].SpaceID, todos[2].SpaceID}
	if got != [3]int{1, 2, 2} {
		t.Errorf("SpaceIDs = %v, want [1 2 2]", got)
	}
	if todos[0].EpicID != 1 {
		t.Errorf("EpicID = %d, want 1", todos[0].EpicID)
	}
}

func TestSpace_MoveEpicMovesChildTodos(t *testing.T) {
	r := setupSpaceRouter(t)

	mustStatus(t, r, http.MethodPost, "/spaces", `{"title":"A"}`, http.StatusCreated)
	mustStatus(t, r, http.MethodPost, "/spaces", `{"title":"B"}`, http.StatusCreated)
	mustStatus(t, r, http.MethodPost, "/epics", `{"title":"E"}`, http.StatusCreated)
	mustStatus(t, r, http.MethodPost, "/todos", `{"title":"child","epicId":1}`, http.StatusCreated)
	mustStatus(t, r, http.MethodPost, "/todos", `{"title":"other"}`, http.StatusCreated)

	mustStatus(t, r, http.MethodPut, "/epics/1", `{"title":"E","spaceId":2}`, http.StatusOK)
	mustStatus(t, r, http.MethodPut, "/epics/1", `{"title":"E","spaceId":99}`, http.StatusBadRequest)
	mustStatus(t, r, http.MethodPut, "/epics/99", `{"title":"E","spaceId":1}`, http.StatusNotFound)
	mustStatus(t, r, http.MethodPut, "/epics/1", `{"spaceId":1}`, http.StatusBadRequest)

	epics := getJSON[[]domain.Epic](t, r, "/epics")
	if len(epics) != 1 || epics[0] != (domain.Epic{ID: 1, Title: "E", SpaceID: 2}) {
		t.Fatalf("epics = %+v", epics)
	}
	todos := getTodos(t, r)
	if todos[0].SpaceID != 2 || todos[0].EpicID != 1 {
		t.Errorf("child = %+v, want SpaceID 2 / EpicID 1", todos[0])
	}
	if todos[1].SpaceID != 0 {
		t.Errorf("other = %+v, want SpaceID 0", todos[1])
	}
}

func TestSpace_DeleteUnassignsEpicsAndTodos(t *testing.T) {
	r := setupSpaceRouter(t)

	mustStatus(t, r, http.MethodPost, "/spaces", `{"title":"A"}`, http.StatusCreated)
	mustStatus(t, r, http.MethodPost, "/spaces", `{"title":"B"}`, http.StatusCreated)
	mustStatus(t, r, http.MethodPost, "/epics", `{"title":"EA","spaceId":1}`, http.StatusCreated)
	mustStatus(t, r, http.MethodPost, "/epics", `{"title":"EB","spaceId":2}`, http.StatusCreated)
	mustStatus(t, r, http.MethodPost, "/todos", `{"title":"a-child","spaceId":1,"epicId":1}`, http.StatusCreated)
	mustStatus(t, r, http.MethodPost, "/todos", `{"title":"a","spaceId":1}`, http.StatusCreated)
	mustStatus(t, r, http.MethodPost, "/todos", `{"title":"b","spaceId":2,"epicId":2}`, http.StatusCreated)

	mustStatus(t, r, http.MethodDelete, "/spaces/1", "", http.StatusOK)
	mustStatus(t, r, http.MethodDelete, "/spaces/1", "", http.StatusInternalServerError)
	mustStatus(t, r, http.MethodDelete, "/spaces/abc", "", http.StatusBadRequest)

	spaces := getJSON[[]domain.Space](t, r, "/spaces")
	if len(spaces) != 1 || spaces[0].ID != 2 {
		t.Fatalf("spaces = %+v", spaces)
	}
	epics := getJSON[[]domain.Epic](t, r, "/epics")
	if epics[0].SpaceID != 0 || epics[1].SpaceID != 2 {
		t.Errorf("epics = %+v", epics)
	}
	todos := getTodos(t, r)
	// エピックとの紐付けは残り、スペースだけ未割り当てになる
	if todos[0].SpaceID != 0 || todos[0].EpicID != 1 || todos[1].SpaceID != 0 || todos[2].SpaceID != 2 || todos[2].EpicID != 2 {
		t.Errorf("todos = %+v", todos)
	}
	// 未割り当てに戻ったエピックとタスクの組み合わせはそのまま更新できる
	mustStatus(t, r, http.MethodPut, "/todos/1", `{"title":"a-child","status":"Closed","epicId":1}`, http.StatusOK)
}
