package repository

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"todo-api/internal/domain"
)

func cond(field, op string, values ...any) domain.IssueFilter {
	return domain.IssueFilter{Op: domain.FilterCondition, Condition: &domain.IssueCondition{Field: field, Op: op, Values: values}}
}

func TestIssueFilterSQL(t *testing.T) {
	created := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	tests := []struct {
		name   string
		filter domain.IssueFilter
		sql    string
		args   []any
	}{
		{"project", cond(domain.FieldProject, domain.CondIn, "DEV", "TASK"), "COALESCE(s.space_key, 'TASK') IN (?, ?)", []any{"DEV", "TASK"}},
		{"type は文字列にする", cond(domain.FieldType, domain.CondNotIn, domain.IssueTypeBug), "t.issue_type NOT IN (?)", []any{"Bug"}},
		{"status", cond(domain.FieldStatus, domain.CondIn, "Done"), "COALESCE(st.name, '') IN (?)", []any{"Done"}},
		{"statusCategory", cond(domain.FieldStatusCategory, domain.CondIn, domain.StatusCategoryDone), "COALESCE(st.category, '') IN (?)", []any{"Done"}},
		{"priority", cond(domain.FieldPriority, domain.CondIn, domain.PriorityHigh), "t.priority IN (?)", []any{"High"}},
		// 優先度の大小は高いほど大きい。FIELD の位置（高いほど小さい）と逆に比べる
		{"priority >", cond(domain.FieldPriority, domain.CondGt, domain.PriorityMedium), "(" + priorityRankSQL + " > 0 AND " + priorityRankSQL + " < ?)", []any{3}},
		{"priority <=", cond(domain.FieldPriority, domain.CondLe, domain.PriorityHigh), "(" + priorityRankSQL + " > 0 AND " + priorityRankSQL + " >= ?)", []any{2}},
		// ID の項目は 0 が値なし。NOT IN は値のない課題に合わない
		{"assignee IN", cond(domain.FieldAssignee, domain.CondIn, 1, 2), "(t.assignee_id <> 0 AND t.assignee_id IN (?, ?))", []any{1, 2}},
		{"assignee NOT IN", cond(domain.FieldAssignee, domain.CondNotIn, 1), "(t.assignee_id <> 0 AND t.assignee_id NOT IN (?))", []any{1}},
		{"reporter EMPTY", cond(domain.FieldReporter, domain.CondEmpty), "(NOT t.reporter_id <> 0)", nil},
		{"epic NOT EMPTY", cond(domain.FieldEpic, domain.CondNotEmpty), "t.epic_id <> 0", nil},
		// 値のない IN（開始中のスプリントがないときの openSprints() など）はどの課題にも合わない
		{"sprint IN ()", cond(domain.FieldSprint, domain.CondIn), "FALSE", nil},
		{"sprint NOT IN ()", cond(domain.FieldSprint, domain.CondNotIn), "(t.sprint_id <> 0 AND TRUE)", nil},
		// ラベル・修正バージョンは副問い合わせ。ラベルは大文字・小文字を区別しない
		{"labels IN", cond(domain.FieldLabels, domain.CondIn, "UI"),
			"t.id IN (SELECT tl.todo_id FROM todo_labels tl JOIN labels l ON l.id = tl.label_id WHERE LOWER(l.name) IN (?))", []any{"ui"}},
		{"labels NOT IN", cond(domain.FieldLabels, domain.CondNotIn, "UI"),
			"(t.id IN (SELECT tl.todo_id FROM todo_labels tl JOIN labels l ON l.id = tl.label_id) AND t.id NOT IN (SELECT tl.todo_id FROM todo_labels tl JOIN labels l ON l.id = tl.label_id WHERE LOWER(l.name) IN (?)))", []any{"ui"}},
		{"labels EMPTY", cond(domain.FieldLabels, domain.CondEmpty), "t.id NOT IN (SELECT tl.todo_id FROM todo_labels tl JOIN labels l ON l.id = tl.label_id)", nil},
		{"fixVersion IN", cond(domain.FieldFixVersion, domain.CondIn, 7), "t.id IN (SELECT tv.todo_id FROM todo_versions tv WHERE tv.version_id IN (?))", []any{7}},
		{"fixVersion NOT EMPTY", cond(domain.FieldFixVersion, domain.CondNotEmpty), "t.id IN (SELECT tv.todo_id FROM todo_versions tv)", nil},
		{"fixVersion IN ()", cond(domain.FieldFixVersion, domain.CondIn), "FALSE", nil},
		// 日時・期限
		{"created >=", cond(domain.FieldCreated, domain.CondGe, created), "t.created_at >= ?", []any{created}},
		{"updated <", cond(domain.FieldUpdated, domain.CondLt, created), "t.updated_at < ?", []any{created}},
		{"due >", cond(domain.FieldDue, domain.CondGt, "2026-10-20"), "(t.deadline IS NOT NULL AND t.deadline > ?)", []any{"2026-10-20"}},
		{"due IN", cond(domain.FieldDue, domain.CondIn, "2026-10-20"), "(t.deadline IS NOT NULL AND t.deadline IN (?))", []any{"2026-10-20"}},
		{"due EMPTY", cond(domain.FieldDue, domain.CondEmpty), "(NOT t.deadline IS NOT NULL)", nil},
		// text は要約・説明・課題キー。LIKE の特殊文字はそのままの文字
		{"text", cond(domain.FieldText, domain.CondContains, " 100%_ "),
			"(t.title LIKE ? OR t.description LIKE ? OR CONCAT(COALESCE(s.space_key, 'TASK'), '-', t.number) = ?)", []any{`%100\%\_%`, `%100\%\_%`, "100%_"}},
		// AND・OR・NOT は括弧で囲む。引数は式の順
		{"and/or/not", domain.IssueFilter{Op: domain.FilterAnd, Children: []domain.IssueFilter{
			cond(domain.FieldProject, domain.CondIn, "DEV"),
			{Op: domain.FilterOr, Children: []domain.IssueFilter{
				cond(domain.FieldType, domain.CondIn, domain.IssueTypeBug),
				{Op: domain.FilterNot, Children: []domain.IssueFilter{cond(domain.FieldEpic, domain.CondIn, 5)}},
			}},
		}}, "(COALESCE(s.space_key, 'TASK') IN (?) AND (t.issue_type IN (?) OR (NOT (t.epic_id <> 0 AND t.epic_id IN (?)))))", []any{"DEV", "Bug", 5}},
	}
	for _, tt := range tests {
		sql, args, err := issueFilterSQL(tt.filter)
		if err != nil {
			t.Errorf("%s: error = %v", tt.name, err)
			continue
		}
		if sql != tt.sql {
			t.Errorf("%s: sql = %s\nwant %s", tt.name, sql, tt.sql)
		}
		if len(args) != 0 || len(tt.args) != 0 {
			if !reflect.DeepEqual(args, tt.args) {
				t.Errorf("%s: args = %#v, want %#v", tt.name, args, tt.args)
			}
		}
		if strings.Count(sql, "?") != len(args) {
			t.Errorf("%s: %d placeholders for %d args", tt.name, strings.Count(sql, "?"), len(args))
		}
	}
}

