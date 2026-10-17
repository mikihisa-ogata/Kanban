package service

import (
	"regexp"
	"strconv"
	"strings"
	"time"

	"todo-api/internal/domain"
	"todo-api/internal/repository"
)

type JQLService interface {
	// SearchJQL は JQL で課題を検索し、1 ページ分を返す。jql の誤り（構文・存在しない値など）は位置付きの *JQLError、
	// ページの指定の誤りは domain.ErrInvalidIssuePage。currentUser() は actorID のユーザー
	SearchJQL(jql string, page, perPage int, actorID int) (domain.IssueSearchResult, error)
	// MatchingIDs は JQL に合うすべての課題の ID を ID 順に返す（ORDER BY は使わない）。誤りは SearchJQL と同じ
	MatchingIDs(jql string, actorID int) ([]int, error)
}

type jqlService struct {
	todoRepo    repository.TodoRepository
	spaceRepo   repository.SpaceRepository
	statusRepo  repository.StatusRepository
	userRepo    repository.UserRepository
	epicRepo    repository.EpicRepository
	sprintRepo  repository.SprintRepository
	versionRepo repository.VersionRepository
	now         func() time.Time
	location    *time.Location
}

func NewJQLService(t repository.TodoRepository, s repository.SpaceRepository, st repository.StatusRepository, u repository.UserRepository, e repository.EpicRepository, sp repository.SprintRepository, v repository.VersionRepository) JQLService {
	return &jqlService{todoRepo: t, spaceRepo: s, statusRepo: st, userRepo: u, epicRepo: e, sprintRepo: sp, versionRepo: v, now: time.Now, location: time.Local}
}

func (s *jqlService) SearchJQL(jql string, page, perPage int, actorID int) (domain.IssueSearchResult, error) {
	search, err := s.buildSearch(jql, page, perPage, actorID)
	if err != nil {
		return domain.IssueSearchResult{}, err
	}
	return s.todoRepo.Search(search)
}

func (s *jqlService) MatchingIDs(jql string, actorID int) ([]int, error) {
	search, err := s.buildSearch(jql, 0, 0, actorID)
	if err != nil {
		return nil, err
	}
	return s.todoRepo.SearchIDs(search)
}

// buildSearch は JQL を課題の検索条件（Normalize 済み）にする
func (s *jqlService) buildSearch(jql string, page, perPage int, actorID int) (domain.IssueSearch, error) {
	q, err := parseJQL(jql)
	if err != nil {
		return domain.IssueSearch{}, err
	}
	ctx, err := s.loadContext(actorID)
	if err != nil {
		return domain.IssueSearch{}, err
	}
	search := domain.IssueSearch{Page: page, PerPage: perPage, ViewerID: actorID}
	if q.where != nil {
		filter, err := ctx.resolve(q.where)
		if err != nil {
			return domain.IssueSearch{}, err
		}
		search.Filter = &filter
	}
	if search.Order, err = resolveOrder(q.order); err != nil {
		return domain.IssueSearch{}, err
	}
	if err := search.Normalize(); err != nil {
		return domain.IssueSearch{}, err
	}
	return search, nil
}

// jqlContext は名前を ID に置き換えるための一覧と、currentUser()・now() の値
type jqlContext struct {
	spaces   []domain.Space
	statuses []domain.WorkflowStatus
	users    []domain.User
	epics    []domain.Epic
	sprints  []domain.Sprint
	versions []domain.Version
	actorID  int
	now      time.Time
	location *time.Location
}

func (s *jqlService) loadContext(actorID int) (*jqlContext, error) {
	ctx := &jqlContext{actorID: actorID, now: s.now(), location: s.location}
	var err error
	if ctx.spaces, err = s.spaceRepo.FindAll(); err != nil {
		return nil, err
	}
	if ctx.statuses, err = s.statusRepo.FindAll(); err != nil {
		return nil, err
	}
	if ctx.users, err = s.userRepo.FindAll(); err != nil {
		return nil, err
	}
	if ctx.epics, err = s.epicRepo.FindAll(); err != nil {
		return nil, err
	}
	if ctx.sprints, err = s.sprintRepo.FindAll(); err != nil {
		return nil, err
	}
	if ctx.versions, err = s.versionRepo.FindAll(); err != nil {
		return nil, err
	}
	return ctx, nil
}

