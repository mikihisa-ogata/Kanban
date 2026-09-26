package handler

import (
	"net/http"
	"testing"

	"todo-api/internal/domain"
)

func TestEpic_RenameKeepsSpaceAndChildTodos(t *testing.T) {
	r := setupSpaceRouter(t)

	mustStatus(t, r, http.MethodPost, "/spaces", `{"title":"S1"}`, http.StatusCreated)
	mustStatus(t, r, http.MethodPost, "/epics", `{"title":"古い名前","spaceId":1}`, http.StatusCreated)
	mustStatus(t, r, http.MethodPost, "/todos", `{"title":"t1","spaceId":1,"epicId":1}`, http.StatusCreated)

	// 名前だけを変える（スペースは今のまま送る）
	mustStatus(t, r, http.MethodPut, "/epics/1", `{"title":"新しい名前, \"引用\"","spaceId":1}`, http.StatusOK)
	epics := getJSON[[]domain.Epic](t, r, "/epics")
	if len(epics) != 1 || epics[0] != (domain.Epic{ID: 1, Title: `新しい名前, "引用"`, SpaceID: 1}) {
		t.Fatalf("epics = %+v", epics)
	}
	todos := getTodos(t, r)
	if len(todos) != 1 || todos[0].EpicID != 1 || todos[0].SpaceID != 1 {
		t.Fatalf("todos = %+v", todos)
	}

	// 空の名前や存在しないエピックは受け付けない
	mustStatus(t, r, http.MethodPut, "/epics/1", `{"title":"","spaceId":1}`, http.StatusBadRequest)
	mustStatus(t, r, http.MethodPut, "/epics/99", `{"title":"x","spaceId":1}`, http.StatusNotFound)
}
