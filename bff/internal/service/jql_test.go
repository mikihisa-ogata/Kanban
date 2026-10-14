package service

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"todo-api/internal/domain"
)

// formatFilter は絞り込みの木を読みやすい文字列にする（テスト用）。日時は UTC の RFC3339 で書く
func formatFilter(f domain.IssueFilter) string {
	switch f.Op {
	case domain.FilterAnd, domain.FilterOr:
		parts := []string{}
		for _, c := range f.Children {
			parts = append(parts, formatFilter(c))
		}
		return "(" + strings.Join(parts, " "+strings.ToUpper(f.Op)+" ") + ")"
	case domain.FilterNot:
		return "NOT " + formatFilter(f.Children[0])
	}
	c := f.Condition
	values := []string{}
	for _, v := range c.Values {
		if tm, ok := v.(time.Time); ok {
			values = append(values, tm.UTC().Format(time.RFC3339))
			continue
		}
		values = append(values, fmt.Sprint(v))
	}
	if len(values) == 0 {
		return fmt.Sprintf("%s %s", c.Field, c.Op)
	}
	return fmt.Sprintf("%s %s %s", c.Field, c.Op, strings.Join(values, ","))
}

// testJQLContext は 2026-10-14 12:00（JST）時点の、ユーザー 1 がログインしている JQL の文脈
func testJQLContext() *jqlContext {
	jst := time.FixedZone("JST", 9*60*60)
	return &jqlContext{
		spaces: []domain.Space{{ID: 1, Title: "開発", Key: "DEV"}, {ID: 2, Title: "運用", Key: "OPS"}},
		statuses: []domain.WorkflowStatus{
			{ID: 1, SpaceID: 1, Name: "オープン", Category: domain.StatusCategoryToDo},
			{ID: 2, SpaceID: 1, Name: "Done", Category: domain.StatusCategoryDone},
			{ID: 3, SpaceID: 2, Name: "done", Category: domain.StatusCategoryDone},
		},
		users: []domain.User{{ID: 1, DisplayName: "山田 太郎", Email: "taro@example.com"}, {ID: 2, DisplayName: "佐藤 花子", Email: "hanako@example.com"}, {ID: 3, DisplayName: "山田 太郎", Email: "taro2@example.com"}},
		epics: []domain.Epic{{ID: 5, Title: "認証"}},
		sprints: []domain.Sprint{
			{ID: 1, Name: "Sprint 1", State: domain.SprintStateClosed},
			{ID: 2, Name: "Sprint 2", State: domain.SprintStateActive},
			{ID: 3, Name: "Sprint 3", State: domain.SprintStateActive},
			{ID: 4, Name: "Sprint 4", State: domain.SprintStateFuture},
		},
		versions: []domain.Version{{ID: 7, Name: "v1.0", Status: domain.VersionReleased}, {ID: 8, Name: "v1.1", Status: domain.VersionUnreleased}},
		actorID:  1,
		now:      time.Date(2026, 10, 14, 12, 0, 0, 0, jst),
		location: jst,
	}
}

func resolveJQL(ctx *jqlContext, query string) (string, error) {
	q, err := parseJQL(query)
	if err != nil {
		return "", err
	}
	f, err := ctx.resolve(q.where)
	if err != nil {
		return "", err
	}
	return formatFilter(f), nil
}