// jqlFieldNames は JQL の項目名（小文字）から domain の項目への対応。Jira の別名も受け付ける
var jqlFieldNames = map[string]string{
	"project":        domain.FieldProject,
	"type":           domain.FieldType,
	"issuetype":      domain.FieldType,
	"status":         domain.FieldStatus,
	"statuscategory": domain.FieldStatusCategory,
	"priority":       domain.FieldPriority,
	"assignee":       domain.FieldAssignee,
	"reporter":       domain.FieldReporter,
	"labels":         domain.FieldLabels,
	"label":          domain.FieldLabels,
	"epic":           domain.FieldEpic,
	"epic link":      domain.FieldEpic,
	"sprint":         domain.FieldSprint,
	"fixversion":     domain.FieldFixVersion,
	"created":        domain.FieldCreated,
	"createddate":    domain.FieldCreated,
	"updated":        domain.FieldUpdated,
	"updateddate":    domain.FieldUpdated,
	"due":            domain.FieldDue,
	"duedate":        domain.FieldDue,
	"text":           domain.FieldText,
	"key":            domain.FieldKey,
	"issuekey":       domain.FieldKey,
	"issue":          domain.FieldKey,
}

// 項目ごとに使える演算子
var (
	jqlEmptyFields   = map[string]bool{domain.FieldAssignee: true, domain.FieldReporter: true, domain.FieldLabels: true, domain.FieldEpic: true, domain.FieldSprint: true, domain.FieldFixVersion: true, domain.FieldDue: true}
	jqlCompareFields = map[string]bool{domain.FieldPriority: true, domain.FieldCreated: true, domain.FieldUpdated: true, domain.FieldDue: true}
)

func (ctx *jqlContext) resolve(node *jqlNode) (domain.IssueFilter, error) {
	switch node.kind {
	case jqlAnd, jqlOr, jqlNot:
		op := map[jqlNodeKind]string{jqlAnd: domain.FilterAnd, jqlOr: domain.FilterOr, jqlNot: domain.FilterNot}[node.kind]
		f := domain.IssueFilter{Op: op}
		for _, child := range node.children {
			c, err := ctx.resolve(child)
			if err != nil {
				return domain.IssueFilter{}, err
			}
			f.Children = append(f.Children, c)
		}
		return f, nil
	}
	return ctx.resolveClause(node.clause)
}

func condition(field, op string, values ...any) domain.IssueFilter {
	return domain.IssueFilter{Op: domain.FilterCondition, Condition: &domain.IssueCondition{Field: field, Op: op, Values: values}}
}

func (ctx *jqlContext) resolveClause(c *jqlClause) (domain.IssueFilter, error) {
	field, ok := jqlFieldNames[strings.ToLower(c.field)]
	if !ok {
		return domain.IssueFilter{}, jqlErrorf(c.fieldPos, "項目「%s」はありません（使える項目: project, type, status, statusCategory, priority, assignee, reporter, labels, epic, sprint, fixVersion, created, updated, due, text, issuekey）", c.field)
	}

	switch c.op {
	case jqlOpIsEmpty, jqlOpIsNotEmpty:
		if !jqlEmptyFields[field] {
			return domain.IssueFilter{}, jqlErrorf(c.opPos, "%s には EMPTY を使えません（値のない課題はありません）", c.field)
		}
		if c.op == jqlOpIsEmpty {
			return condition(field, domain.CondEmpty), nil
		}
		return condition(field, domain.CondNotEmpty), nil
	case jqlOpContains:
		if field != domain.FieldText {
			return domain.IssueFilter{}, jqlErrorf(c.opPos, "~ は text だけに使えます（例: text ~ \"ログイン\"）")
		}
		v := c.values[0]
		if v.kind != jqlValueText || strings.TrimSpace(v.text) == "" {
			return domain.IssueFilter{}, jqlErrorf(v.pos, "text ~ には検索する文字列を指定してください")
		}
		return condition(field, domain.CondContains, strings.TrimSpace(v.text)), nil
	case jqlOpGt, jqlOpGe, jqlOpLt, jqlOpLe:
		if !jqlCompareFields[field] {
			return domain.IssueFilter{}, jqlErrorf(c.opPos, "%s には %s を使えません（大小を比べられるのは priority, created, updated, due）", c.field, c.op)
		}
	}
	if field == domain.FieldText {
		return domain.IssueFilter{}, jqlErrorf(c.opPos, "text には ~ を使ってください（例: text ~ \"ログイン\"）")
	}

	switch field {
	case domain.FieldKey:
		return ctx.resolveKey(c)
	case domain.FieldCreated, domain.FieldUpdated:
		return ctx.resolveDateTime(field, c)
	case domain.FieldDue:
		return ctx.resolveDue(c)
	}

	// = != IN NOT IN、または priority の大小
	values := []any{}
	hasEmpty := false
	for _, v := range c.values {
		if v.kind == jqlValueEmpty {
			if !jqlEmptyFields[field] {
				return domain.IssueFilter{}, jqlErrorf(v.pos, "%s には EMPTY を使えません（値のない課題はありません）", c.field)
			}
			hasEmpty = true
			continue
		}
		resolved, err := ctx.resolveValue(field, v)
		if err != nil {
			return domain.IssueFilter{}, err
		}
		values = append(values, resolved...)
	}

	if field == domain.FieldPriority && (c.op == jqlOpGt || c.op == jqlOpGe || c.op == jqlOpLt || c.op == jqlOpLe) {
		return condition(field, c.op, values[0]), nil
	}
	positive := c.op == jqlOpEq || c.op == jqlOpIn
	op := domain.CondIn
	if !positive {
		op = domain.CondNotIn
	}
	// IN (EMPTY, x) は「値がない、または x」、NOT IN (EMPTY, x) は「値があり、かつ x でない」
	if !hasEmpty {
		return condition(field, op, values...), nil
	}
	if len(values) == 0 {
		if positive {
			return condition(field, domain.CondEmpty), nil
		}
		return condition(field, domain.CondNotEmpty), nil
	}
	if positive {
		return domain.IssueFilter{Op: domain.FilterOr, Children: []domain.IssueFilter{condition(field, domain.CondEmpty), condition(field, op, values...)}}, nil
	}
	return condition(field, op, values...), nil // NOT IN は値のない課題に合わないので、EMPTY を外すのと同じ
}

