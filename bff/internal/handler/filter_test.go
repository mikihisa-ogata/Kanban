package handler

import (
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"todo-api/internal/domain"
	"todo-api/internal/service"
)

// newFilterRouter はログインが必要なスペース・課題・フィルター・JQL 検索の API を持つルーターを返す
func newFilterRouter(t *testing.T) *gin.Engine {
	t.Helper()
	r, repos := newPeopleRouter(t)
	auth := NewAuthHandler(service.NewAuthService(repos.user, repos.session))
	api := r.Group("/", auth.RequireLogin)
	api.POST("/spaces", NewSpaceHandler(service.NewSpaceService(repos.space, repos.epic, repos.todo, repos.history, repos.status)).CreateSpace)
	api.DELETE("/todos/:id", NewTodoHandler(service.NewTodoService(repos.todo, repos.epic, repos.space, repos.link, repos.history, repos.attachment, repos.status, repos.user, repos.watcher, repos.notification)).DeleteTodo)
	api.GET("/issues/jql", NewJQLHandler(service.NewJQLService(repos.todo, repos.space, repos.status, repos.user, repos.epic, repos.sprint, repos.version)).SearchJQL)
	h := NewFilterHandler(service.NewFilterService(repos.filter), service.NewIssueViewService(repos.issueView, repos.todo))
	api.GET("/filters", h.GetFilters)
	api.POST("/filters", h.CreateFilter)
	api.PUT("/filters/:id", h.UpdateFilter)
	api.PUT("/filters/:id/favourite", h.SetFavourite)
	api.DELETE("/filters/:id", h.DeleteFilter)
	api.POST("/todos/:id/view", h.RecordView)
	return r
}

