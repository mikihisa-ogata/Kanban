package handler

import (
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"testing"

	"github.com/gin-gonic/gin"

	"todo-api/internal/domain"
	"todo-api/internal/service"
)

// newSearchRouter はログインが必要なスペース・課題・課題一覧の API を持つルーターを返す
func newSearchRouter(t *testing.T) *gin.Engine {
	t.Helper()
	r, repos := newPeopleRouter(t)
	auth := NewAuthHandler(service.NewAuthService(repos.user, repos.session))
	spaces := NewSpaceHandler(service.NewSpaceService(repos.space, repos.epic, repos.todo, repos.history, repos.status))
	todos := NewTodoHandler(service.NewTodoService(repos.todo, repos.epic, repos.space, repos.link, repos.history, repos.attachment, repos.status, repos.user, repos.watcher, repos.notification))
	api := r.Group("/", auth.RequireLogin)
	api.POST("/spaces", spaces.CreateSpace)
	api.GET("/issues/search", todos.SearchIssues)
	api.GET("/issues/:key", todos.GetIssue)
	return r
}

func TestSearchIssues(t *testing.T) {
	r := newSearchRouter(t)
	taro, token := signup(t, r, "山田 太郎", "taro@example.com")
	hanako, _ := signup(t, r, "佐藤 花子", "hanako@example.com")
	sendAs(t, r, token, http.MethodPost, "/spaces", `{"Title":"開発","Key":"DEV"}`, http.StatusCreated)
	sendAs(t, r, token, http.MethodPost, "/spaces", `{"Title":"運用","Key":"OPS"}`, http.StatusCreated)

	// DEV-1〜DEV-3、OPS-1、TASK-1（スペース未割り当て）
	bodies := []string{
		fmt.Sprintf(`{"title":"ログイン画面","spaceId":1,"issueType":"Story","priority":"High","assigneeId":%d,"labels":["UI"],"deadline":"2026-11-01"}`, taro.ID),
		fmt.Sprintf(`{"title":"100%%_完了の表示","spaceId":1,"issueType":"Bug","priority":"Highest","assigneeId":%d,"description":"進捗率"}`, hanako.ID),
		`{"title":"API 実装","spaceId":1,"priority":"Low","labels":["ui","backend"],"deadline":"2026-10-20"}`,
		fmt.Sprintf(`{"title":"監視","spaceId":2,"issueType":"Bug","assigneeId":%d}`, taro.ID),
		`{"title":"メモ","priority":"Lowest"}`,
	}
	for _, body := range bodies {
		sendAs(t, r, token, http.MethodPost, "/todos", body, http.StatusCreated)
	}
	sendAs(t, r, token, http.MethodPut, "/todos/3", `{"title":"API 実装","spaceId":1,"status":"InProgress","labels":["ui","backend"],"deadline":"2026-10-20"}`, http.StatusOK)

	search := func(query string) domain.IssueSearchResult {
		t.Helper()
		return decode[domain.IssueSearchResult](t, sendAs(t, r, token, http.MethodGet, "/issues/search?"+query, "", http.StatusOK))
	}
	keys := func(res domain.IssueSearchResult) []string {
		list := []string{}
		for _, issue := range res.Issues {
			list = append(list, issue.Key)
		}
		return list
	}
	tests := []struct {
		name  string
		query string
		want  []string
	}{
		{"既定は作成日の降順", "", []string{"TASK-1", "OPS-1", "DEV-3", "DEV-2", "DEV-1"}},
		{"スペース（キー、複数は OR）", "space=dev&space=TASK&sort=key", []string{"DEV-1", "DEV-2", "DEV-3", "TASK-1"}},
		{"存在しないスペースは合わない", "space=NONE", []string{}},
		{"タイプ", "type=Bug&sort=key", []string{"DEV-2", "OPS-1"}},
		{"ステータス名", "status=" + url.QueryEscape("進行中"), []string{"DEV-3"}},
		{"担当者と未割り当て", fmt.Sprintf("assignee=%d&assignee=unassigned&sort=key", taro.ID), []string{"DEV-1", "DEV-3", "OPS-1", "TASK-1"}},
		{"ラベルは大文字・小文字を区別しない", "label=ui&sort=key", []string{"DEV-1", "DEV-3"}},
		{"条件どうしは AND", "label=ui&status=" + url.QueryEscape("オープン"), []string{"DEV-1"}},
		{"テキストは要約・説明", "text=" + url.QueryEscape("進捗"), []string{"DEV-2"}},
		{"テキストの % と _ はそのままの文字", "text=" + url.QueryEscape("%_"), []string{"DEV-2"}},
		{"テキストは課題キーにも合う", "text=ops-1", []string{"OPS-1"}},
		{"優先度は高い順", "sort=priority", []string{"DEV-2", "DEV-1", "OPS-1", "DEV-3", "TASK-1"}},
		{"優先度の降順", "sort=priority&order=desc", []string{"TASK-1", "DEV-3", "OPS-1", "DEV-1", "DEV-2"}},
		{"期限なしは末尾", "sort=due", []string{"DEV-3", "DEV-1", "DEV-2", "OPS-1", "TASK-1"}},
		{"期限なしは降順でも末尾", "sort=due&order=desc", []string{"DEV-1", "DEV-3", "TASK-1", "OPS-1", "DEV-2"}},
		{"担当者の表示名順、未割り当ては末尾", "sort=assignee&space=DEV", []string{"DEV-2", "DEV-1", "DEV-3"}},
		{"要約順", "sort=summary&space=DEV", []string{"DEV-2", "DEV-3", "DEV-1"}},
	}
	for _, tt := range tests {
		res := search(tt.query)
		if got := keys(res); !reflect.DeepEqual(got, tt.want) {
			t.Errorf("%s: %s = %v, want %v", tt.name, tt.query, got, tt.want)
		}
		if res.Total != len(tt.want) {
			t.Errorf("%s: total = %d, want %d", tt.name, res.Total, len(tt.want))
		}
	}

	// ページング: 総数は全件、課題は指定したページの分だけ
	page := search("sort=key&perPage=2&page=2")
	if got := keys(page); !reflect.DeepEqual(got, []string{"DEV-3", "OPS-1"}) || page.Total != 5 || page.Page != 2 || page.PerPage != 2 {
		t.Errorf("page 2 = %v, total %d, page %d, perPage %d", got, page.Total, page.Page, page.PerPage)
	}
	if got := keys(search("sort=key&perPage=2&page=4")); len(got) != 0 {
		t.Errorf("page beyond the end = %v", got)
	}
	// ラベル・担当者などは課題と同じ形で返る
	first := search("sort=key&perPage=1").Issues[0]
	if first.Key != "DEV-1" || !reflect.DeepEqual(first.Labels, []string{"UI"}) || first.Assignee == nil || first.Assignee.ID != taro.ID || first.StatusName != "オープン" {
		t.Errorf("first issue = %+v", first)
	}

	// 誤った指定は 400
	for _, query := range []string{"sort=title", "order=up", "page=0", "perPage=101", "page=x", "assignee=me", "type=Epic"} {
		sendAs(t, r, token, http.MethodGet, "/issues/search?"+query, "", http.StatusBadRequest)
	}
	// /issues/:key はそのまま使える
	sendAs(t, r, token, http.MethodGet, "/issues/DEV-1", "", http.StatusOK)
}
