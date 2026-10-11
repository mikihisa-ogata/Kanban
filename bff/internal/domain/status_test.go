package domain

import "testing"

func TestLegacyCategoryAndStatus(t *testing.T) {
	for legacy, want := range map[Status]StatusCategory{
		StatusOpen: StatusCategoryToDo, StatusInProgress: StatusCategoryInProgress, StatusWaiting: StatusCategoryInProgress, StatusClosed: StatusCategoryDone,
	} {
		if got, ok := LegacyCategory(legacy); !ok || got != want {
			t.Errorf("LegacyCategory(%s) = %s, %v", legacy, got, ok)
		}
	}
	if _, ok := LegacyCategory("Done"); ok {
		t.Error("unknown status should not have a category")
	}
	// 既定のステータスは導入前の値のまま、追加したステータスはカテゴリから決まる
	for _, s := range DefaultWorkflow() {
		if s.LegacyStatus() != s.LegacyKey {
			t.Errorf("%s LegacyStatus = %s", s.Name, s.LegacyStatus())
		}
	}
	for category, want := range map[StatusCategory]Status{
		StatusCategoryToDo: StatusOpen, StatusCategoryInProgress: StatusInProgress, StatusCategoryDone: StatusClosed,
	} {
		if got := (WorkflowStatus{Name: "レビュー", Category: category}).LegacyStatus(); got != want {
			t.Errorf("custom %s LegacyStatus = %s, want %s", category, got, want)
		}
	}
}

func TestMatchStatus(t *testing.T) {
	target := []WorkflowStatus{
		{ID: 11, Name: "未着手", Category: StatusCategoryToDo},
		{ID: 12, Name: "作業中", Category: StatusCategoryInProgress},
		{ID: 13, Name: "レビュー", Category: StatusCategoryInProgress},
		{ID: 14, Name: "完了", Category: StatusCategoryDone},
	}
	tests := []struct {
		from WorkflowStatus
		want int
	}{
		{WorkflowStatus{Name: "レビュー", Category: StatusCategoryInProgress}, 13}, // 同じ名前
		{WorkflowStatus{Name: "待ち状況", Category: StatusCategoryInProgress}, 12}, // 同じカテゴリの先頭
		{WorkflowStatus{Name: "クローズ", Category: StatusCategoryDone}, 14},
		{WorkflowStatus{Name: "オープン", Category: StatusCategoryToDo}, 11},
	}
	for _, tt := range tests {
		if got, ok := MatchStatus(target, tt.from); !ok || got.ID != tt.want {
			t.Errorf("MatchStatus(%s) = %d, want %d", tt.from.Name, got.ID, tt.want)
		}
	}
	// カテゴリがなければ先頭
	if got, _ := FirstInCategory(target[:1], StatusCategoryDone); got.ID != 11 {
		t.Errorf("FirstInCategory fallback = %d", got.ID)
	}
	if _, ok := MatchStatus(nil, target[0]); ok {
		t.Error("empty target should not match")
	}
}
