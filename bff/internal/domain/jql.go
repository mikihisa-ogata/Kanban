package domain

// IssueFilter は課題の絞り込み条件の木（JQL を解釈して名前・関数を ID や日時に置き換えたもの）。
// Op が FilterAnd・FilterOr なら Children のすべて・いずれか、FilterNot なら Children[0] を満たさないもの、
// FilterCondition なら Condition を満たすものを表す
type IssueFilter struct {
	Op        string
	Children  []IssueFilter
	Condition *IssueCondition
}

const (
	FilterAnd       = "and"
	FilterOr        = "or"
	FilterNot       = "not"
	FilterCondition = "condition"
)

// IssueCondition は 1 つの項目の条件。Values の型は項目による（IssueFilterFields を参照）
type IssueCondition struct {
	Field  string
	Op     string
	Values []any
}

// 条件の演算子。In・NotIn は Values のいずれか（NotIn は値のない課題には合わない。Jira と同じ）、
// Contains は Values[0] の文字列を含む、Gt などは Values[0] との大小、Empty・NotEmpty は値の有無
const (
	CondIn       = "in"
	CondNotIn    = "not in"
	CondContains = "contains"
	CondGt       = ">"
	CondGe       = ">="
	CondLt       = "<"
	CondLe       = "<="
	CondEmpty    = "empty"
	CondNotEmpty = "not empty"
)

// 絞り込みに使える項目。Values の型:
// project はスペースのキー（string。TASK はスペース未割り当て）、type は IssueType、status はステータス名（string）、
// statusCategory は StatusCategory、priority は Priority、assignee・reporter はユーザー ID（int）、labels はラベル名（string）、
// epic・sprint・fixVersion は ID（int）、created・updated は日時（time.Time）、due は日付（"2006-01-02" の string）、text は文字列、
// key は課題キー（大文字の string。例: DEV-12）、viewedBy は課題を表示したユーザーの ID（int。JQL の issueHistory()）
const (
	FieldProject        = "project"
	FieldType           = "type"
	FieldStatus         = "status"
	FieldStatusCategory = "statusCategory"
	FieldPriority       = "priority"
	FieldAssignee       = "assignee"
	FieldReporter       = "reporter"
	FieldLabels         = "labels"
	FieldEpic           = "epic"
	FieldSprint         = "sprint"
	FieldFixVersion     = "fixVersion"
	FieldCreated        = "created"
	FieldUpdated        = "updated"
	FieldDue            = "due"
	FieldText           = "text"
	FieldKey            = "key"
	FieldViewedBy       = "viewedBy"
)

// IssueOrder は並べ替えの 1 つの列（Field は IssueSortFields のいずれか）
type IssueOrder struct {
	Field string
	Desc  bool
}
