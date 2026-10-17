package handler

import (
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"todo-api/internal/domain"
	"todo-api/internal/service"
)

// newDashboardRouter はダッシュボードの API と、課題・フィルターの操作の API を持つルーターを返す
func newDashboardRouter(t *testing.T) *gin.Engine {
	t.Helper()
	r, repos := newPeopleRouter(t)
	auth := NewAuthHandler(service.NewAuthService(repos.user, repos.session))
	api := r.Group("/", auth.RequireLogin)
	api.POST("/spaces", NewSpaceHandler(service.NewSpaceService(repos.space, repos.epic, repos.todo, repos.history, repos.status)).CreateSpace)
	api.DELETE("/todos/:id", NewTodoHandler(service.NewTodoService(repos.todo, repos.epic, repos.space, repos.link, repos.history, repos.attachment, repos.status, repos.user, repos.watcher, repos.notification)).DeleteTodo)
	filters := NewFilterHandler(service.NewFilterService(repos.filter), service.NewIssueViewService(repos.issueView, repos.todo))
	api.POST("/filters", filters.CreateFilter)
	api.DELETE("/filters/:id", filters.DeleteFilter)
	jql := service.NewJQLService(repos.todo, repos.space, repos.status, repos.user, repos.epic, repos.sprint, repos.version)
	h := NewDashboardHandler(service.NewDashboardService(repos.dashboard, repos.filter, repos.history, repos.todo, jql))
	api.GET("/dashboard", h.GetDashboard)
	api.PUT("/dashboard", h.SaveDashboard)
	api.GET("/dashboard/activity", h.GetActivity)
	api.GET("/dashboard/status-counts", h.GetStatusCounts)
	api.GET("/dashboard/created-vs-resolved", h.GetCreatedVsResolved)
	return r
}

