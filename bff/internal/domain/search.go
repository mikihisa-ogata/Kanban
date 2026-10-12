package domain

import "errors"

// IssueSearch は課題一覧（Jira の課題ナビゲーター）の検索条件。条件の種類どうしは AND、同じ種類の値どうしは OR で絞り込む
type IssueSearch struct {
	// SpaceIDs はスペース。0 はスペース未割り当て。空なら全スペース
	SpaceIDs []int
	// Types は課題タイプ
	Types []IssueType
	// StatusNames はステータスの名前（スペースをまたいで同じ名前のステータスをまとめて選ぶ）
	StatusNames []string
	// AssigneeIDs は担当者。0 は未割り当て
	AssigneeIDs []int
	// Labels はラベルの名前（大文字・小文字は区別しない）
	Labels []string
	// Text は要約・説明・課題キーに含まれる文字列。空なら絞り込まない
	Text string
	// Sort は並べ替える列（IssueSortFields のいずれか）。Desc が true なら降順
	Sort string
	Desc bool
	// Page は 1 から始まるページ番号、PerPage は 1 ページの件数
	Page    int
	PerPage int
}

// IssueSearchResult は課題一覧の 1 ページ分の結果。Total は条件に合う課題の総数
type IssueSearchResult struct {
	Issues  []Todo
	Total   int
	Page    int
	PerPage int
}

// 並べ替えに使える列
const (
	IssueSortKey      = "key"
	IssueSortType     = "type"
	IssueSortSummary  = "summary"
	IssueSortStatus   = "status"
	IssueSortAssignee = "assignee"
	IssueSortReporter = "reporter"
	IssueSortPriority = "priority"
	IssueSortCreated  = "created"
	IssueSortUpdated  = "updated"
	IssueSortDue      = "due"
)

// IssueSortFields は並べ替えに使える列の一覧
var IssueSortFields = []string{IssueSortKey, IssueSortType, IssueSortSummary, IssueSortStatus, IssueSortAssignee, IssueSortReporter, IssueSortPriority, IssueSortCreated, IssueSortUpdated, IssueSortDue}

// DefaultIssuePerPage は 1 ページの既定の件数、MaxIssuePerPage はその上限
const (
	DefaultIssuePerPage = 50
	MaxIssuePerPage     = 100
)

var ErrInvalidIssueSort = errors.New("並べ替えは key・type・summary・status・assignee・reporter・priority・created・updated・due のいずれかにしてください")

var ErrInvalidIssueSearchType = errors.New("課題タイプは Task・Bug・Story・Subtask のいずれかにしてください")

var ErrInvalidIssuePage = errors.New("ページは 1 以上、1 ページの件数は 1〜100 にしてください")

// Normalize は並べ替え・ページングの既定値を入れて検証する。並べ替えの既定は Jira と同じく作成日の降順
func (q *IssueSearch) Normalize() error {
	if q.Sort == "" {
		q.Sort, q.Desc = IssueSortCreated, true
	}
	valid := false
	for _, f := range IssueSortFields {
		if q.Sort == f {
			valid = true
		}
	}
	if !valid {
		return ErrInvalidIssueSort
	}
	if q.Page == 0 {
		q.Page = 1
	}
	if q.PerPage == 0 {
		q.PerPage = DefaultIssuePerPage
	}
	if q.Page < 1 || q.PerPage < 1 || q.PerPage > MaxIssuePerPage {
		return ErrInvalidIssuePage
	}
	for _, t := range q.Types {
		if !ValidIssueType(t) {
			return ErrInvalidIssueSearchType
		}
	}
	return nil
}