// resolveValue は = や IN の値を、項目に合わせた ID や名前にする。名前に合うものが複数あればすべて返す
func (ctx *jqlContext) resolveValue(field string, v jqlValue) ([]any, error) {
	if v.kind == jqlValueFunc {
		return ctx.resolveFunc(field, v)
	}
	text := strings.TrimSpace(v.text)
	id, isNumber := 0, false
	if n, err := strconv.Atoi(text); err == nil && !v.quoted {
		id, isNumber = n, true
	}
	notFound := func(what string) ([]any, error) {
		return nil, jqlErrorf(v.pos, "%s「%s」は存在しません", what, v.text)
	}

	switch field {
	case domain.FieldProject:
		if strings.EqualFold(text, domain.UnassignedKeyPrefix) {
			return []any{domain.UnassignedKeyPrefix}, nil
		}
		for _, sp := range ctx.spaces {
			if strings.EqualFold(sp.Key, text) || strings.EqualFold(sp.Title, text) {
				return []any{sp.Key}, nil
			}
		}
		return notFound("プロジェクト")
	case domain.FieldType:
		if t, ok := jqlIssueTypes[strings.ToLower(text)]; ok {
			return []any{t}, nil
		}
		return notFound("課題タイプ")
	case domain.FieldStatus:
		names := []any{}
		for _, st := range ctx.statuses {
			if strings.EqualFold(st.Name, text) && !containsAny(names, st.Name) {
				names = append(names, st.Name)
			}
		}
		if len(names) == 0 {
			return notFound("ステータス")
		}
		return names, nil
	case domain.FieldStatusCategory:
		if c, ok := jqlStatusCategories[strings.ToLower(strings.ReplaceAll(text, " ", ""))]; ok {
			return []any{c}, nil
		}
		return notFound("ステータスのカテゴリ")
	case domain.FieldPriority:
		if p, ok := jqlPriorities[strings.ToLower(text)]; ok {
			return []any{p}, nil
		}
		return notFound("優先度")
	case domain.FieldAssignee, domain.FieldReporter:
		ids := []any{}
		for _, u := range ctx.users {
			if (isNumber && u.ID == id) || strings.EqualFold(u.Email, text) || strings.EqualFold(u.DisplayName, text) {
				ids = append(ids, u.ID)
			}
		}
		if len(ids) == 0 {
			return notFound("ユーザー")
		}
		return ids, nil
	case domain.FieldLabels:
		if text == "" {
			return nil, jqlErrorf(v.pos, "ラベルの名前を指定してください")
		}
		return []any{text}, nil
	case domain.FieldEpic:
		ids := []any{}
		for _, e := range ctx.epics {
			if (isNumber && e.ID == id) || strings.EqualFold(e.Title, text) {
				ids = append(ids, e.ID)
			}
		}
		if len(ids) == 0 {
			return notFound("エピック")
		}
		return ids, nil
	case domain.FieldSprint:
		ids := []any{}
		for _, sp := range ctx.sprints {
			if (isNumber && sp.ID == id) || strings.EqualFold(sp.Name, text) {
				ids = append(ids, sp.ID)
			}
		}
		if len(ids) == 0 {
			return notFound("スプリント")
		}
		return ids, nil
	case domain.FieldFixVersion:
		ids := []any{}
		for _, ver := range ctx.versions {
			if (isNumber && ver.ID == id) || strings.EqualFold(ver.Name, text) {
				ids = append(ids, ver.ID)
			}
		}
		if len(ids) == 0 {
			return notFound("バージョン")
		}
		return ids, nil
	}
	return nil, jqlErrorf(v.pos, "%s には値を指定できません", field)
}