// 値は SQL の文字列に埋め込まず、引数で渡す（SQL インジェクションを防ぐ）
func TestIssueFilterSQL_ValuesArePlaceholders(t *testing.T) {
	evil := "x') OR 1=1 -- "
	for _, f := range []domain.IssueFilter{
		cond(domain.FieldProject, domain.CondIn, evil),
		cond(domain.FieldStatus, domain.CondNotIn, evil),
		cond(domain.FieldLabels, domain.CondIn, evil),
		cond(domain.FieldText, domain.CondContains, evil),
		cond(domain.FieldDue, domain.CondLt, evil),
	} {
		sql, args, err := issueFilterSQL(f)
		if err != nil {
			t.Fatalf("%s: error = %v", f.Condition.Field, err)
		}
		if strings.Contains(sql, "1=1") || len(args) == 0 {
			t.Errorf("%s: value is embedded in SQL: %s %v", f.Condition.Field, sql, args)
		}
	}
}

func TestIssueFilterSQL_Errors(t *testing.T) {
	for name, f := range map[string]domain.IssueFilter{
		"項目がない":         {Op: domain.FilterCondition, Condition: &domain.IssueCondition{Field: "summary", Op: domain.CondIn, Values: []any{"x"}}},
		"条件がない":         {Op: domain.FilterCondition},
		"種類がない":         {Op: "xor"},
		"AND に子がない":     {Op: domain.FilterAnd},
		"NOT の子が 2 つ":   {Op: domain.FilterNot, Children: []domain.IssueFilter{cond(domain.FieldType, domain.CondIn, "Bug"), cond(domain.FieldType, domain.CondIn, "Bug")}},
		"子の誤り":          {Op: domain.FilterOr, Children: []domain.IssueFilter{cond(domain.FieldType, domain.CondIn, "Bug"), cond("x", domain.CondIn, 1)}},
		"値のない項目に EMPTY": cond(domain.FieldProject, domain.CondEmpty),
		"text に IN":     cond(domain.FieldText, domain.CondIn, "x"),
		"text に値がない":    cond(domain.FieldText, domain.CondContains),
		"大小を比べられない項目":   cond(domain.FieldStatus, domain.CondGt, "Done"),
		"大小に値がない":       cond(domain.FieldCreated, domain.CondGt),
		"存在しない優先度":      cond(domain.FieldPriority, domain.CondGt, "Urgent"),
		"ラベルに大小":        cond(domain.FieldLabels, domain.CondGt, "x"),
		"演算子がない":        cond(domain.FieldType, "like", "x"),
	} {
		if _, _, err := issueFilterSQL(f); err == nil {
			t.Errorf("%s: error = nil", name)
		}
	}
}
