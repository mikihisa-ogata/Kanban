package handler

import (
	"fmt"
	"net/http"
	"reflect"
	"testing"

	"github.com/gin-gonic/gin"

	"todo-api/internal/domain"
	"todo-api/internal/service"
)

// newComponentRouter はログインが必要なスペース・課題・コンポーネントの API を持つルーターと、リポジトリを返す
func newComponentRouter(t *testing.T) (*gin.Engine, testRepos) {
	t.Helper()
	r, repos := newPeopleRouter(t)
	auth := NewAuthHandler(service.NewAuthService(repos.user, repos.session))
	spaces := NewSpaceHandler(service.NewSpaceService(repos.space, repos.epic, repos.todo, repos.history, repos.status))
	h := NewComponentHandler(service.NewComponentService(repos.component, repos.todo, repos.epic, repos.space, repos.user, repos.watcher, repos.history, repos.status))
	api := r.Group("/", auth.RequireLogin)
	api.POST("/spaces", spaces.CreateSpace)
	api.DELETE("/spaces/:id", spaces.DeleteSpace)
	api.DELETE("/todos/:id", NewTodoHandler(service.NewTodoService(repos.todo, repos.epic, repos.space, repos.link, repos.history, repos.attachment, repos.status, repos.user, repos.watcher, repos.notification)).DeleteTodo)
	api.GET("/components", h.GetComponents)
	api.POST("/spaces/:id/components", h.CreateComponent)
	api.PUT("/components/:id", h.UpdateComponent)
	api.DELETE("/components/:id", h.DeleteComponent)
	api.PUT("/todos/:id/components", h.SetTodoComponents)
	return r, repos
}

// todoOf は課題（ID）を返す
func todoOf(t *testing.T, r *gin.Engine, token string, id int) domain.Todo {
	t.Helper()
	for _, todo := range decode[[]domain.Todo](t, sendAs(t, r, token, http.MethodGet, "/todos", "", http.StatusOK)) {
		if todo.ID == id {
			return todo
		}
	}
	t.Fatalf("todo %d not found", id)
	return domain.Todo{}
}