func TestResolveJQL(t *testing.T) {
	tests := []struct {
		query string
		want  string
	}{
		// project はキーかスペース名（大文字・小文字は区別しない）。TASK はスペース未割り当て
		{"project = dev", "project in DEV"},
		{"project = 運用", "project in OPS"},
		{"project IN (DEV, TASK)", "project in DEV,TASK"},
		{"project != DEV", "project not in DEV"},
		// type・priority・statusCategory は英語と日本語の名前
		{"type = bug", "type in Bug"},
		{"issuetype IN (Story, サブタスク)", "type in Story,Subtask"},
		{"priority = 高", "priority in High"},
		{"priority NOT IN (Low, Lowest)", "priority not in Low,Lowest"},
		{"priority >= High", "priority >= High"},
		{"statusCategory = \"In Progress\"", "statusCategory in InProgress"},
		{"statusCategory != 完了", "statusCategory not in Done"},
		// status はスペースをまたいで同じ名前（大文字・小文字は区別しない）をすべて選ぶ
		{"status = DONE", "status in Done,done"},
		{"status IN (オープン, Done)", "status in オープン,Done,done"},
		// assignee・reporter はユーザー ID・メール・表示名（同じ表示名はすべて）・currentUser()
		{"assignee = currentUser()", "assignee in 1"},
		{"assignee = \"山田 太郎\"", "assignee in 1,3"},
		{"reporter = hanako@example.com", "reporter in 2"},
		{"assignee = 2", "assignee in 2"},
		{"assignee IS EMPTY", "assignee empty"},
		{"assignee IS NOT EMPTY", "assignee not empty"},
		{"assignee IN (EMPTY, currentUser())", "(assignee empty OR assignee in 1)"},
		{"assignee NOT IN (EMPTY, 2)", "assignee not in 2"},
		{"assignee IN (EMPTY)", "assignee empty"},
		{"assignee NOT IN (EMPTY)", "assignee not empty"},
		// labels は名前をそのまま、epic・sprint・fixVersion は名前か ID
		{"labels = UI", "labels in UI"},
		{"labels IS EMPTY", "labels empty"},
		{"epic = 認証", "epic in 5"},
		{"\"Epic Link\" = 5", "epic in 5"},
		{"sprint = \"sprint 2\"", "sprint in 2"},
		{"sprint IN (openSprints())", "sprint in 2,3"},
		{"sprint = closedSprints()", "sprint in 1"},
		{"sprint = futureSprints()", "sprint in 4"},
		{"sprint IS EMPTY", "sprint empty"},
		{"fixVersion = v1.0", "fixVersion in 7"},
		{"fixVersion IN (unreleasedVersions())", "fixVersion in 8"},
		{"fixVersion = releasedVersions()", "fixVersion in 7"},
		// text は ~ で、前後の空白は除く
		{"text ~ \" ログイン \"", "text contains ログイン"},
		// created・updated は日時の範囲。日付だけならその日（JST）の 0 時から翌日 0 時
		{"created = 2026-10-01", "(created >= 2026-09-30T15:00:00Z AND created < 2026-10-01T15:00:00Z)"},
		{"created > 2026-10-01", "created >= 2026-10-01T15:00:00Z"},
		{"created >= 2026/10/01", "created >= 2026-09-30T15:00:00Z"},
		{"created < 2026-10-01", "created < 2026-09-30T15:00:00Z"},
		{"created <= 2026-10-01", "created < 2026-10-01T15:00:00Z"},
		{"updated >= \"2026-10-01 09:30\"", "updated >= 2026-10-01T00:30:00Z"},
		{"updated > \"2026-10-01 09:30\"", "updated >= 2026-10-01T00:31:00Z"},
		{"created != 2026-10-01", "NOT (created >= 2026-09-30T15:00:00Z AND created < 2026-10-01T15:00:00Z)"},
		{"created IN (2026-10-01, 2026-10-03)", "((created >= 2026-09-30T15:00:00Z AND created < 2026-10-01T15:00:00Z) OR (created >= 2026-10-02T15:00:00Z AND created < 2026-10-03T15:00:00Z))"},
		// 相対日付と now()・startOfDay()
		{"created >= -7d", "created >= 2026-10-07T03:00:00Z"},
		{"created >= -2w", "created >= 2026-09-30T03:00:00Z"},
		{"updated > -1h", "updated >= 2026-10-14T02:00:00Z"},
		{"updated > -30m", "updated >= 2026-10-14T02:30:00Z"},
		{"created < now()", "created < 2026-10-14T03:00:00Z"},
		{"created <= now()", "created < 2026-10-14T03:00:01Z"},
		{"created >= startOfDay()", "created >= 2026-10-13T15:00:00Z"},
		// due は日付。相対日付は日付に切り捨てる
		{"due = 2026-10-20", "due in 2026-10-20"},
		{"due < now()", "due < 2026-10-14"},
		{"due <= 7d", "due <= 2026-10-21"},
		{"due >= -1d", "due >= 2026-10-13"},
		{"due IS EMPTY", "due empty"},
		{"due IN (EMPTY, 2026-10-20)", "(due empty OR due in 2026-10-20)"},
		{"due NOT IN (2026-10-20)", "due not in 2026-10-20"},
		// 項目名は大文字・小文字を区別せず、Jira の別名も使える
		{"STATUS = done", "status in Done,done"},
		{"duedate IS NOT EMPTY", "due not empty"},
		// AND・OR・NOT はそのまま木になる
		{"project = DEV AND (type = Bug OR NOT priority = Low)", "(project in DEV AND (type in Bug OR NOT priority in Low))"},
	}
	ctx := testJQLContext()
	for _, tt := range tests {
		got, err := resolveJQL(ctx, tt.query)
		if err != nil {
			t.Errorf("%q error = %v", tt.query, err)
			continue
		}
		if got != tt.want {
			t.Errorf("%q = %s, want %s", tt.query, got, tt.want)
		}
	}
}