func TestFilter_SaveUpdateFavouriteDelete(t *testing.T) {
	r := newFilterRouter(t)
	_, token := signup(t, r, "山田 太郎", "taro@example.com")
	_, other := signup(t, r, "佐藤 花子", "hanako@example.com")
	sendAs(t, r, token, http.MethodPost, "/spaces", `{"Title":"開発","Key":"DEV"}`, http.StatusCreated)
	sendAs(t, r, token, http.MethodPost, "/todos", `{"title":"ログイン画面","spaceId":1,"issueType":"Bug"}`, http.StatusCreated)
	sendAs(t, r, token, http.MethodPost, "/todos", `{"title":"API 実装","spaceId":1}`, http.StatusCreated)

	create := func(body string) domain.SavedFilter {
		t.Helper()
		return decode[struct{ Filter domain.SavedFilter }](t, sendAs(t, r, token, http.MethodPost, "/filters", body, http.StatusCreated)).Filter
	}
	bugs := create(`{"name":" DEV のバグ ","jql":" project = DEV AND type = Bug ORDER BY key ","favourite":true}`)
	if bugs.Name != "DEV のバグ" || bugs.JQL != "project = DEV AND type = Bug ORDER BY key" || !bugs.Favourite || bugs.ID == 0 {
		t.Fatalf("created filter = %+v", bugs)
	}
	all := create(`{"name":"すべて","jql":""}`)
	if all.Favourite || all.JQL != "" {
		t.Fatalf("created filter = %+v", all)
	}

	// 保存した JQL で検索すると、同じ結果になる
	keysOf := func(jql string) []string {
		t.Helper()
		res := decode[domain.IssueSearchResult](t, sendAs(t, r, token, http.MethodGet, "/issues/jql?jql="+url.QueryEscape(jql), "", http.StatusOK))
		keys := []string{}
		for _, issue := range res.Issues {
			keys = append(keys, issue.Key)
		}
		return keys
	}
	if got := keysOf(bugs.JQL); !reflect.DeepEqual(got, []string{"DEV-1"}) {
		t.Errorf("search by saved JQL = %v", got)
	}

	// 一覧は名前順。ほかのユーザーのフィルターは見えない
	names := func(token string) []string {
		t.Helper()
		list := []string{}
		for _, f := range decode[[]domain.SavedFilter](t, sendAs(t, r, token, http.MethodGet, "/filters", "", http.StatusOK)) {
			list = append(list, f.Name)
		}
		return list
	}
	if got := names(token); !reflect.DeepEqual(got, []string{"DEV のバグ", "すべて"}) {
		t.Errorf("filters = %v", got)
	}
	if got := names(other); len(got) != 0 {
		t.Errorf("other user's filters = %v", got)
	}

	// 上書き保存
	updated := decode[struct{ Filter domain.SavedFilter }](t, sendAs(t, r, token, http.MethodPut, fmt.Sprintf("/filters/%d", bugs.ID),
		`{"name":"DEV のタスク","jql":"project = DEV AND type = Task"}`, http.StatusOK)).Filter
	if updated.Name != "DEV のタスク" || updated.JQL != "project = DEV AND type = Task" || !updated.Favourite {
		t.Errorf("updated filter = %+v", updated)
	}
	if got := keysOf(updated.JQL); !reflect.DeepEqual(got, []string{"DEV-2"}) {
		t.Errorf("search by updated JQL = %v", got)
	}
	// 名前を変えずに上書きできる（自分自身とは重複しない）
	sendAs(t, r, token, http.MethodPut, fmt.Sprintf("/filters/%d", bugs.ID), `{"name":"dev のタスク","jql":"project = DEV"}`, http.StatusOK)

	// お気に入り
	unstarred := decode[struct{ Filter domain.SavedFilter }](t, sendAs(t, r, token, http.MethodPut, fmt.Sprintf("/filters/%d/favourite", bugs.ID), `{"favourite":false}`, http.StatusOK)).Filter
	if unstarred.Favourite || unstarred.JQL != "project = DEV" {
		t.Errorf("unstarred filter = %+v", unstarred)
	}
	starred := decode[struct{ Filter domain.SavedFilter }](t, sendAs(t, r, token, http.MethodPut, fmt.Sprintf("/filters/%d/favourite", all.ID), `{"favourite":true}`, http.StatusOK)).Filter
	if !starred.Favourite || starred.Name != "すべて" {
		t.Errorf("starred filter = %+v", starred)
	}

	// 誤り
	for _, tt := range []struct {
		method, path, body string
		status             int
		msg                string
	}{
		{http.MethodPost, "/filters", `{"name":"  ","jql":""}`, http.StatusBadRequest, "名前"},
		{http.MethodPost, "/filters", `{"name":"` + strings.Repeat("あ", 101) + `","jql":""}`, http.StatusBadRequest, "名前"},
		{http.MethodPost, "/filters", `{"name":"すべて","jql":"project = DEV"}`, http.StatusConflict, "同じ名前"},
		{http.MethodPost, "/filters", `{"name":"大文字","jql":""}`, http.StatusCreated, ""},
		{http.MethodPost, "/filters", `{"name":"誤り","jql":"status = Done)"}`, http.StatusBadRequest, "「(」がない「)」"},
		{http.MethodPut, fmt.Sprintf("/filters/%d", bugs.ID), `{"name":"すべて","jql":""}`, http.StatusConflict, "同じ名前"},
		{http.MethodPut, "/filters/999", `{"name":"x","jql":""}`, http.StatusNotFound, "存在しません"},
		{http.MethodPut, "/filters/x", `{"name":"x","jql":""}`, http.StatusBadRequest, ""},
	} {
		w := sendAs(t, r, token, tt.method, tt.path, tt.body, tt.status)
		if !strings.Contains(string(w), tt.msg) {
			t.Errorf("%s %s body = %s, want …%s…", tt.method, tt.path, w, tt.msg)
		}
	}
	// JQL の構文の誤りは位置も返す
	res := decode[struct{ Position int }](t, sendAs(t, r, token, http.MethodPost, "/filters", `{"name":"誤り","jql":"status = Done)"}`, http.StatusBadRequest))
	if res.Position != 13 {
		t.Errorf("position = %d, want 13", res.Position)
	}
	// 構文が正しければ、存在しない名前があっても保存できる（検索するときに誤りになる）
	sendAs(t, r, token, http.MethodPost, "/filters", `{"name":"なし","jql":"project = NONE"}`, http.StatusCreated)

	// ほかのユーザーのフィルターは変えられない・消せない
	sendAs(t, r, other, http.MethodPut, fmt.Sprintf("/filters/%d", bugs.ID), `{"name":"x","jql":""}`, http.StatusNotFound)
	sendAs(t, r, other, http.MethodPut, fmt.Sprintf("/filters/%d/favourite", bugs.ID), `{"favourite":true}`, http.StatusNotFound)
	sendAs(t, r, other, http.MethodDelete, fmt.Sprintf("/filters/%d", bugs.ID), "", http.StatusNotFound)

	// 削除
	sendAs(t, r, token, http.MethodDelete, fmt.Sprintf("/filters/%d", bugs.ID), "", http.StatusOK)
	sendAs(t, r, token, http.MethodDelete, fmt.Sprintf("/filters/%d", bugs.ID), "", http.StatusNotFound)
	if got := names(token); !reflect.DeepEqual(got, []string{"すべて", "なし", "大文字"}) {
		t.Errorf("filters after delete = %v", got)
	}

	// ログインが必要
	sendAs(t, r, "", http.MethodGet, "/filters", "", http.StatusUnauthorized)
}

