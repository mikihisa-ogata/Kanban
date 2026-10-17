package handler

import (
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"testing"

	"todo-api/internal/domain"
	"todo-api/internal/service"
)

func TestSearchJQL(t *testing.T) {
	r, repos := newPeopleRouter(t)
	auth := NewAuthHandler(service.NewAuthService(repos.user, repos.session))
	api := r.Group("/", auth.RequireLogin)
	spaces := NewSpaceHandler(service.NewSpaceService(repos.space, repos.epic, repos.todo, repos.history, repos.status))
	api.POST("/spaces", spaces.CreateSpace)
	api.POST("/epics", NewEpicHandler(service.NewEpicService(repos.epic, repos.todo, repos.space, repos.history, repos.status)).CreateEpic)
	sprints := NewSprintHandler(service.NewSprintService(repos.sprint, repos.todo, repos.space, repos.history))
	api.POST("/spaces/:id/sprints", sprints.CreateSprint)
	api.POST("/sprints/:id/start", sprints.StartSprint)
	api.PUT("/todos/:id/sprint", sprints.SetTodoSprint)
	versions := NewVersionHandler(service.NewVersionService(repos.version, repos.todo, repos.space))
	api.POST("/spaces/:id/versions", versions.CreateVersion)
	api.PUT("/todos/:id/fix-versions", versions.SetTodoVersions)
	api.GET("/issues/jql", NewJQLHandler(service.NewJQLService(repos.todo, repos.space, repos.status, repos.user, repos.epic, repos.sprint, repos.version)).SearchJQL)

	taro, token := signup(t, r, "山田 太郎", "taro@example.com")
	hanako, _ := signup(t, r, "佐藤 花子", "hanako@example.com")
	sendAs(t, r, token, http.MethodPost, "/spaces", `{"Title":"開発","Key":"DEV"}`, http.StatusCreated)
	sendAs(t, r, token, http.MethodPost, "/spaces", `{"Title":"運用","Key":"OPS"}`, http.StatusCreated)
	sendAs(t, r, token, http.MethodPost, "/epics", `{"title":"認証","spaceId":1}`, http.StatusCreated)

	// DEV-1〜DEV-3、OPS-1、TASK-1（スペース未割り当て）。報告者はすべて taro
	bodies := []string{
		fmt.Sprintf(`{"title":"ログイン画面","spaceId":1,"issueType":"Story","priority":"High","assigneeId":%d,"labels":["UI"],"deadline":"2026-11-01","epicId":1}`, taro.ID),
		fmt.Sprintf(`{"title":"100%%_完了の表示","spaceId":1,"issueType":"Bug","priority":"Highest","assigneeId":%d,"description":"進捗率"}`, hanako.ID),
		`{"title":"API 実装","spaceId":1,"priority":"Low","labels":["ui","backend"],"deadline":"2026-10-20"}`,
		fmt.Sprintf(`{"title":"監視","spaceId":2,"issueType":"Bug","assigneeId":%d}`, taro.ID),
		`{"title":"メモ","priority":"Lowest"}`,
	}
	for _, body := range bodies {
		sendAs(t, r, token, http.MethodPost, "/todos", body, http.StatusCreated)
	}
	sendAs(t, r, token, http.MethodPut, "/todos/3", `{"title":"API 実装","spaceId":1,"priority":"Low","status":"InProgress","labels":["ui","backend"],"deadline":"2026-10-20"}`, http.StatusOK)
	// DEV-3 は開始中のスプリント、DEV-1 は未開始のスプリント、DEV-2 はバージョン v1.0
	active := decode[struct{ Sprint domain.Sprint }](t, sendAs(t, r, token, http.MethodPost, "/spaces/1/sprints", `{}`, http.StatusCreated)).Sprint
	future := decode[struct{ Sprint domain.Sprint }](t, sendAs(t, r, token, http.MethodPost, "/spaces/1/sprints", `{"name":"次のスプリント"}`, http.StatusCreated)).Sprint
	sendAs(t, r, token, http.MethodPost, fmt.Sprintf("/sprints/%d/start", active.ID), `{"name":"DEV スプリント 1","startDate":"2026-10-12","endDate":"2026-10-26"}`, http.StatusOK)
	sendAs(t, r, token, http.MethodPut, "/todos/3/sprint", fmt.Sprintf(`{"sprintId":%d}`, active.ID), http.StatusOK)
	sendAs(t, r, token, http.MethodPut, "/todos/1/sprint", fmt.Sprintf(`{"sprintId":%d}`, future.ID), http.StatusOK)
	v1 := decode[struct{ Version domain.Version }](t, sendAs(t, r, token, http.MethodPost, "/spaces/1/versions", `{"name":"v1.0"}`, http.StatusCreated)).Version
	sendAs(t, r, token, http.MethodPut, "/todos/2/fix-versions", fmt.Sprintf(`{"versionIds":[%d]}`, v1.ID), http.StatusOK)

	search := func(jql string) domain.IssueSearchResult {
		t.Helper()
		return decode[domain.IssueSearchResult](t, sendAs(t, r, token, http.MethodGet, "/issues/jql?jql="+url.QueryEscape(jql), "", http.StatusOK))
	}
	keys := func(res domain.IssueSearchResult) []string {
		list := []string{}
		for _, issue := range res.Issues {
			list = append(list, issue.Key)
		}
		return list
	}
	all := []string{"DEV-1", "DEV-2", "DEV-3", "OPS-1", "TASK-1"}
	tests := []struct {
		jql  string
		want []string
	}{
		// 既定は作成日の降順
		{"", []string{"TASK-1", "OPS-1", "DEV-3", "DEV-2", "DEV-1"}},
		{"project = DEV AND status != 進行中 AND assignee = currentUser() ORDER BY priority DESC", []string{"DEV-1"}},
		{"project = DEV ORDER BY priority DESC", []string{"DEV-2", "DEV-1", "DEV-3"}},
		{"project = DEV ORDER BY priority ASC", []string{"DEV-3", "DEV-1", "DEV-2"}},
		{"project IN (OPS, TASK) ORDER BY key", []string{"OPS-1", "TASK-1"}},
		{"project != DEV ORDER BY key", []string{"OPS-1", "TASK-1"}},
		{"type = Bug ORDER BY key", []string{"DEV-2", "OPS-1"}},
		{"status = 進行中", []string{"DEV-3"}},
		{"statusCategory = ToDo ORDER BY key", []string{"DEV-1", "DEV-2", "OPS-1", "TASK-1"}},
		{"priority > Medium ORDER BY key", []string{"DEV-1", "DEV-2"}},
		{"priority <= Low ORDER BY key", []string{"DEV-3", "TASK-1"}},
		{"assignee IS EMPTY ORDER BY key", []string{"DEV-3", "TASK-1"}},
		{"assignee IN (EMPTY, \"佐藤 花子\") ORDER BY key", []string{"DEV-2", "DEV-3", "TASK-1"}},
		// NOT IN は値のない課題に合わない。NOT は値のない課題も含めた補集合
		{"assignee NOT IN (currentUser()) ORDER BY key", []string{"DEV-2"}},
		{"NOT assignee = currentUser() ORDER BY key", []string{"DEV-2", "DEV-3", "TASK-1"}},
		{"reporter = taro@example.com ORDER BY key", all},
		{"labels = ui ORDER BY key", []string{"DEV-1", "DEV-3"}},
		{"labels NOT IN (backend) ORDER BY key", []string{"DEV-1"}},
		{"labels IS EMPTY ORDER BY key", []string{"DEV-2", "OPS-1", "TASK-1"}},
		{"epic = 認証", []string{"DEV-1"}},
		{"epic IS EMPTY ORDER BY key", []string{"DEV-2", "DEV-3", "OPS-1", "TASK-1"}},
		{"sprint IN (openSprints())", []string{"DEV-3"}},
		{"sprint = futureSprints()", []string{"DEV-1"}},
		{"sprint = closedSprints()", []string{}},
		{"NOT sprint IN (closedSprints()) ORDER BY key", all},
		{"sprint IS EMPTY ORDER BY key", []string{"DEV-2", "OPS-1", "TASK-1"}},
		{"fixVersion = v1.0", []string{"DEV-2"}},
		{"fixVersion IS EMPTY ORDER BY key", []string{"DEV-1", "DEV-3", "OPS-1", "TASK-1"}},
		{"due < 2026-10-25", []string{"DEV-3"}},
		{"due >= 2026-10-20 ORDER BY key", []string{"DEV-1", "DEV-3"}},
		{"NOT due = 2026-10-20 ORDER BY key", []string{"DEV-1", "DEV-2", "OPS-1", "TASK-1"}},
		{"due IN (EMPTY, 2026-11-01) ORDER BY key", []string{"DEV-1", "DEV-2", "OPS-1", "TASK-1"}},
		{"created >= -1d ORDER BY key", all},
		{"created < -1d", []string{}},
		{"updated <= now() ORDER BY key", all},
		{"text ~ 進捗", []string{"DEV-2"}},
		{"text ~ ops-1", []string{"OPS-1"}},
		{"text ~ \"%_\"", []string{"DEV-2"}},
		{"(type = Bug OR priority = Lowest) AND NOT project = OPS ORDER BY key", []string{"DEV-2", "TASK-1"}},
		// 期限なしは降順でも末尾。複数の列で並べ替えられる
		{"ORDER BY due DESC", []string{"DEV-1", "DEV-3", "TASK-1", "OPS-1", "DEV-2"}},
		{"project = DEV ORDER BY assignee, key", []string{"DEV-2", "DEV-1", "DEV-3"}},
		{"ORDER BY type, key DESC", []string{"OPS-1", "DEV-2", "DEV-1", "TASK-1", "DEV-3"}},
		// 値は SQL に埋め込まれない
		{"text ~ \"' OR 1=1 --\"", []string{}},
	}
	for _, tt := range tests {
		res := search(tt.jql)
		if got := keys(res); !reflect.DeepEqual(got, tt.want) {
			t.Errorf("%q = %v, want %v", tt.jql, got, tt.want)
		}
		if res.Total != len(tt.want) {
			t.Errorf("%q total = %d, want %d", tt.jql, res.Total, len(tt.want))
		}
	}

	// ページング
	page := decode[domain.IssueSearchResult](t, sendAs(t, r, token, http.MethodGet, "/issues/jql?jql="+url.QueryEscape("ORDER BY key")+"&perPage=2&page=2", "", http.StatusOK))
	if got := keys(page); !reflect.DeepEqual(got, []string{"DEV-3", "OPS-1"}) || page.Total != 5 || page.Page != 2 || page.PerPage != 2 {
		t.Errorf("page 2 = %v, total %d, page %d, perPage %d", got, page.Total, page.Page, page.PerPage)
	}

	// JQL の誤りは 400 で、メッセージと位置を返す
	for _, tt := range []struct {
		jql string
		pos int
		msg string
	}{
		{"status = Done)", 13, "「(」がない「)」"},
		{"foo = 1", 0, "項目「foo」はありません"},
		{"project = NONE", 10, "プロジェクト「NONE」は存在しません"},
		{`project = "x') OR 1=1 -- "`, 10, "存在しません"},
		{"status = 進行中 ORDER BY labels", 22, "並べ替えられません"},
	} {
		res := decode[struct {
			Error    string
			Position int
		}](t, sendAs(t, r, token, http.MethodGet, "/issues/jql?jql="+url.QueryEscape(tt.jql), "", http.StatusBadRequest))
		if res.Position != tt.pos || !strings.Contains(res.Error, tt.msg) {
			t.Errorf("%q error = %d: %s, want %d: …%s…", tt.jql, res.Position, res.Error, tt.pos, tt.msg)
		}
	}
	for _, query := range []string{"page=0", "perPage=101", "page=x"} {
		sendAs(t, r, token, http.MethodGet, "/issues/jql?jql=&"+query, "", http.StatusBadRequest)
	}
	// ログインが必要
	sendAs(t, r, "", http.MethodGet, "/issues/jql?jql=", "", http.StatusUnauthorized)
}