func TestComponent_CRUDAndDefaultAssignee(t *testing.T) {
	r, repos := newComponentRouter(t)
	taro, token := signup(t, r, "山田 太郎", "taro@example.com")
	hanako, _ := signup(t, r, "佐藤 花子", "hanako@example.com")
	jiro, _ := signup(t, r, "鈴木 次郎", "jiro@example.com")
	sendAs(t, r, token, http.MethodPost, "/spaces", `{"Title":"開発","Key":"DEV"}`, http.StatusCreated)
	sendAs(t, r, token, http.MethodPost, "/spaces", `{"Title":"運用","Key":"OPS"}`, http.StatusCreated)

	create := func(spaceID int, body string) domain.Component {
		t.Helper()
		res := decode[struct{ Component domain.Component }](t,
			sendAs(t, r, token, http.MethodPost, fmt.Sprintf("/spaces/%d/components", spaceID), body, http.StatusCreated))
		return res.Component
	}
	front := create(1, fmt.Sprintf(`{"name":" フロント ","description":"画面","defaultAssigneeId":%d}`, hanako.ID))
	if front.Name != "フロント" || front.Description != "画面" || front.DefaultAssignee == nil || front.DefaultAssignee.DisplayName != "佐藤 花子" {
		t.Fatalf("created component = %+v", front)
	}
	api := create(1, fmt.Sprintf(`{"name":"API","defaultAssigneeId":%d}`, jiro.ID))
	docs := create(1, `{"name":"ドキュメント"}`)
	ops := create(2, `{"name":"フロント"}`) // 別のスペースなら同じ名前でよい
	if docs.DefaultAssignee != nil || docs.DefaultAssigneeID != 0 {
		t.Errorf("component without default assignee = %+v", docs)
	}
	// 重複名は 409。空の名前・存在しないユーザー・未割り当て・存在しないスペース
	sendAs(t, r, token, http.MethodPost, "/spaces/1/components", `{"name":"api"}`, http.StatusConflict)
	sendAs(t, r, token, http.MethodPost, "/spaces/1/components", `{"name":" "}`, http.StatusBadRequest)
	sendAs(t, r, token, http.MethodPost, "/spaces/1/components", `{"name":"x","defaultAssigneeId":99}`, http.StatusBadRequest)
	sendAs(t, r, token, http.MethodPost, "/spaces/0/components", `{"name":"x"}`, http.StatusBadRequest)
	sendAs(t, r, token, http.MethodPost, "/spaces/99/components", `{"name":"x"}`, http.StatusNotFound)

	// 一覧はスペース・名前順
	var names []string
	for _, c := range decode[[]domain.Component](t, sendAs(t, r, token, http.MethodGet, "/components", "", http.StatusOK)) {
		names = append(names, fmt.Sprintf("%d:%s", c.SpaceID, c.Name))
	}
	if want := []string{"1:API", "1:ドキュメント", "1:フロント", "2:フロント"}; !reflect.DeepEqual(names, want) {
		t.Errorf("components = %v, want %v", names, want)
	}

	// 編集で既定の担当者を外せる。存在しないコンポーネントは 404
	sendAs(t, r, token, http.MethodPut, fmt.Sprintf("/components/%d", docs.ID), `{"name":"Docs","description":"手順書"}`, http.StatusOK)
	sendAs(t, r, token, http.MethodPut, fmt.Sprintf("/components/%d", docs.ID), `{"name":"API"}`, http.StatusConflict)
	sendAs(t, r, token, http.MethodPut, "/components/99", `{"name":"x"}`, http.StatusNotFound)

	// 課題: A・B（DEV、担当者なし）、C（DEV、担当者あり）、X（OPS）
	for _, body := range []string{`{"title":"A","spaceId":1}`, `{"title":"B","spaceId":1}`, fmt.Sprintf(`{"title":"C","spaceId":1,"assigneeId":%d}`, taro.ID), `{"title":"X","spaceId":2}`} {
		sendAs(t, r, token, http.MethodPost, "/todos", body, http.StatusCreated)
	}
	set := func(todoID int, ids string, want int) {
		t.Helper()
		sendAs(t, r, token, http.MethodPut, fmt.Sprintf("/todos/%d/components", todoID), fmt.Sprintf(`{"componentIds":%s}`, ids), want)
	}

	// 既定の担当者のないコンポーネントだけなら担当者は空のまま。複数付けられ、重複は 1 つにまとめて ID 順
	set(1, fmt.Sprintf("[%d,%d]", docs.ID, docs.ID), http.StatusOK)
	if a := todoOf(t, r, token, 1); a.AssigneeID != 0 || !reflect.DeepEqual(a.ComponentIDs, []int{docs.ID}) {
		t.Fatalf("A = assignee %d, components %v", a.AssigneeID, a.ComponentIDs)
	}
	// 担当者が空の課題に既定の担当者を持つコンポーネントを付けると、ID 順で最初のコンポーネントの既定の担当者が入る
	set(1, fmt.Sprintf("[%d,%d,%d]", api.ID, front.ID, docs.ID), http.StatusOK)
	a := todoOf(t, r, token, 1)
	if a.AssigneeID != hanako.ID || !reflect.DeepEqual(a.ComponentIDs, []int{front.ID, api.ID, docs.ID}) {
		t.Fatalf("A = assignee %d, components %v", a.AssigneeID, a.ComponentIDs)
	}
	// 担当者の変化は履歴に残り、新しい担当者はウォッチャーになる
	history := decode[[]domain.HistoryEntry](t, sendAs(t, r, token, http.MethodGet, "/todos/1/history", "", http.StatusOK))
	found := false
	for _, h := range history {
		if h.Field == domain.HistoryFieldAssignee && h.NewValue != nil && *h.NewValue == "佐藤 花子" && h.UserID == taro.ID {
			found = true
		}
	}
	if !found {
		t.Errorf("assignee history not recorded: %+v", history)
	}
	watchers, err := repos.watcher.FindByTodo(1)
	if err != nil {
		t.Fatal(err)
	}
	if !containsWatcher(watchers, hanako.ID) {
		t.Errorf("default assignee is not a watcher: %+v", watchers)
	}

	// 担当者がいる課題では変わらない
	set(3, fmt.Sprintf("[%d]", front.ID), http.StatusOK)
	if c := todoOf(t, r, token, 3); c.AssigneeID != taro.ID {
		t.Errorf("C assignee = %d, want %d", c.AssigneeID, taro.ID)
	}
	// 既に付いていたコンポーネントでは入らない（担当者を外したあとに付け直さない限り）
	sendAs(t, r, token, http.MethodPut, "/todos/1", `{"title":"A","spaceId":1,"assigneeId":null}`, http.StatusOK)
	set(1, fmt.Sprintf("[%d]", front.ID), http.StatusOK)
	if a := todoOf(t, r, token, 1); a.AssigneeID != 0 {
		t.Errorf("A assignee = %d, want 0 (component was already set)", a.AssigneeID)
	}
	set(1, fmt.Sprintf("[%d,%d]", front.ID, api.ID), http.StatusOK)
	if a := todoOf(t, r, token, 1); a.AssigneeID != jiro.ID {
		t.Errorf("A assignee = %d, want %d", a.AssigneeID, jiro.ID)
	}

	// 別スペースのコンポーネントは 400、存在しないコンポーネント・課題は 404。空の配列で外せる
	set(4, fmt.Sprintf("[%d]", front.ID), http.StatusBadRequest)
	set(4, "[99]", http.StatusNotFound)
	sendAs(t, r, token, http.MethodPut, "/todos/99/components", `{"componentIds":[]}`, http.StatusNotFound)
	set(4, fmt.Sprintf("[%d]", ops.ID), http.StatusOK)
	set(4, "[]", http.StatusOK)
	if x := todoOf(t, r, token, 4); len(x.ComponentIDs) != 0 {
		t.Errorf("X components = %v", x.ComponentIDs)
	}

	// 削除すると課題からも外れる
	sendAs(t, r, token, http.MethodDelete, fmt.Sprintf("/components/%d", api.ID), "", http.StatusOK)
	if a := todoOf(t, r, token, 1); !reflect.DeepEqual(a.ComponentIDs, []int{front.ID}) {
		t.Errorf("A components after delete = %v", a.ComponentIDs)
	}
	sendAs(t, r, token, http.MethodDelete, fmt.Sprintf("/components/%d", api.ID), "", http.StatusNotFound)

	// スペースを移すと外れる
	sendAs(t, r, token, http.MethodPut, "/todos/1", `{"title":"A","spaceId":2}`, http.StatusOK)
	if a := todoOf(t, r, token, 1); len(a.ComponentIDs) != 0 {
		t.Errorf("A components after moving space = %v", a.ComponentIDs)
	}

	// 課題やスペースを削除しても付与が残らない
	set(3, fmt.Sprintf("[%d]", front.ID), http.StatusOK)
	set(2, fmt.Sprintf("[%d]", front.ID), http.StatusOK)
	sendAs(t, r, token, http.MethodDelete, "/todos/2", "", http.StatusOK)
	sendAs(t, r, token, http.MethodDelete, "/spaces/1", "", http.StatusOK)
	for _, c := range decode[[]domain.Component](t, sendAs(t, r, token, http.MethodGet, "/components", "", http.StatusOK)) {
		if c.SpaceID == 1 {
			t.Errorf("component of deleted space remains: %+v", c)
		}
	}
	// スペースの削除で未割り当てに移った課題 C からも外れる
	if c := todoOf(t, r, token, 3); c.SpaceID != 0 || len(c.ComponentIDs) != 0 {
		t.Errorf("C after deleting space = space %d, components %v", c.SpaceID, c.ComponentIDs)
	}
}

func containsWatcher(watchers []domain.UserSummary, userID int) bool {
	for _, w := range watchers {
		if w.ID == userID {
			return true
		}
	}
	return false
}
