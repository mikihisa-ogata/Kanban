package repository

import (
	"fmt"
	"strings"

	"todo-api/internal/domain"
)

// JQL の絞り込み（domain.IssueFilter）を WHERE 句の式にする。値はすべてプレースホルダで渡し、SQL の文字列には埋め込まない。
// 式は課題一覧の検索と同じく todos t・spaces s・statuses st だけを参照する。
// NOT が値のない課題（担当者なしなど）でも補集合になるよう、どの条件も NULL にならない式（真か偽）にする

// issueFilterColumns は値が 1 つの項目の列。0 や NULL は「値がない」を表す
var issueFilterColumns = map[string]string{
	domain.FieldProject:        "COALESCE(s.space_key, '" + domain.UnassignedKeyPrefix + "')",
	domain.FieldType:           "t.issue_type",
	domain.FieldStatus:         "COALESCE(st.name, '')",
	domain.FieldStatusCategory: "COALESCE(st.category, '')",
	domain.FieldPriority:       "t.priority",
	domain.FieldAssignee:       "t.assignee_id",
	domain.FieldReporter:       "t.reporter_id",
	domain.FieldEpic:           "t.epic_id",
	domain.FieldSprint:         "t.sprint_id",
	domain.FieldCreated:        "t.created_at",
	domain.FieldUpdated:        "t.updated_at",
	domain.FieldDue:            "t.deadline",
	domain.FieldKey:            "CONCAT(COALESCE(s.space_key, '" + domain.UnassignedKeyPrefix + "'), '-', t.number)",
}

// issueFilterIDFields は 0 が「値がない」を表す項目
var issueFilterIDFields = map[string]bool{
	domain.FieldAssignee: true,
	domain.FieldReporter: true,
	domain.FieldEpic:     true,
	domain.FieldSprint:   true,
}

// issueFilterSets は 1 つの課題に複数の値が付く項目の、課題と値の組を返す副問い合わせ
var issueFilterSets = map[string]struct{ query, value string }{
	domain.FieldLabels:     {`SELECT tl.todo_id FROM todo_labels tl JOIN labels l ON l.id = tl.label_id`, "LOWER(l.name)"},
	domain.FieldFixVersion: {`SELECT tv.todo_id FROM todo_versions tv`, "tv.version_id"},
	domain.FieldViewedBy:   {`SELECT iv.todo_id FROM issue_views iv`, "iv.user_id"},
}

// priorityRanks は優先度の高い順（FIELD の位置）。大小の比較は高いほど大きいとする（Jira と同じ）
var priorityRanks = []domain.Priority{domain.PriorityHighest, domain.PriorityHigh, domain.PriorityMedium, domain.PriorityLow, domain.PriorityLowest}

const priorityRankSQL = "FIELD(t.priority, 'Highest', 'High', 'Medium', 'Low', 'Lowest')"

// issueFilterSQL は絞り込みを式と引数にする。条件の誤り（項目と演算子・値の組み合わせが合わない）はエラーを返す
func issueFilterSQL(f domain.IssueFilter) (string, []any, error) {
	switch f.Op {
	case domain.FilterAnd, domain.FilterOr:
		if len(f.Children) == 0 {
			return "", nil, fmt.Errorf("絞り込みの %s に条件がありません", f.Op)
		}
		parts := []string{}
		args := []any{}
		for _, child := range f.Children {
			expr, childArgs, err := issueFilterSQL(child)
			if err != nil {
				return "", nil, err
			}
			parts = append(parts, expr)
			args = append(args, childArgs...)
		}
		return "(" + strings.Join(parts, " "+strings.ToUpper(f.Op)+" ") + ")", args, nil
	case domain.FilterNot:
		if len(f.Children) != 1 {
			return "", nil, fmt.Errorf("絞り込みの not には条件が 1 つ必要です")
		}
		expr, args, err := issueFilterSQL(f.Children[0])
		if err != nil {
			return "", nil, err
		}
		return "(NOT " + expr + ")", args, nil
	case domain.FilterCondition:
		if f.Condition == nil {
			return "", nil, fmt.Errorf("絞り込みの条件がありません")
		}
		return issueConditionSQL(*f.Condition)
	}
	return "", nil, fmt.Errorf("絞り込みの種類 %q はありません", f.Op)
}

