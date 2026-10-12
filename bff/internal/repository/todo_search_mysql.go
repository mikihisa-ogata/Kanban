package repository

import (
	"strings"

	"todo-api/internal/domain"
)

// issueSortColumns は並べ替えの列ごとの ORDER BY の式。値のない行（未割り当て・期限なし）は昇順・降順とも末尾にする
var issueSortColumns = map[string][]string{
	domain.IssueSortKey:      {"COALESCE(s.space_key, '" + domain.UnassignedKeyPrefix + "')", "t.number"},
	domain.IssueSortType:     {"t.issue_type"},
	domain.IssueSortSummary:  {"t.title"},
	domain.IssueSortStatus:   {"st.name"},
	domain.IssueSortAssignee: {"ua.display_name"},
	domain.IssueSortReporter: {"ur.display_name"},
	// 優先度は高い順を昇順とする（Jira と同じ）
	domain.IssueSortPriority: {"FIELD(t.priority, 'Highest', 'High', 'Medium', 'Low', 'Lowest')"},
	domain.IssueSortCreated:  {"t.created_at"},
	domain.IssueSortUpdated:  {"t.updated_at"},
	domain.IssueSortDue:      {"t.deadline"},
}

// nullableSortColumns は値がない行を末尾にする列
var nullableSortColumns = map[string]bool{
	domain.IssueSortAssignee: true,
	domain.IssueSortReporter: true,
	domain.IssueSortDue:      true,
}

func (r *todoMySQLRepository) Search(q domain.IssueSearch) (domain.IssueSearchResult, error) {
	where, args := issueSearchWhere(q)

	result := domain.IssueSearchResult{Issues: []domain.Todo{}, Page: q.Page, PerPage: q.PerPage}
	if err := r.db.QueryRow(`SELECT COUNT(*) FROM todos t LEFT JOIN spaces s ON s.id = t.space_id LEFT JOIN statuses st ON st.id = t.status_id`+where, args...).
		Scan(&result.Total); err != nil {
		return domain.IssueSearchResult{}, err
	}

	direction := " ASC"
	if q.Desc {
		direction = " DESC"
	}
	order := []string{}
	if nullableSortColumns[q.Sort] {
		order = append(order, issueSortColumns[q.Sort][0]+" IS NULL")
	}
	for _, column := range issueSortColumns[q.Sort] {
		order = append(order, column+direction)
	}
	order = append(order, "t.id"+direction)

	rows, err := r.db.Query(selectTodos+where+` ORDER BY `+strings.Join(order, ", ")+` LIMIT ? OFFSET ?`,
		append(args, q.PerPage, (q.Page-1)*q.PerPage)...)
	if err != nil {
		return domain.IssueSearchResult{}, err
	}
	defer rows.Close()
	for rows.Next() {
		todo, err := scanTodo(rows)
		if err != nil {
			return domain.IssueSearchResult{}, err
		}
		result.Issues = append(result.Issues, todo)
	}
	if err := rows.Err(); err != nil {
		return domain.IssueSearchResult{}, err
	}
	if len(result.Issues) == 0 {
		return result, nil
	}

	// ラベル・修正バージョン・コンポーネントは、このページの課題の分だけ読む
	ids := make([]any, len(result.Issues))
	for i, todo := range result.Issues {
		ids[i] = todo.ID
	}
	in := placeholders(len(ids))
	labels, err := r.labelsByTodo(`SELECT tl.todo_id, l.name FROM todo_labels tl JOIN labels l ON l.id = tl.label_id WHERE tl.todo_id IN (`+in+`)`, ids...)
	if err != nil {
		return domain.IssueSearchResult{}, err
	}
	versions, err := r.idsByTodo(`SELECT todo_id, version_id FROM todo_versions WHERE todo_id IN (`+in+`)`, "version_id", ids...)
	if err != nil {
		return domain.IssueSearchResult{}, err
	}
	components, err := r.idsByTodo(`SELECT todo_id, component_id FROM todo_components WHERE todo_id IN (`+in+`)`, "component_id", ids...)
	if err != nil {
		return domain.IssueSearchResult{}, err
	}
	for i := range result.Issues {
		id := result.Issues[i].ID
		if names, ok := labels[id]; ok {
			result.Issues[i].Labels = names
		}
		if v, ok := versions[id]; ok {
			result.Issues[i].FixVersionIDs = v
		}
		if c, ok := components[id]; ok {
			result.Issues[i].ComponentIDs = c
		}
	}
	return result, nil
}

// issueSearchWhere は検索条件を WHERE 句（空の条件なら空文字）と引数にする
func issueSearchWhere(q domain.IssueSearch) (string, []any) {
	conditions := []string{}
	args := []any{}
	in := func(column string, values []any) {
		conditions = append(conditions, column+` IN (`+placeholders(len(values))+`)`)
		args = append(args, values...)
	}
	if len(q.SpaceIDs) > 0 {
		in("t.space_id", toAny(q.SpaceIDs))
	}
	if len(q.Types) > 0 {
		values := make([]any, len(q.Types))
		for i, t := range q.Types {
			values[i] = string(t)
		}
		in("t.issue_type", values)
	}
	if len(q.StatusNames) > 0 {
		values := make([]any, len(q.StatusNames))
		for i, name := range q.StatusNames {
			values[i] = name
		}
		in("st.name", values)
	}
	if len(q.AssigneeIDs) > 0 {
		in("t.assignee_id", toAny(q.AssigneeIDs))
	}
	if len(q.Labels) > 0 {
		values := make([]any, len(q.Labels))
		for i, name := range q.Labels {
			values[i] = strings.ToLower(name)
		}
		conditions = append(conditions, `t.id IN (SELECT tl.todo_id FROM todo_labels tl JOIN labels l ON l.id = tl.label_id WHERE LOWER(l.name) IN (`+placeholders(len(values))+`))`)
		args = append(args, values...)
	}
	if text := strings.TrimSpace(q.Text); text != "" {
		like := "%" + escapeLike(text) + "%"
		conditions = append(conditions, `(t.title LIKE ? OR t.description LIKE ? OR CONCAT(COALESCE(s.space_key, '`+domain.UnassignedKeyPrefix+`'), '-', t.number) = ?)`)
		args = append(args, like, like, strings.ToUpper(text))
	}
	if len(conditions) == 0 {
		return "", args
	}
	return ` WHERE ` + strings.Join(conditions, " AND "), args
}

// escapeLike は LIKE の特殊文字（\・%・_）をそのままの文字として扱うようにする
func escapeLike(s string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(s)
}

func placeholders(n int) string {
	return strings.TrimSuffix(strings.Repeat("?, ", n), ", ")
}

func toAny(ids []int) []any {
	values := make([]any, len(ids))
	for i, id := range ids {
		values[i] = id
	}
	return values
}
