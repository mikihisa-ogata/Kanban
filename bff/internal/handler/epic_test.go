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
	if len(epics) != 1 || epicWithoutTimes(epics[0]) != (domain.Epic{ID: 1, Title: `新しい名前, "引用"`, SpaceID: 1}) {
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

func TestEpic_DatesAndColor(t *testing.T) {
	r := setupSpaceRouter(t)

	mustStatus(t, r, http.MethodPost, "/spaces", `{"title":"S1"}`, http.StatusCreated)
	// 作成で日付と色を指定でき、省略すると未設定になる
	mustStatus(t, r, http.MethodPost, "/epics", `{"title":"E1","spaceId":1,"startDate":"2026-10-01","dueDate":"2026-10-31","color":"#6554c0"}`, http.StatusCreated)
	mustStatus(t, r, http.MethodPost, "/epics", `{"title":"E2","spaceId":1}`, http.StatusCreated)
	epics := getJSON[[]domain.Epic](t, r, "/epics")
	want := []domain.Epic{
		{ID: 1, Title: "E1", SpaceID: 1, StartDate: "2026-10-01", DueDate: "2026-10-31", Color: "#6554C0"},
		{ID: 2, Title: "E2", SpaceID: 1},
	}
	if len(epics) != 2 || epicWithoutTimes(epics[0]) != want[0] || epicWithoutTimes(epics[1]) != want[1] {
		t.Fatalf("epics = %+v", epics)
	}

	// タイムラインのドラッグ: 期間だけを変える。色は省略したので今のまま
	mustStatus(t, r, http.MethodPut, "/epics/1", `{"title":"E1","spaceId":1,"startDate":"2026-10-05","dueDate":"2026-11-10"}`, http.StatusOK)
	// 名前だけの変更（ボードからの更新）では日付と色を変えない
	mustStatus(t, r, http.MethodPut, "/epics/1", `{"title":"E1 改","spaceId":1}`, http.StatusOK)
	epics = getJSON[[]domain.Epic](t, r, "/epics")
	if got := epicWithoutTimes(epics[0]); got != (domain.Epic{ID: 1, Title: "E1 改", SpaceID: 1, StartDate: "2026-10-05", DueDate: "2026-11-10", Color: "#6554C0"}) {
		t.Fatalf("epic = %+v", got)
	}

	// 空文字を送ると未設定に戻す
	mustStatus(t, r, http.MethodPut, "/epics/1", `{"title":"E1 改","spaceId":1,"startDate":"","color":""}`, http.StatusOK)
	epics = getJSON[[]domain.Epic](t, r, "/epics")
	if got := epicWithoutTimes(epics[0]); got != (domain.Epic{ID: 1, Title: "E1 改", SpaceID: 1, DueDate: "2026-11-10"}) {
		t.Fatalf("epic = %+v", got)
	}

	// 期限日が開始日より前、形式違いの日付や色は 400 で、保存しない
	mustStatus(t, r, http.MethodPut, "/epics/1", `{"title":"E1 改","spaceId":1,"startDate":"2026-11-20"}`, http.StatusBadRequest)
	mustStatus(t, r, http.MethodPut, "/epics/1", `{"title":"E1 改","spaceId":1,"dueDate":"2026/11/10"}`, http.StatusBadRequest)
	mustStatus(t, r, http.MethodPut, "/epics/1", `{"title":"E1 改","spaceId":1,"color":"purple"}`, http.StatusBadRequest)
	mustStatus(t, r, http.MethodPost, "/epics", `{"title":"E3","spaceId":1,"startDate":"2026-10-10","dueDate":"2026-10-01"}`, http.StatusBadRequest)
	epics = getJSON[[]domain.Epic](t, r, "/epics")
	if len(epics) != 2 || epics[0].StartDate != "" || epics[0].DueDate != "2026-11-10" || epics[0].Color != "" {
		t.Fatalf("epics = %+v", epics)
	}

	// スペースを削除してエピックが未割り当てになっても、日付と色は残る
	mustStatus(t, r, http.MethodPut, "/epics/2", `{"title":"E2","spaceId":1,"startDate":"2026-12-01","dueDate":"2026-12-24","color":"#00875A"}`, http.StatusOK)
	mustStatus(t, r, http.MethodDelete, "/spaces/1", "", http.StatusOK)
	epics = getJSON[[]domain.Epic](t, r, "/epics")
	if got := epicWithoutTimes(epics[1]); got != (domain.Epic{ID: 2, Title: "E2", StartDate: "2026-12-01", DueDate: "2026-12-24", Color: "#00875A"}) {
		t.Fatalf("epic = %+v", got)
	}
}