func TestDashboard_SaveAndReload(t *testing.T) {
	r := newDashboardRouter(t)
	_, token := signup(t, r, "山田 太郎", "taro@example.com")
	_, other := signup(t, r, "佐藤 花子", "hanako@example.com")
	gadgetsOf := func(body []byte) []domain.DashboardGadget {
		t.Helper()
		return decode[struct{ Gadgets []domain.DashboardGadget }](t, body).Gadgets
	}

	// 保存していなければ既定の並び
	if got := gadgetsOf(sendAs(t, r, token, http.MethodGet, "/dashboard", "", http.StatusOK)); !reflect.DeepEqual(got, domain.DefaultDashboard()) {
		t.Errorf("default dashboard = %+v", got)
	}

	mine := decode[struct{ Filter domain.SavedFilter }](t, sendAs(t, r, token, http.MethodPost, "/filters", `{"name":"バグ","jql":"type = Bug"}`, http.StatusCreated)).Filter
	theirs := decode[struct{ Filter domain.SavedFilter }](t, sendAs(t, r, other, http.MethodPost, "/filters", `{"name":"花子の","jql":""}`, http.StatusCreated)).Filter

	// 追加・削除・並べ替えた並びを保存すると、読み直しても同じ（リロード後も残る）。
	// ほかのユーザーのフィルターは使えず、未選択になる
	body := fmt.Sprintf(`{"gadgets":[{"id":"f1","type":"filterResults","filterId":%d},{"id":"s1","type":"statusPie","filterId":%d},{"id":"assigned","type":"assignedToMe","filterId":%d}]}`,
		mine.ID, theirs.ID, mine.ID)
	want := []domain.DashboardGadget{
		{ID: "f1", Type: domain.GadgetFilterResults, FilterID: mine.ID},
		{ID: "s1", Type: domain.GadgetStatusPie},
		{ID: "assigned", Type: domain.GadgetAssignedToMe},
	}
	if got := gadgetsOf(sendAs(t, r, token, http.MethodPut, "/dashboard", body, http.StatusOK)); !reflect.DeepEqual(got, want) {
		t.Errorf("saved = %+v, want %+v", got, want)
	}
	if got := gadgetsOf(sendAs(t, r, token, http.MethodGet, "/dashboard", "", http.StatusOK)); !reflect.DeepEqual(got, want) {
		t.Errorf("reloaded = %+v, want %+v", got, want)
	}
	// ユーザーごとに保存する
	if got := gadgetsOf(sendAs(t, r, other, http.MethodGet, "/dashboard", "", http.StatusOK)); !reflect.DeepEqual(got, domain.DefaultDashboard()) {
		t.Errorf("other user's dashboard = %+v", got)
	}

	// フィルターを消すと、そのガジェットはフィルターが未選択になる
	sendAs(t, r, token, http.MethodDelete, fmt.Sprintf("/filters/%d", mine.ID), "", http.StatusOK)
	if got := gadgetsOf(sendAs(t, r, token, http.MethodGet, "/dashboard", "", http.StatusOK)); got[0].FilterID != 0 {
		t.Errorf("gadget of deleted filter = %+v", got[0])
	}

	// すべて外した空のダッシュボードも保存できる
	if got := gadgetsOf(sendAs(t, r, token, http.MethodPut, "/dashboard", `{"gadgets":[]}`, http.StatusOK)); len(got) != 0 {
		t.Errorf("empty dashboard = %+v", got)
	}
	if got := gadgetsOf(sendAs(t, r, token, http.MethodGet, "/dashboard", "", http.StatusOK)); got == nil || len(got) != 0 {
		t.Errorf("reloaded empty dashboard = %#v", got)
	}

	// 誤り
	for _, tt := range []struct {
		body string
		msg  string
	}{
		{`{"gadgets":[{"id":"a","type":"calendar"}]}`, "種類"},
		{`{"gadgets":[{"id":"a","type":"statusPie"},{"id":"a","type":"activityStream"}]}`, "ID"},
		{`{"gadgets":[{"id":"","type":"statusPie"}]}`, "ID"},
		{`{"gadgets":[{"id":"a","type":"statusPie","filterId":-1}]}`, "フィルター"},
		{`{"gadgets":[` + strings.TrimSuffix(strings.Repeat(`{"id":"x","type":"statusPie"},`, 21), ",") + `]}`, "20 個"},
	} {
		res := decode[struct{ Error string }](t, sendAs(t, r, token, http.MethodPut, "/dashboard", tt.body, http.StatusBadRequest))
		if !strings.Contains(res.Error, tt.msg) {
			t.Errorf("PUT %s error = %q, want %q", tt.body, res.Error, tt.msg)
		}
	}
	// 誤りでは前に保存した並びのまま
	if got := gadgetsOf(sendAs(t, r, token, http.MethodGet, "/dashboard", "", http.StatusOK)); len(got) != 0 {
		t.Errorf("dashboard after errors = %+v", got)
	}
}