func issueConditionSQL(c domain.IssueCondition) (string, []any, error) {
	values := make([]any, len(c.Values))
	for i, v := range c.Values {
		values[i] = sqlValue(v)
	}
	needValue := func() error {
		if len(values) == 0 {
			return fmt.Errorf("%s の %s には値が必要です", c.Field, c.Op)
		}
		return nil
	}

	if c.Field == domain.FieldText {
		if c.Op != domain.CondContains {
			return "", nil, fmt.Errorf("text には contains だけを使えます")
		}
		if err := needValue(); err != nil {
			return "", nil, err
		}
		text, _ := values[0].(string)
		like := "%" + escapeLike(strings.TrimSpace(text)) + "%"
		return `(t.title LIKE ? OR t.description LIKE ? OR CONCAT(COALESCE(s.space_key, '` + domain.UnassignedKeyPrefix + `'), '-', t.number) = ?)`,
			[]any{like, like, strings.ToUpper(strings.TrimSpace(text))}, nil
	}

	if set, ok := issueFilterSets[c.Field]; ok {
		if c.Field == domain.FieldLabels {
			for i, v := range values {
				if s, ok := v.(string); ok {
					values[i] = strings.ToLower(s)
				}
			}
		}
		has := `t.id IN (` + set.query + `)`
		switch c.Op {
		case domain.CondEmpty:
			return `t.id NOT IN (` + set.query + `)`, nil, nil
		case domain.CondNotEmpty:
			return has, nil, nil
		case domain.CondIn:
			if len(values) == 0 {
				return "FALSE", nil, nil
			}
			return `t.id IN (` + set.query + ` WHERE ` + set.value + ` IN (` + placeholders(len(values)) + `))`, values, nil
		case domain.CondNotIn:
			if len(values) == 0 {
				return has, nil, nil
			}
			return `(` + has + ` AND t.id NOT IN (` + set.query + ` WHERE ` + set.value + ` IN (` + placeholders(len(values)) + `)))`, values, nil
		}
		return "", nil, fmt.Errorf("%s には %s を使えません", c.Field, c.Op)
	}

	column, ok := issueFilterColumns[c.Field]
	if !ok {
		return "", nil, fmt.Errorf("絞り込みの項目 %q はありません", c.Field)
	}
	// hasValue は値があるときに真になる式（値のない課題がない項目では空）
	hasValue := ""
	switch {
	case issueFilterIDFields[c.Field]:
		hasValue = column + " <> 0"
	case c.Field == domain.FieldDue:
		hasValue = column + " IS NOT NULL"
	}
	withValue := func(expr string) string {
		if hasValue == "" {
			return expr
		}
		return "(" + hasValue + " AND " + expr + ")"
	}

	switch c.Op {
	case domain.CondEmpty, domain.CondNotEmpty:
		if hasValue == "" {
			return "", nil, fmt.Errorf("%s には empty を使えません", c.Field)
		}
		if c.Op == domain.CondEmpty {
			return "(NOT " + hasValue + ")", nil, nil
		}
		return hasValue, nil, nil
	case domain.CondIn:
		if len(values) == 0 {
			return "FALSE", nil, nil
		}
		return withValue(column + ` IN (` + placeholders(len(values)) + `)`), values, nil
	case domain.CondNotIn:
		if len(values) == 0 {
			return withValue("TRUE"), nil, nil
		}
		return withValue(column + ` NOT IN (` + placeholders(len(values)) + `)`), values, nil
	case domain.CondGt, domain.CondGe, domain.CondLt, domain.CondLe:
		if err := needValue(); err != nil {
			return "", nil, err
		}
		switch c.Field {
		case domain.FieldPriority:
			// 高いほど FIELD の位置が小さいので、大小を逆にして比べる
			rank := 0
			for i, p := range priorityRanks {
				if values[0] == string(p) {
					rank = i + 1
				}
			}
			if rank == 0 {
				return "", nil, fmt.Errorf("優先度 %v はありません", values[0])
			}
			reversed := map[string]string{domain.CondGt: "<", domain.CondGe: "<=", domain.CondLt: ">", domain.CondLe: ">="}[c.Op]
			return "(" + priorityRankSQL + " > 0 AND " + priorityRankSQL + " " + reversed + " ?)", []any{rank}, nil
		case domain.FieldCreated, domain.FieldUpdated, domain.FieldDue:
			return withValue(column + " " + c.Op + " ?"), values[:1], nil
		}
		return "", nil, fmt.Errorf("%s には %s を使えません", c.Field, c.Op)
	}
	return "", nil, fmt.Errorf("%s には %s を使えません", c.Field, c.Op)
}

// sqlValue は domain の文字列型（課題タイプ・優先度など）を string にする
func sqlValue(v any) any {
	switch x := v.(type) {
	case domain.IssueType:
		return string(x)
	case domain.Priority:
		return string(x)
	case domain.StatusCategory:
		return string(x)
	}
	return v
}