// resolveFunc は値の位置の関数（currentUser()・openSprints() など）を ID にする
func (ctx *jqlContext) resolveFunc(field string, v jqlValue) ([]any, error) {
	name := strings.ToLower(v.text)
	switch {
	case name == "currentuser" && (field == domain.FieldAssignee || field == domain.FieldReporter):
		if ctx.actorID == 0 {
			return nil, jqlErrorf(v.pos, "currentUser() はログインしているときだけ使えます")
		}
		return []any{ctx.actorID}, nil
	case field == domain.FieldSprint && (name == "opensprints" || name == "closedsprints" || name == "futuresprints"):
		state := map[string]domain.SprintState{"opensprints": domain.SprintStateActive, "closedsprints": domain.SprintStateClosed, "futuresprints": domain.SprintStateFuture}[name]
		ids := []any{}
		for _, sp := range ctx.sprints {
			if sp.State == state {
				ids = append(ids, sp.ID)
			}
		}
		return ids, nil
	case field == domain.FieldFixVersion && (name == "releasedversions" || name == "unreleasedversions"):
		status := domain.VersionReleased
		if name == "unreleasedversions" {
			status = domain.VersionUnreleased
		}
		ids := []any{}
		for _, ver := range ctx.versions {
			if ver.Status == status {
				ids = append(ids, ver.ID)
			}
		}
		return ids, nil
	}
	return nil, jqlErrorf(v.pos, "%s には関数 %s() を使えません", field, v.text)
}

// resolveKey は issuekey の条件にする。値は課題キー（例: DEV-12）か issueHistory()（自分が表示した課題）
func (ctx *jqlContext) resolveKey(c *jqlClause) (domain.IssueFilter, error) {
	keys := []any{}
	history := false
	for _, v := range c.values {
		switch {
		case v.kind == jqlValueEmpty:
			return domain.IssueFilter{}, jqlErrorf(v.pos, "%s には EMPTY を使えません（値のない課題はありません）", c.field)
		case v.kind == jqlValueFunc && strings.EqualFold(v.text, "issueHistory"):
			if ctx.actorID == 0 {
				return domain.IssueFilter{}, jqlErrorf(v.pos, "issueHistory() はログインしているときだけ使えます")
			}
			history = true
		case v.kind == jqlValueFunc:
			return domain.IssueFilter{}, jqlErrorf(v.pos, "%s には関数 %s() を使えません（使える関数: issueHistory()）", c.field, v.text)
		default:
			key := strings.ToUpper(strings.TrimSpace(v.text))
			if key == "" {
				return domain.IssueFilter{}, jqlErrorf(v.pos, "課題キーを指定してください（例: DEV-12）")
			}
			keys = append(keys, key)
		}
	}
	matched := domain.IssueFilter{Op: domain.FilterOr}
	if len(keys) > 0 {
		matched.Children = append(matched.Children, condition(domain.FieldKey, domain.CondIn, keys...))
	}
	if history {
		matched.Children = append(matched.Children, condition(domain.FieldViewedBy, domain.CondIn, ctx.actorID))
	}
	if len(matched.Children) == 1 {
		matched = matched.Children[0]
	}
	if c.op == jqlOpEq || c.op == jqlOpIn {
		return matched, nil
	}
	return domain.IssueFilter{Op: domain.FilterNot, Children: []domain.IssueFilter{matched}}, nil
}

// relativeDatePattern は相対日付（例: -7d・2w・-1h・+30m）。w は週、d は日、h は時間、m は分
var relativeDatePattern = regexp.MustCompile(`^([-+]?)(\d+)([wdhm])$`)

