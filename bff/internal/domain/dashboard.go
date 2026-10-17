package domain

import (
	"errors"
	"regexp"
	"sort"
	"time"
)

// GadgetType はダッシュボードのガジェットの種類
type GadgetType string

const (
	GadgetAssignedToMe      GadgetType = "assignedToMe"      // 自分の担当課題（未完了）
	GadgetActivityStream    GadgetType = "activityStream"    // 最近のアクティビティ（全課題の変更履歴）
	GadgetFilterResults     GadgetType = "filterResults"     // 保存済みフィルターの結果
	GadgetStatusPie         GadgetType = "statusPie"         // ステータス別の円グラフ
	GadgetCreatedVsResolved GadgetType = "createdVsResolved" // 作成 vs 解決の推移
)

// gadgetUsesFilter はフィルター（FilterID）を選べるガジェットの種類
var gadgetUsesFilter = map[GadgetType]bool{
	GadgetFilterResults:     true,
	GadgetStatusPie:         true,
	GadgetCreatedVsResolved: true,
}

// GadgetUsesFilter はフィルターを選べるガジェットの種類かを返す
func GadgetUsesFilter(t GadgetType) bool {
	return gadgetUsesFilter[t]
}

// DashboardGadget はダッシュボードに置いたガジェット 1 つ。ダッシュボードの並びが表示順
type DashboardGadget struct {
	// ID はダッシュボードの中でガジェットを見分ける文字列（画面で採番する）
	ID   string
	Type GadgetType
	// FilterID は表示する保存済みフィルター（SavedFilter.ID）。0 は未選択で、
	// 保存済みフィルターの結果では「フィルターを選ぶ」、円グラフ・作成 vs 解決ではすべての課題になる
	FilterID int
}

// DashboardGadgetMax はダッシュボードに置けるガジェットの数の上限
const DashboardGadgetMax = 20

var gadgetIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,40}$`)

var (
	ErrTooManyGadgets      = errors.New("ダッシュボードに置けるガジェットは 20 個までです")
	ErrInvalidGadgetID     = errors.New("ガジェットの ID は英数字・-・_ の 1〜40 文字で、重複しないようにしてください")
	ErrInvalidGadgetType   = errors.New("ガジェットの種類は assignedToMe・activityStream・filterResults・statusPie・createdVsResolved のいずれかにしてください")
	ErrInvalidGadgetFilter = errors.New("ガジェットのフィルターの ID が正しくありません")
)

// ValidateDashboard はガジェットの並びが保存できるかを確かめ、フィルターを選べない種類の FilterID を 0 にしたものを返す
func ValidateDashboard(gadgets []DashboardGadget) ([]DashboardGadget, error) {
	if len(gadgets) > DashboardGadgetMax {
		return nil, ErrTooManyGadgets
	}
	seen := map[string]bool{}
	result := make([]DashboardGadget, 0, len(gadgets))
	for _, g := range gadgets {
		if !gadgetIDPattern.MatchString(g.ID) || seen[g.ID] {
			return nil, ErrInvalidGadgetID
		}
		seen[g.ID] = true
		switch g.Type {
		case GadgetAssignedToMe, GadgetActivityStream, GadgetFilterResults, GadgetStatusPie, GadgetCreatedVsResolved:
		default:
			return nil, ErrInvalidGadgetType
		}
		if g.FilterID < 0 {
			return nil, ErrInvalidGadgetFilter
		}
		if !GadgetUsesFilter(g.Type) {
			g.FilterID = 0
		}
		result = append(result, g)
	}
	return result, nil
}

// DefaultDashboard はまだダッシュボードを保存していないユーザーのガジェットの並び
func DefaultDashboard() []DashboardGadget {
	return []DashboardGadget{
		{ID: "assigned", Type: GadgetAssignedToMe},
		{ID: "activity", Type: GadgetActivityStream},
		{ID: "status", Type: GadgetStatusPie},
		{ID: "trend", Type: GadgetCreatedVsResolved},
	}
}

// ActivityItem は最近のアクティビティの 1 件。課題の変更履歴に、課題のキーと要約を付けたもの
type ActivityItem struct {
	HistoryEntry
	Key   string
	Title string
}

// StatusCount はステータスごとの課題数（円グラフ用）
type StatusCount struct {
	Name     string
	Category StatusCategory
	Count    int
}

// CountByStatus は課題をステータスの名前とカテゴリごとに数える。並びはカテゴリ（To Do・進行中・完了）の順、同じカテゴリは多い順・名前順
func CountByStatus(todos []Todo) []StatusCount {
	type key struct {
		name     string
		category StatusCategory
	}
	counts := map[key]int{}
	for _, todo := range todos {
		counts[key{todo.StatusName, todo.StatusCategory}]++
	}
	result := make([]StatusCount, 0, len(counts))
	for k, n := range counts {
		result = append(result, StatusCount{Name: k.name, Category: k.category, Count: n})
	}
	categoryOrder := map[StatusCategory]int{StatusCategoryToDo: 0, StatusCategoryInProgress: 1, StatusCategoryDone: 2}
	sort.Slice(result, func(i, j int) bool {
		a, b := result[i], result[j]
		if categoryOrder[a.Category] != categoryOrder[b.Category] {
			return categoryOrder[a.Category] < categoryOrder[b.Category]
		}
		if a.Count != b.Count {
			return a.Count > b.Count
		}
		return a.Name < b.Name
	})
	return result
}

// CreatedResolvedDay は 1 日分の作成数と解決数
type CreatedResolvedDay struct {
	Date     string // YYYY-MM-DD（集計したタイムゾーン）
	Created  int
	Resolved int
}

// CreatedVsResolved は期間中の日ごとの作成数と解決数。Created・Resolved は期間の合計
type CreatedVsResolved struct {
	Days     []CreatedResolvedDay
	Created  int
	Resolved int
}

// CreatedVsResolvedDaysMax は作成 vs 解決で集計できる日数の上限
const CreatedVsResolvedDaysMax = 365

var ErrInvalidTrendDays = errors.New("日数は 1〜365 の整数で指定してください")

// ComputeCreatedVsResolved は todos のうち、今日までの days 日間に作成・解決された課題を日ごとに数える。
// 作成は課題の作成日時、解決は今完了（Done カテゴリ）の課題が最後に完了になった日時（Jira の解決日と同じく、再開すると消える）。
// 完了になった日時は history（statusCategory・done・status の変更履歴）から求め、記録がない課題は解決数に入れない
func ComputeCreatedVsResolved(todos []Todo, history []HistoryEntry, days int, now time.Time, loc *time.Location) (CreatedVsResolved, error) {
	if days < 1 || days > CreatedVsResolvedDaysMax {
		return CreatedVsResolved{}, ErrInvalidTrendDays
	}
	today := now.In(loc)
	start := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, loc).AddDate(0, 0, -(days - 1))
	result := CreatedVsResolved{Days: make([]CreatedResolvedDay, days)}
	index := map[string]int{}
	for i := range result.Days {
		date := start.AddDate(0, 0, i).Format("2006-01-02")
		result.Days[i].Date = date
		index[date] = i
	}

	resolvedAt := map[int]time.Time{}
	for _, entry := range history {
		if !isResolution(entry) {
			continue
		}
		if at, ok := resolvedAt[entry.TodoID]; !ok || !entry.CreatedAt.Before(at) {
			resolvedAt[entry.TodoID] = entry.CreatedAt
		}
	}

	for _, todo := range todos {
		if i, ok := index[todo.CreatedAt.In(loc).Format("2006-01-02")]; ok {
			result.Days[i].Created++
			result.Created++
		}
		if todo.StatusCategory != StatusCategoryDone {
			continue
		}
		at, ok := resolvedAt[todo.ID]
		if !ok {
			continue
		}
		if i, ok := index[at.In(loc).Format("2006-01-02")]; ok {
			result.Days[i].Resolved++
			result.Resolved++
		}
	}
	return result, nil
}

// isResolution は課題が完了になった変更履歴かを返す。ワークフローの導入前の記録（done・status の Closed）も含める
func isResolution(entry HistoryEntry) bool {
	if entry.NewValue == nil {
		return false
	}
	switch entry.Field {
	case HistoryFieldStatusCategory:
		return *entry.NewValue == string(StatusCategoryDone) && (entry.OldValue == nil || *entry.OldValue != string(StatusCategoryDone))
	case HistoryFieldDone:
		return *entry.NewValue == "true"
	case HistoryFieldStatus:
		return *entry.NewValue == string(StatusClosed)
	}
	return false
}