func TestDashboard_GadgetData(t *testing.T) {
	r := newDashboardRouter(t)
	taro, token := signup(t, r, "山田 太郎", "taro@example.com")
	sendAs(t, r, token, http.MethodPost, "/spaces", `{"Title":"開発","Key":"DEV"}`, http.StatusCreated)
	sendAs(t, r, token, http.MethodPost, "/todos", `{"title":"ログイン画面","spaceId":1,"issueType":"Bug"}`, http.StatusCreated)
	sendAs(t, r, token, http.MethodPost, "/todos", `{"title":"API 実装","spaceId":1}`, http.StatusCreated)
	sendAs(t, r, token, http.MethodPost, "/todos", `{"title":"設計","spaceId":1}`, http.StatusCreated)
	sendAs(t, r, token, http.MethodPost, "/todos", `{"title":"消す課題","spaceId":1}`, http.StatusCreated)
	sendAs(t, r, token, http.MethodPut, "/todos/2", `{"title":"API 実装","status":"Closed","spaceId":1}`, http.StatusOK)
	sendAs(t, r, token, http.MethodPut, "/todos/3", `{"title":"設計","status":"InProgress","spaceId":1}`, http.StatusOK)
	sendAs(t, r, token, http.MethodDelete, "/todos/4", "", http.StatusOK)

	// 最近のアクティビティ: 新しい順で、課題のキーと要約、変更者が付く。消した課題の履歴は出ない
	activity := decode[[]domain.ActivityItem](t, sendAs(t, r, token, http.MethodGet, "/dashboard/activity", "", http.StatusOK))
	if len(activity) == 0 {
		t.Fatal("activity is empty")
	}
	first := activity[0]
	if first.Key != "DEV-3" || first.Title != "設計" || first.User == nil || first.User.ID != taro.ID {
		t.Errorf("latest activity = %+v", first)
	}
	for i, item := range activity {
		if item.TodoID == 4 {
			t.Errorf("activity of deleted issue = %+v", item)
		}
		if i > 0 && (item.CreatedAt.After(activity[i-1].CreatedAt) || item.CreatedAt.Equal(activity[i-1].CreatedAt) && item.ID > activity[i-1].ID) {
			t.Errorf("activity is not newest first: %+v after %+v", item, activity[i-1])
		}
	}
	if got := decode[[]domain.ActivityItem](t, sendAs(t, r, token, http.MethodGet, "/dashboard/activity?limit=2", "", http.StatusOK)); len(got) != 2 {
		t.Errorf("activity limit=2 = %d items", len(got))
	}
	for _, limit := range []string{"0", "101", "x"} {
		sendAs(t, r, token, http.MethodGet, "/dashboard/activity?limit="+limit, "", http.StatusBadRequest)
	}

	// ステータス別の件数（JQL で絞り込める）
	counts := decode[[]domain.StatusCount](t, sendAs(t, r, token, http.MethodGet, "/dashboard/status-counts", "", http.StatusOK))
	want := []domain.StatusCount{
		{Name: "オープン", Category: domain.StatusCategoryToDo, Count: 1},
		{Name: "進行中", Category: domain.StatusCategoryInProgress, Count: 1},
		{Name: "クローズ", Category: domain.StatusCategoryDone, Count: 1},
	}
	if !reflect.DeepEqual(counts, want) {
		t.Errorf("status counts = %+v, want %+v", counts, want)
	}
	bugs := decode[[]domain.StatusCount](t, sendAs(t, r, token, http.MethodGet, "/dashboard/status-counts?jql="+url.QueryEscape("type = Bug"), "", http.StatusOK))
	if !reflect.DeepEqual(bugs, want[:1]) {
		t.Errorf("status counts of bugs = %+v", bugs)
	}
	jqlErr := decode[struct {
		Error    string
		Position int
	}](t, sendAs(t, r, token, http.MethodGet, "/dashboard/status-counts?jql="+url.QueryEscape("type = Bug)"), "", http.StatusBadRequest))
	if jqlErr.Error == "" || jqlErr.Position != 10 {
		t.Errorf("JQL error = %+v", jqlErr)
	}

	// 作成 vs 解決: 今日に 3 件作成（消した課題は入らない）、1 件解決
	trend := decode[domain.CreatedVsResolved](t, sendAs(t, r, token, http.MethodGet, "/dashboard/created-vs-resolved?days=7", "", http.StatusOK))
	if len(trend.Days) != 7 || trend.Created != 3 || trend.Resolved != 1 {
		t.Fatalf("created vs resolved = %+v", trend)
	}
	today := trend.Days[6]
	if today.Date != time.Now().Format("2006-01-02") || today.Created != 3 || today.Resolved != 1 {
		t.Errorf("today = %+v", today)
	}
	if got := decode[domain.CreatedVsResolved](t, sendAs(t, r, token, http.MethodGet, "/dashboard/created-vs-resolved", "", http.StatusOK)); len(got.Days) != 30 {
		t.Errorf("default days = %d", len(got.Days))
	}
	closed := decode[domain.CreatedVsResolved](t, sendAs(t, r, token, http.MethodGet, "/dashboard/created-vs-resolved?jql="+url.QueryEscape("statusCategory = Done"), "", http.StatusOK))
	if closed.Created != 1 || closed.Resolved != 1 {
		t.Errorf("created vs resolved of done issues = %+v", closed)
	}
	for _, q := range []string{"days=0", "days=366", "days=x", "jql=" + url.QueryEscape("status = ")} {
		sendAs(t, r, token, http.MethodGet, "/dashboard/created-vs-resolved?"+q, "", http.StatusBadRequest)
	}
}