// jqlTimeRange は日時の値が表す範囲 [start, end)。日付だけなら 1 日、時刻付きなら 1 分、now() や相対日付はその瞬間
type jqlTimeRange struct {
	start, end time.Time
	dateOnly   bool
}

func (ctx *jqlContext) timeValue(v jqlValue) (jqlTimeRange, error) {
	if v.kind == jqlValueFunc {
		switch strings.ToLower(v.text) {
		case "now":
			return jqlTimeRange{start: ctx.now, end: ctx.now}, nil
		case "startofday":
			y, m, d := ctx.now.In(ctx.location).Date()
			start := time.Date(y, m, d, 0, 0, 0, 0, ctx.location)
			return jqlTimeRange{start: start, end: start, dateOnly: true}, nil
		}
		return jqlTimeRange{}, jqlErrorf(v.pos, "日付には関数 %s() を使えません（使える関数: now(), startOfDay()）", v.text)
	}
	if v.kind != jqlValueText {
		return jqlTimeRange{}, jqlErrorf(v.pos, "日付が必要です")
	}
	text := strings.TrimSpace(v.text)
	if m := relativeDatePattern.FindStringSubmatch(text); m != nil {
		n, _ := strconv.Atoi(m[2])
		unit := map[string]time.Duration{"w": 7 * 24 * time.Hour, "d": 24 * time.Hour, "h": time.Hour, "m": time.Minute}[m[3]]
		offset := time.Duration(n) * unit
		if m[1] == "-" {
			offset = -offset
		}
		t := ctx.now.Add(offset)
		return jqlTimeRange{start: t, end: t}, nil
	}
	for _, layout := range []string{"2006-01-02", "2006/01/02"} {
		if t, err := time.ParseInLocation(layout, text, ctx.location); err == nil {
			return jqlTimeRange{start: t, end: t.AddDate(0, 0, 1), dateOnly: true}, nil
		}
	}
	for _, layout := range []string{"2006-01-02 15:04", "2006/01/02 15:04"} {
		if t, err := time.ParseInLocation(layout, text, ctx.location); err == nil {
			return jqlTimeRange{start: t, end: t.Add(time.Minute)}, nil
		}
	}
	return jqlTimeRange{}, jqlErrorf(v.pos, "日付「%s」を読めません（例: \"2026-10-01\"、\"2026-10-01 09:00\"、-7d、now()）", v.text)
}

// resolveDateTime は created・updated の条件を、日時の範囲の比較にする
func (ctx *jqlContext) resolveDateTime(field string, c *jqlClause) (domain.IssueFilter, error) {
	ranges := []jqlTimeRange{}
	for _, v := range c.values {
		if v.kind == jqlValueEmpty {
			return domain.IssueFilter{}, jqlErrorf(v.pos, "%s には EMPTY を使えません（値のない課題はありません）", c.field)
		}
		r, err := ctx.timeValue(v)
		if err != nil {
			return domain.IssueFilter{}, err
		}
		ranges = append(ranges, r)
	}
	within := func(r jqlTimeRange) domain.IssueFilter {
		if !r.end.After(r.start) {
			r.end = r.start.Add(time.Second)
		}
		return domain.IssueFilter{Op: domain.FilterAnd, Children: []domain.IssueFilter{
			condition(field, domain.CondGe, r.start.UTC()), condition(field, domain.CondLt, r.end.UTC()),
		}}
	}
	switch c.op {
	case jqlOpGt:
		return condition(field, domain.CondGe, ranges[0].end.UTC()), nil
	case jqlOpGe:
		return condition(field, domain.CondGe, ranges[0].start.UTC()), nil
	case jqlOpLt:
		return condition(field, domain.CondLt, ranges[0].start.UTC()), nil
	case jqlOpLe:
		end := ranges[0].end
		if !ranges[0].dateOnly && !end.After(ranges[0].start) {
			end = end.Add(time.Second)
		}
		return condition(field, domain.CondLt, end.UTC()), nil
	}
	matched := domain.IssueFilter{Op: domain.FilterOr}
	for _, r := range ranges {
		matched.Children = append(matched.Children, within(r))
	}
	if len(matched.Children) == 1 {
		matched = matched.Children[0]
	}
	if c.op == jqlOpEq || c.op == jqlOpIn {
		return matched, nil
	}
	return domain.IssueFilter{Op: domain.FilterNot, Children: []domain.IssueFilter{matched}}, nil
}

