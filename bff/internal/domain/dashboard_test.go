package domain

import (
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestValidateDashboard(t *testing.T) {
	got, err := ValidateDashboard([]DashboardGadget{
		{ID: "a", Type: GadgetFilterResults, FilterID: 3},
		{ID: "b-2_x", Type: GadgetAssignedToMe, FilterID: 5}, // フィルターを選べない種類は 0 にする
		{ID: "c", Type: GadgetStatusPie},
	})
	if err != nil {
		t.Fatal(err)
	}
	want := []DashboardGadget{{ID: "a", Type: GadgetFilterResults, FilterID: 3}, {ID: "b-2_x", Type: GadgetAssignedToMe}, {ID: "c", Type: GadgetStatusPie}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("ValidateDashboard() = %+v, want %+v", got, want)
	}
	if got, err := ValidateDashboard([]DashboardGadget{}); err != nil || len(got) != 0 {
		t.Errorf("ValidateDashboard(empty) = %+v, %v", got, err)
	}

	many := make([]DashboardGadget, DashboardGadgetMax+1)
	for i := range many {
		many[i] = DashboardGadget{ID: strings.Repeat("g", i+1), Type: GadgetActivityStream}
	}
	for _, tt := range []struct {
		name    string
		gadgets []DashboardGadget
		want    error
	}{
		{"多すぎる", many, ErrTooManyGadgets},
		{"ID が空", []DashboardGadget{{ID: "", Type: GadgetStatusPie}}, ErrInvalidGadgetID},
		{"ID に使えない文字", []DashboardGadget{{ID: "a b", Type: GadgetStatusPie}}, ErrInvalidGadgetID},
		{"ID が長すぎる", []DashboardGadget{{ID: strings.Repeat("a", 41), Type: GadgetStatusPie}}, ErrInvalidGadgetID},
		{"ID の重複", []DashboardGadget{{ID: "a", Type: GadgetStatusPie}, {ID: "a", Type: GadgetActivityStream}}, ErrInvalidGadgetID},
		{"種類の誤り", []DashboardGadget{{ID: "a", Type: "calendar"}}, ErrInvalidGadgetType},
		{"負のフィルター", []DashboardGadget{{ID: "a", Type: GadgetStatusPie, FilterID: -1}}, ErrInvalidGadgetFilter},
	} {
		if _, err := ValidateDashboard(tt.gadgets); !errors.Is(err, tt.want) {
			t.Errorf("%s: err = %v, want %v", tt.name, err, tt.want)
		}
	}
	if _, err := ValidateDashboard(many[:DashboardGadgetMax]); err != nil {
		t.Errorf("上限ちょうど: err = %v", err)
	}
	if _, err := ValidateDashboard(DefaultDashboard()); err != nil {
		t.Errorf("既定の並び: err = %v", err)
	}
}

func TestCountByStatus(t *testing.T) {
	todos := []Todo{
		{StatusName: "完了", StatusCategory: StatusCategoryDone},
		{StatusName: "To Do", StatusCategory: StatusCategoryToDo},
		{StatusName: "レビュー", StatusCategory: StatusCategoryInProgress},
		{StatusName: "進行中", StatusCategory: StatusCategoryInProgress},
		{StatusName: "進行中", StatusCategory: StatusCategoryInProgress},
		{StatusName: "To Do", StatusCategory: StatusCategoryToDo},
	}
	want := []StatusCount{
		{Name: "To Do", Category: StatusCategoryToDo, Count: 2},
		{Name: "進行中", Category: StatusCategoryInProgress, Count: 2},
		{Name: "レビュー", Category: StatusCategoryInProgress, Count: 1},
		{Name: "完了", Category: StatusCategoryDone, Count: 1},
	}
	if got := CountByStatus(todos); !reflect.DeepEqual(got, want) {
		t.Errorf("CountByStatus() = %+v, want %+v", got, want)
	}
	if got := CountByStatus(nil); got == nil || len(got) != 0 {
		t.Errorf("CountByStatus(nil) = %#v, want empty", got)
	}
}

func TestComputeCreatedVsResolved(t *testing.T) {
	jst := time.FixedZone("JST", 9*60*60)
	now := time.Date(2026, 10, 17, 10, 0, 0, 0, jst)
	// UTC の 10/14 20:00 は JST の 10/15 05:00
	at := func(day, hour int) time.Time { return time.Date(2026, 10, day, hour, 0, 0, 0, time.UTC) }
	str := func(s string) *string { return &s }
	todos := []Todo{
		{ID: 1, CreatedAt: at(14, 20), StatusCategory: StatusCategoryDone},      // 10/15 作成、10/16 解決（最後の完了）
		{ID: 2, CreatedAt: at(1, 0), StatusCategory: StatusCategoryDone},        // 期間外に作成、10/17 解決
		{ID: 3, CreatedAt: at(16, 0), StatusCategory: StatusCategoryInProgress}, // 10/16 作成。完了したが再開したので解決に入れない
		{ID: 4, CreatedAt: at(15, 0), StatusCategory: StatusCategoryDone},       // 完了の記録がないので解決に入れない
		{ID: 5, CreatedAt: at(1, 0), StatusCategory: StatusCategoryDone},        // ワークフロー導入前の done の記録で、期間外に解決
	}
	history := []HistoryEntry{
		{TodoID: 1, Field: HistoryFieldStatusCategory, OldValue: str("ToDo"), NewValue: str("Done"), CreatedAt: at(15, 1)},
		{TodoID: 1, Field: HistoryFieldStatusCategory, OldValue: str("Done"), NewValue: str("ToDo"), CreatedAt: at(15, 2)},
		{TodoID: 1, Field: HistoryFieldStatusCategory, OldValue: str("ToDo"), NewValue: str("Done"), CreatedAt: at(16, 1)},
		// 完了のまま別の完了ステータスへ移っても、解決日は変わらない
		{TodoID: 1, Field: HistoryFieldStatusCategory, OldValue: str("Done"), NewValue: str("Done"), CreatedAt: at(16, 5)},
		{TodoID: 2, Field: HistoryFieldStatus, OldValue: str("Open"), NewValue: str("Closed"), CreatedAt: at(17, 0)},
		{TodoID: 3, Field: HistoryFieldStatusCategory, OldValue: str("InProgress"), NewValue: str("Done"), CreatedAt: at(16, 3)},
		{TodoID: 3, Field: HistoryFieldStatusCategory, OldValue: str("Done"), NewValue: str("InProgress"), CreatedAt: at(16, 4)},
		{TodoID: 5, Field: HistoryFieldDone, OldValue: str("false"), NewValue: str("true"), CreatedAt: at(2, 0)},
		{TodoID: 4, Field: HistoryFieldTitle, NewValue: str("Done"), CreatedAt: at(16, 0)},
	}
	got, err := ComputeCreatedVsResolved(todos, history, 3, now, jst)
	if err != nil {
		t.Fatal(err)
	}
	want := CreatedVsResolved{
		Days: []CreatedResolvedDay{
			{Date: "2026-10-15", Created: 2, Resolved: 0},
			{Date: "2026-10-16", Created: 1, Resolved: 1},
			{Date: "2026-10-17", Created: 0, Resolved: 1},
		},
		Created:  3,
		Resolved: 2,
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("ComputeCreatedVsResolved() = %+v, want %+v", got, want)
	}

	for _, days := range []int{0, -1, CreatedVsResolvedDaysMax + 1} {
		if _, err := ComputeCreatedVsResolved(nil, nil, days, now, jst); !errors.Is(err, ErrInvalidTrendDays) {
			t.Errorf("days %d: err = %v", days, err)
		}
	}
	if got, err := ComputeCreatedVsResolved(nil, nil, CreatedVsResolvedDaysMax, now, jst); err != nil || len(got.Days) != CreatedVsResolvedDaysMax {
		t.Errorf("days max: len = %d, err = %v", len(got.Days), err)
	}
}