func TestFilter_IssueHistoryAndDefaultFilters(t *testing.T) {
	r := newFilterRouter(t)
	taro, token := signup(t, r, "山田 太郎", "taro@example.com")
	hanako, other := signup(t, r, "佐藤 花子", "hanako@example.com")
	sendAs(t, r, token, http.MethodPost, "/spaces", `{"Title":"開発","Key":"DEV"}`, http.StatusCreated)
	// DEV-1・DEV-2 は taro が報告、DEV-3 は hanako が報告。DEV-1・DEV-3 の担当は taro、DEV-2 は完了
	sendAs(t, r, token, http.MethodPost, "/todos", fmt.Sprintf(`{"title":"ログイン画面","spaceId":1,"assigneeId":%d}`, taro.ID), http.StatusCreated)
	sendAs(t, r, token, http.MethodPost, "/todos", fmt.Sprintf(`{"title":"API 実装","spaceId":1,"assigneeId":%d}`, taro.ID), http.StatusCreated)
	sendAs(t, r, token, http.MethodPut, "/todos/2", fmt.Sprintf(`{"title":"API 実装","spaceId":1,"status":"Closed","assigneeId":%d}`, taro.ID), http.StatusOK)
	sendAs(t, r, other, http.MethodPost, "/todos", fmt.Sprintf(`{"title":"監視","spaceId":1,"assigneeId":%d}`, taro.ID), http.StatusCreated)
	sendAs(t, r, other, http.MethodPost, "/todos", fmt.Sprintf(`{"title":"メモ","spaceId":1,"assigneeId":%d}`, hanako.ID), http.StatusCreated)

	keysOf := func(token, jql string) []string {
		t.Helper()
		res := decode[domain.IssueSearchResult](t, sendAs(t, r, token, http.MethodGet, "/issues/jql?jql="+url.QueryEscape(jql), "", http.StatusOK))
		keys := []string{}
		for _, issue := range res.Issues {
			keys = append(keys, issue.Key)
		}
		return keys
	}

	// 表示を記録すると issueHistory() に出る。最後に表示した順に並べられ、もう一度表示すると先頭になる
	if got := keysOf(token, "issuekey IN issueHistory()"); len(got) != 0 {
		t.Errorf("history before viewing = %v", got)
	}
	for _, id := range []int{3, 1, 2, 3} {
		sendAs(t, r, token, http.MethodPost, fmt.Sprintf("/todos/%d/view", id), "", http.StatusOK)
	}
	sendAs(t, r, other, http.MethodPost, "/todos/4/view", "", http.StatusOK)
	sendAs(t, r, token, http.MethodPost, "/todos/999/view", "", http.StatusNotFound)

	tests := []struct {
		token, jql string
		want       []string
	}{
		// 既定のフィルター（フロントの DEFAULT_FILTERS と同じ JQL）
		{token, "assignee = currentUser() AND statusCategory != Done ORDER BY updated DESC", []string{"DEV-3", "DEV-1"}},
		{token, "reporter = currentUser() ORDER BY created DESC", []string{"DEV-2", "DEV-1"}},
		{token, "issuekey IN issueHistory() ORDER BY lastViewed DESC", []string{"DEV-3", "DEV-2", "DEV-1"}},
		{token, "ORDER BY key", []string{"DEV-1", "DEV-2", "DEV-3", "DEV-4"}},
		// 履歴はユーザーごと
		{other, "issuekey IN issueHistory() ORDER BY lastViewed DESC", []string{"DEV-4"}},
		// 表示していない課題は lastViewed の並べ替えで末尾
		{token, "ORDER BY lastViewed ASC", []string{"DEV-1", "DEV-2", "DEV-3", "DEV-4"}},
		// 課題キーで選べる。issueHistory() と組み合わせられる
		{token, "issuekey = dev-2", []string{"DEV-2"}},
		{token, "key IN (DEV-1, DEV-4) ORDER BY key", []string{"DEV-1", "DEV-4"}},
		{token, "issue NOT IN (DEV-1, issueHistory()) ORDER BY key", []string{"DEV-4"}},
		{token, "issuekey != DEV-1 AND issuekey IN issueHistory() ORDER BY key", []string{"DEV-2", "DEV-3"}},
	}
	for _, tt := range tests {
		if got := keysOf(tt.token, tt.jql); !reflect.DeepEqual(got, tt.want) {
			t.Errorf("%q = %v, want %v", tt.jql, got, tt.want)
		}
	}

	for _, tt := range []struct {
		jql string
		msg string
	}{
		{"issuekey IS EMPTY", "EMPTY を使えません"},
		{"issuekey IN openSprints()", "使えません"},
		{"issuekey > DEV-1", "使えません"},
		{"ORDER BY viewed", "並べ替えられません"},
	} {
		w := sendAs(t, r, token, http.MethodGet, "/issues/jql?jql="+url.QueryEscape(tt.jql), "", http.StatusBadRequest)
		if !strings.Contains(string(w), tt.msg) {
			t.Errorf("%q body = %s, want …%s…", tt.jql, w, tt.msg)
		}
	}

	// 課題を消すと、履歴からも消える
	sendAs(t, r, token, http.MethodDelete, "/todos/3", "", http.StatusOK)
	if got := keysOf(token, "issuekey IN issueHistory() ORDER BY lastViewed DESC"); !reflect.DeepEqual(got, []string{"DEV-2", "DEV-1"}) {
		t.Errorf("history after delete = %v", got)
	}
}