func TestResolveJQL_Errors(t *testing.T) {
	tests := []struct {
		query string
		pos   int
		msg   string
	}{
		{"foo = 1", 0, "項目「foo」はありません"},
		{"status = Done AND bar = 1", 18, "項目「bar」"},
		{"project = NONE", 10, "プロジェクト「NONE」は存在しません"},
		{"type = Epic", 7, "課題タイプ「Epic」"},
		{"status = 削除済み", 9, "ステータス「削除済み」"},
		{"statusCategory = Waiting", 17, "ステータスのカテゴリ"},
		{"priority = Urgent", 11, "優先度「Urgent」"},
		{"priority > Urgent", 11, "優先度「Urgent」"},
		{"assignee = nobody@example.com", 11, "ユーザー"},
		{"assignee = \"2\"", 11, "ユーザー「2」"}, // 引用符で囲んだ数字は ID ではなく名前
		{"epic = 存在しない", 7, "エピック"},
		{"sprint = 99", 9, "スプリント"},
		{"fixVersion = v9", 13, "バージョン"},
		{"labels = \"  \"", 9, "ラベルの名前"},
		{"sprint IN openSprints()", 10, "値の一覧"},
		// 項目と演算子・関数の組み合わせ
		{"status IS EMPTY", 7, "EMPTY を使えません"},
		{"project = EMPTY", 8, "EMPTY を使えません"},
		{"type IN (EMPTY, Bug)", 9, "EMPTY を使えません"},
		{"created IS EMPTY", 8, "EMPTY を使えません"},
		{"created IN (EMPTY)", 12, "EMPTY を使えません"},
		{"status ~ Done", 7, "~ は text だけ"},
		{"text = ログイン", 5, "~ を使って"},
		{"text IN (a)", 5, "~ を使って"},
		{"text IS EMPTY", 5, "EMPTY を使えません"},
		{"text ~ \"  \"", 7, "検索する文字列"},
		{"text ~ now()", 7, "検索する文字列"},
		{"status > Done", 7, "> を使えません"},
		{"assignee <= 1", 9, "<= を使えません"},
		{"status = currentUser()", 9, "関数 currentUser()"},
		{"assignee = openSprints()", 11, "関数 openSprints()"},
		{"sprint = releasedVersions()", 9, "関数 releasedVersions()"},
		{"created > yesterday()", 10, "関数 yesterday()"},
		{"created > 2026-13-01", 10, "日付「2026-13-01」を読めません"},
		{"created > -7x", 10, "日付「-7x」"},
		{"due = 明日", 6, "日付「明日」"},
	}
	ctx := testJQLContext()
	for _, tt := range tests {
		_, err := resolveJQL(ctx, tt.query)
		var jqlErr *JQLError
		if !errors.As(err, &jqlErr) {
			t.Errorf("%q error = %v, want *JQLError", tt.query, err)
			continue
		}
		if jqlErr.Pos != tt.pos || !strings.Contains(jqlErr.Msg, tt.msg) {
			t.Errorf("%q error = %d: %s, want %d: …%s…", tt.query, jqlErr.Pos, jqlErr.Msg, tt.pos, tt.msg)
		}
	}

	// ログインしていなければ currentUser() は使えない
	ctx.actorID = 0
	if _, err := resolveJQL(ctx, "assignee = currentUser()"); err == nil || !strings.Contains(err.Error(), "ログイン") {
		t.Errorf("currentUser() without login error = %v", err)
	}
}

func TestResolveOrder(t *testing.T) {
	q, err := parseJQL("ORDER BY priority DESC, Created, duedate desc, issuekey ASC")
	if err != nil {
		t.Fatal(err)
	}
	got, err := resolveOrder(q.order)
	// priority は DESC が高い順（並べ替えの列 priority は昇順が高い順）
	want := []domain.IssueOrder{
		{Field: domain.IssueSortPriority, Desc: false},
		{Field: domain.IssueSortCreated, Desc: false},
		{Field: domain.IssueSortDue, Desc: true},
		{Field: domain.IssueSortKey, Desc: false},
	}
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Errorf("resolveOrder = %+v, %v, want %+v", got, err, want)
	}

	q, _ = parseJQL("status = Done ORDER BY labels")
	_, err = resolveOrder(q.order)
	var jqlErr *JQLError
	if !errors.As(err, &jqlErr) || jqlErr.Pos != 23 || !strings.Contains(jqlErr.Msg, "「labels」では並べ替えられません") {
		t.Errorf("resolveOrder(labels) error = %v", err)
	}
}