// resolveDue は期限（日付だけの項目）の条件にする。値は日付（時刻は切り捨て）
func (ctx *jqlContext) resolveDue(c *jqlClause) (domain.IssueFilter, error) {
	dates := []any{}
	hasEmpty := false
	for _, v := range c.values {
		if v.kind == jqlValueEmpty {
			hasEmpty = true
			continue
		}
		r, err := ctx.timeValue(v)
		if err != nil {
			return domain.IssueFilter{}, err
		}
		dates = append(dates, r.start.In(ctx.location).Format("2006-01-02"))
	}
	switch c.op {
	case jqlOpGt, jqlOpGe, jqlOpLt, jqlOpLe:
		return condition(domain.FieldDue, c.op, dates[0]), nil
	}
	positive := c.op == jqlOpEq || c.op == jqlOpIn
	op := domain.CondIn
	if !positive {
		op = domain.CondNotIn
	}
	switch {
	case !hasEmpty:
		return condition(domain.FieldDue, op, dates...), nil
	case len(dates) == 0 && positive:
		return condition(domain.FieldDue, domain.CondEmpty), nil
	case len(dates) == 0:
		return condition(domain.FieldDue, domain.CondNotEmpty), nil
	case positive:
		return domain.IssueFilter{Op: domain.FilterOr, Children: []domain.IssueFilter{condition(domain.FieldDue, domain.CondEmpty), condition(domain.FieldDue, op, dates...)}}, nil
	}
	return condition(domain.FieldDue, op, dates...), nil
}

// jqlOrderFields は ORDER BY の項目名（小文字）から並べ替えの列への対応
var jqlOrderFields = map[string]string{
	"key":         domain.IssueSortKey,
	"issuekey":    domain.IssueSortKey,
	"project":     domain.IssueSortKey,
	"type":        domain.IssueSortType,
	"issuetype":   domain.IssueSortType,
	"summary":     domain.IssueSortSummary,
	"status":      domain.IssueSortStatus,
	"assignee":    domain.IssueSortAssignee,
	"reporter":    domain.IssueSortReporter,
	"priority":    domain.IssueSortPriority,
	"created":     domain.IssueSortCreated,
	"createddate": domain.IssueSortCreated,
	"updated":     domain.IssueSortUpdated,
	"updateddate": domain.IssueSortUpdated,
	"due":         domain.IssueSortDue,
	"duedate":     domain.IssueSortDue,
	"lastviewed":  domain.IssueSortLastViewed,
}

// resolveOrder は ORDER BY を並べ替えの列にする。priority は Jira と同じく DESC が高い順
// （並べ替えの列 priority は昇順が高い順なので、向きを逆にする）
func resolveOrder(items []jqlOrderItem) ([]domain.IssueOrder, error) {
	order := []domain.IssueOrder{}
	for _, item := range items {
		field, ok := jqlOrderFields[strings.ToLower(item.field)]
		if !ok {
			return nil, jqlErrorf(item.pos, "「%s」では並べ替えられません（使える項目: key, type, summary, status, assignee, reporter, priority, created, updated, due, lastViewed）", item.field)
		}
		desc := item.desc
		if field == domain.IssueSortPriority {
			desc = !desc
		}
		order = append(order, domain.IssueOrder{Field: field, Desc: desc})
	}
	return order, nil
}

var jqlIssueTypes = map[string]domain.IssueType{
	"task": domain.IssueTypeTask, "タスク": domain.IssueTypeTask,
	"bug": domain.IssueTypeBug, "バグ": domain.IssueTypeBug,
	"story": domain.IssueTypeStory, "ストーリー": domain.IssueTypeStory,
	"subtask": domain.IssueTypeSubtask, "sub-task": domain.IssueTypeSubtask, "サブタスク": domain.IssueTypeSubtask,
}

var jqlStatusCategories = map[string]domain.StatusCategory{
	"todo": domain.StatusCategoryToDo, "未着手": domain.StatusCategoryToDo,
	"inprogress": domain.StatusCategoryInProgress, "進行中": domain.StatusCategoryInProgress,
	"done": domain.StatusCategoryDone, "完了": domain.StatusCategoryDone,
}

var jqlPriorities = map[string]domain.Priority{
	"highest": domain.PriorityHighest, "最高": domain.PriorityHighest,
	"high": domain.PriorityHigh, "高": domain.PriorityHigh,
	"medium": domain.PriorityMedium, "中": domain.PriorityMedium,
	"low": domain.PriorityLow, "低": domain.PriorityLow,
	"lowest": domain.PriorityLowest, "最低": domain.PriorityLowest,
}

func containsAny(list []any, v any) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}
