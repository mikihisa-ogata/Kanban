package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"todo-api/internal/domain"
	"todo-api/internal/service"
)

// setupSprintRouter はスペース・課題・スプリントの API を持つルーターを返す
func setupSprintRouter(t *testing.T) *gin.Engine {
	t.Helper()
	repos := newTestRepos(t)
	r := newTodoRouter(repos)
	sh := NewSpaceHandler(service.NewSpaceService(repos.space, repos.epic, repos.todo, repos.history, repos.status))
	r.POST("/spaces", sh.CreateSpace)
	r.DELETE("/spaces/:id", sh.DeleteSpace)
	h := NewSprintHandler(service.NewSprintService(repos.sprint, repos.todo, repos.space, repos.history))
	r.GET("/sprints", h.GetSprints)
	r.POST("/spaces/:id/sprints", h.CreateSprint)
	r.PUT("/sprints/:id", h.UpdateSprint)
	r.POST("/sprints/:id/start", h.StartSprint)
	r.POST("/sprints/:id/complete", h.CompleteSprint)
	r.DELETE("/sprints/:id", h.DeleteSprint)
	r.GET("/sprints/:id/issues", h.GetSprintIssues)
	r.GET("/sprints/:id/burndown", h.GetBurndown)
	r.GET("/spaces/:id/velocity", h.GetVelocity)
	r.PUT("/todos/:id/sprint", h.SetTodoSprint)
	return r
}

func createSprint(t *testing.T, r *gin.Engine, spaceID int, body string) domain.Sprint {
	t.Helper()
	w := doRequest(t, r, http.MethodPost, fmt.Sprintf("/spaces/%d/sprints", spaceID), body)
	if w.Code != http.StatusCreated {
		t.Fatalf("POST sprints status = %d, body = %s", w.Code, w.Body)
	}
	var res struct{ Sprint domain.Sprint }
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		t.Fatal(err)
	}
	return res.Sprint
}

func findSprint(t *testing.T, r *gin.Engine, id int) domain.Sprint {
	t.Helper()
	for _, s := range getJSON[[]domain.Sprint](t, r, "/sprints") {
		if s.ID == id {
			return s
		}
	}
	t.Fatalf("sprint %d not found", id)
	return domain.Sprint{}
}

// sprintOf は課題（ID）のスプリントを返す
func sprintOf(t *testing.T, r *gin.Engine, todoID int) int {
	t.Helper()
	for _, todo := range getTodos(t, r) {
		if todo.ID == todoID {
			return todo.SprintID
		}
	}
	t.Fatalf("todo %d not found", todoID)
	return 0
}

func setSprint(t *testing.T, r *gin.Engine, todoID, sprintID, want int) {
	t.Helper()
	mustStatus(t, r, http.MethodPut, fmt.Sprintf("/todos/%d/sprint", todoID), fmt.Sprintf(`{"sprintId":%d}`, sprintID), want)
}

func TestSprint_CreateStartComplete(t *testing.T) {
	r := setupSprintRouter(t)
	mustStatus(t, r, http.MethodPost, "/spaces", `{"Title":"開発","Key":"DEV"}`, http.StatusCreated)
	mustStatus(t, r, http.MethodPost, "/spaces", `{"Title":"運用","Key":"OPS"}`, http.StatusCreated)

	// 名前を省略すると「<キー> スプリント <番号>」。未割り当て・存在しないスペースには作れない
	s1 := createSprint(t, r, 1, `{}`)
	if s1.Name != "DEV スプリント 1" || s1.State != domain.SprintStateFuture || s1.SpaceID != 1 {
		t.Errorf("created = %+v", s1)
	}
	s2 := createSprint(t, r, 1, `{"name":"  次のスプリント "}`)
	if s2.Name != "次のスプリント" {
		t.Errorf("named sprint = %q", s2.Name)
	}
	mustStatus(t, r, http.MethodPost, "/spaces/0/sprints", `{}`, http.StatusBadRequest)
	mustStatus(t, r, http.MethodPost, "/spaces/99/sprints", `{}`, http.StatusNotFound)

	// 課題: A（3pt・サブタスク a1 あり）、B（5pt）、C、別スペースの X
	mustStatus(t, r, http.MethodPost, "/todos", `{"title":"A","spaceId":1,"storyPoints":3}`, http.StatusCreated)
	mustStatus(t, r, http.MethodPost, "/todos", `{"title":"B","spaceId":1,"storyPoints":5}`, http.StatusCreated)
	mustStatus(t, r, http.MethodPost, "/todos", `{"title":"C","spaceId":1}`, http.StatusCreated)
	mustStatus(t, r, http.MethodPost, "/todos", `{"title":"X","spaceId":2}`, http.StatusCreated)
	mustStatus(t, r, http.MethodPost, "/todos", `{"title":"a1","parentId":1}`, http.StatusCreated)
	if got := sprintOf(t, r, 1); got != 0 {
		t.Errorf("new todo sprint = %d, want 0 (backlog)", got)
	}

	// スプリントに入れると、サブタスクも一緒に入る。サブタスクだけ・別スペースのスプリントは 400、存在しないものは 404
	setSprint(t, r, 1, s1.ID, http.StatusOK)
	setSprint(t, r, 2, s1.ID, http.StatusOK)
	setSprint(t, r, 3, s1.ID, http.StatusOK)
	if got := sprintOf(t, r, 5); got != s1.ID {
		t.Errorf("subtask sprint = %d, want %d", got, s1.ID)
	}
	setSprint(t, r, 5, 0, http.StatusBadRequest)
	setSprint(t, r, 4, s1.ID, http.StatusBadRequest)
	setSprint(t, r, 99, s1.ID, http.StatusNotFound)
	setSprint(t, r, 1, 99, http.StatusNotFound)
	// スプリントの親課題に作ったサブタスクは、同じスプリントに入る
	mustStatus(t, r, http.MethodPost, "/todos", `{"title":"a2","parentId":1}`, http.StatusCreated)
	if got := sprintOf(t, r, 6); got != s1.ID {
		t.Errorf("new subtask sprint = %d, want %d", got, s1.ID)
	}
	// C はバックログに戻す
	setSprint(t, r, 3, 0, http.StatusOK)
	if got := sprintOf(t, r, 3); got != 0 {
		t.Errorf("C sprint = %d, want 0", got)
	}
	setSprint(t, r, 3, s1.ID, http.StatusOK)

	// 開始には期間が要る（形式の誤り・逆順も 400）。未開始でなければ 400、同じスペースで 2 つ目は 409
	start := fmt.Sprintf("/sprints/%d/start", s1.ID)
	mustStatus(t, r, http.MethodPost, start, `{"name":"DEV スプリント 1"}`, http.StatusBadRequest)
	mustStatus(t, r, http.MethodPost, start, `{"name":"DEV スプリント 1","startDate":"2026-10-12","endDate":"2026/10/26"}`, http.StatusBadRequest)
	mustStatus(t, r, http.MethodPost, start, `{"name":"DEV スプリント 1","startDate":"2026-10-12","endDate":"2026-10-11"}`, http.StatusBadRequest)
	mustStatus(t, r, http.MethodPost, start, `{"name":"","startDate":"2026-10-12","endDate":"2026-10-26"}`, http.StatusBadRequest)
	mustStatus(t, r, http.MethodPost, start, `{"name":"DEV スプリント 1","goal":"ログイン","startDate":"2026-10-12","endDate":"2026-10-26"}`, http.StatusOK)
	active := findSprint(t, r, s1.ID)
	if active.State != domain.SprintStateActive || active.Goal != "ログイン" || active.StartDate != "2026-10-12" || active.EndDate != "2026-10-26" {
		t.Errorf("started = %+v", active)
	}
	mustStatus(t, r, http.MethodPost, start, `{"name":"x","startDate":"2026-10-12","endDate":"2026-10-26"}`, http.StatusBadRequest)
	mustStatus(t, r, http.MethodPost, fmt.Sprintf("/sprints/%d/start", s2.ID), `{"name":"x","startDate":"2026-10-12","endDate":"2026-10-26"}`, http.StatusConflict)
	// 別のスペースなら開始できる
	ops := createSprint(t, r, 2, `{}`)
	mustStatus(t, r, http.MethodPost, fmt.Sprintf("/sprints/%d/start", ops.ID), `{"name":"OPS スプリント 1","startDate":"2026-10-12","endDate":"2026-10-19"}`, http.StatusOK)

	// 開始中のスプリントの編集（期間は外せない）。開始中は削除できない
	mustStatus(t, r, http.MethodPut, fmt.Sprintf("/sprints/%d", s1.ID), `{"name":"DEV 1","goal":"","startDate":"2026-10-12","endDate":"2026-10-30"}`, http.StatusOK)
	mustStatus(t, r, http.MethodPut, fmt.Sprintf("/sprints/%d", s1.ID), `{"name":"DEV 1"}`, http.StatusBadRequest)
	if got := findSprint(t, r, s1.ID); got.Name != "DEV 1" || got.EndDate != "2026-10-30" || got.Goal != "" {
		t.Errorf("updated = %+v", got)
	}
	mustStatus(t, r, http.MethodDelete, fmt.Sprintf("/sprints/%d", s1.ID), "", http.StatusBadRequest)

	// B を完了にする
	mustStatus(t, r, http.MethodPut, "/todos/2", `{"title":"B","status":"Closed","spaceId":1}`, http.StatusOK)

	// 完了: 移し先は同じスペースの未開始のスプリントかバックログ。未開始のスプリントは完了できない
	complete := fmt.Sprintf("/sprints/%d/complete", s1.ID)
	mustStatus(t, r, http.MethodPost, complete, fmt.Sprintf(`{"moveTo":%d}`, ops.ID), http.StatusBadRequest)
	mustStatus(t, r, http.MethodPost, complete, fmt.Sprintf(`{"moveTo":%d}`, s1.ID), http.StatusBadRequest)
	mustStatus(t, r, http.MethodPost, complete, `{"moveTo":99}`, http.StatusBadRequest)
	mustStatus(t, r, http.MethodPost, fmt.Sprintf("/sprints/%d/complete", s2.ID), `{}`, http.StatusBadRequest)
	mustStatus(t, r, http.MethodPost, complete, fmt.Sprintf(`{"moveTo":%d}`, s2.ID), http.StatusOK)

	closed := findSprint(t, r, s1.ID)
	if closed.State != domain.SprintStateClosed || closed.CompletedAt == nil {
		t.Errorf("closed = %+v", closed)
	}
	if closed.CompletedIssues != 1 || closed.IncompleteIssues != 2 || closed.CompletedPoints != 5 {
		t.Errorf("closed counts = %d/%d/%v, want 1/2/5", closed.CompletedIssues, closed.IncompleteIssues, closed.CompletedPoints)
	}
	// 未完了の A（とサブタスク）・C は次のスプリントへ、完了した B は完了したスプリントに残る
	for todoID, want := range map[int]int{1: s2.ID, 5: s2.ID, 6: s2.ID, 3: s2.ID, 2: s1.ID} {
		if got := sprintOf(t, r, todoID); got != want {
			t.Errorf("todo %d sprint = %d, want %d", todoID, got, want)
		}
	}
	// 記録した課題（サブタスクは含まない）
	issues := getJSON[[]domain.SprintIssue](t, r, fmt.Sprintf("/sprints/%d/issues", s1.ID))
	if len(issues) != 3 || issues[0].TodoID != 1 || issues[0].Completed || *issues[0].StoryPoints != 3 ||
		issues[1].TodoID != 2 || !issues[1].Completed || issues[2].TodoID != 3 || issues[2].StoryPoints != nil {
		t.Errorf("sprint issues = %+v", issues)
	}
	// 完了したスプリントは変更できず、課題も入れられない
	mustStatus(t, r, http.MethodPost, complete, `{}`, http.StatusBadRequest)
	mustStatus(t, r, http.MethodPut, fmt.Sprintf("/sprints/%d", s1.ID), `{"name":"x"}`, http.StatusBadRequest)
	setSprint(t, r, 3, s1.ID, http.StatusBadRequest)
	mustStatus(t, r, http.MethodGet, "/sprints/99/issues", "", http.StatusNotFound)

	// 次のスプリントを開始して、バックログへ戻して完了する
	mustStatus(t, r, http.MethodPost, fmt.Sprintf("/sprints/%d/start", s2.ID), `{"name":"次","startDate":"2026-10-27","endDate":"2026-11-09"}`, http.StatusOK)
	mustStatus(t, r, http.MethodPost, fmt.Sprintf("/sprints/%d/complete", s2.ID), `{"moveTo":0}`, http.StatusOK)
	for _, todoID := range []int{1, 3, 5} {
		if got := sprintOf(t, r, todoID); got != 0 {
			t.Errorf("todo %d sprint = %d, want 0 (backlog)", todoID, got)
		}
	}
}

func TestSprint_DeleteAndSpaceChange(t *testing.T) {
	r := setupSprintRouter(t)
	mustStatus(t, r, http.MethodPost, "/spaces", `{"Title":"開発","Key":"DEV"}`, http.StatusCreated)
	mustStatus(t, r, http.MethodPost, "/spaces", `{"Title":"運用","Key":"OPS"}`, http.StatusCreated)
	sprint := createSprint(t, r, 1, `{}`)
	mustStatus(t, r, http.MethodPost, "/todos", `{"title":"A","spaceId":1}`, http.StatusCreated)
	mustStatus(t, r, http.MethodPost, "/todos", `{"title":"B","spaceId":1}`, http.StatusCreated)
	mustStatus(t, r, http.MethodPost, "/todos", `{"title":"b1","parentId":2}`, http.StatusCreated)
	setSprint(t, r, 1, sprint.ID, http.StatusOK)
	setSprint(t, r, 2, sprint.ID, http.StatusOK)

	// スペースを移すとバックログに戻る（サブタスクも）。スペースが変わらない更新ではスプリントは変わらない
	mustStatus(t, r, http.MethodPut, "/todos/1", `{"title":"A2","spaceId":1}`, http.StatusOK)
	if got := sprintOf(t, r, 1); got != sprint.ID {
		t.Errorf("sprint after update = %d, want %d", got, sprint.ID)
	}
	mustStatus(t, r, http.MethodPut, "/todos/2", `{"title":"B","spaceId":2}`, http.StatusOK)
	if got, sub := sprintOf(t, r, 2), sprintOf(t, r, 3); got != 0 || sub != 0 {
		t.Errorf("sprint after space change = %d (subtask %d), want 0", got, sub)
	}

	// 未開始のスプリントを削除すると、その課題はバックログに戻る
	mustStatus(t, r, http.MethodDelete, fmt.Sprintf("/sprints/%d", sprint.ID), "", http.StatusOK)
	if got := sprintOf(t, r, 1); got != 0 {
		t.Errorf("sprint after delete = %d, want 0", got)
	}
	mustStatus(t, r, http.MethodDelete, fmt.Sprintf("/sprints/%d", sprint.ID), "", http.StatusNotFound)

	// スペースを削除すると、そのスプリントも消え、課題はバックログに戻る
	other := createSprint(t, r, 1, `{}`)
	setSprint(t, r, 1, other.ID, http.StatusOK)
	mustStatus(t, r, http.MethodDelete, "/spaces/1", "", http.StatusOK)
	if sprints := getJSON[[]domain.Sprint](t, r, "/sprints"); len(sprints) != 0 {
		t.Errorf("sprints after space delete = %+v", sprints)
	}
	if got := sprintOf(t, r, 1); got != 0 {
		t.Errorf("sprint after space delete = %d, want 0", got)
	}
}

// スプリントの出し入れ（完了時の移動・削除を含む）は、スプリントの ID 付きで履歴に残り、バーンダウンはその履歴から再計算する
func TestSprint_HistoryAndBurndown(t *testing.T) {
	r := setupSprintRouter(t)
	mustStatus(t, r, http.MethodPost, "/spaces", `{"Title":"開発","Key":"DEV"}`, http.StatusCreated)
	s1 := createSprint(t, r, 1, `{}`)
	s2 := createSprint(t, r, 1, `{}`)
	mustStatus(t, r, http.MethodPost, "/todos", `{"title":"A","spaceId":1,"storyPoints":3}`, http.StatusCreated)
	mustStatus(t, r, http.MethodPost, "/todos", `{"title":"B","spaceId":1,"storyPoints":5}`, http.StatusCreated)
	mustStatus(t, r, http.MethodPost, "/todos", `{"title":"C","spaceId":1,"storyPoints":2}`, http.StatusCreated)
	mustStatus(t, r, http.MethodPost, "/todos", `{"title":"a1","parentId":1,"storyPoints":1}`, http.StatusCreated)
	setSprint(t, r, 1, s1.ID, http.StatusOK)
	setSprint(t, r, 2, s1.ID, http.StatusOK)

	// 親課題とサブタスクの両方に、スプリントの名前と ID が残る
	lastSprintEntry := func(todoID int) domain.HistoryEntry {
		t.Helper()
		var found domain.HistoryEntry
		for _, e := range getJSON[[]domain.HistoryEntry](t, r, fmt.Sprintf("/todos/%d/history", todoID)) {
			if e.Field == domain.HistoryFieldSprint {
				found = e
			}
		}
		if found.ID == 0 {
			t.Fatalf("todo %d has no sprint history", todoID)
		}
		return found
	}
	for _, todoID := range []int{1, 4} {
		e := lastSprintEntry(todoID)
		if e.OldValue != nil || e.NewValue == nil || *e.NewValue != s1.Name || e.OldRefID == nil || *e.OldRefID != 0 || e.NewRefID == nil || *e.NewRefID != s1.ID {
			t.Errorf("todo %d sprint history = %+v", todoID, e)
		}
	}

	// 未開始・存在しないスプリントは表示できない
	burndownPath := fmt.Sprintf("/sprints/%d/burndown?tz=Asia/Tokyo", s1.ID)
	mustStatus(t, r, http.MethodGet, burndownPath, "", http.StatusBadRequest)
	mustStatus(t, r, http.MethodGet, "/sprints/99/burndown", "", http.StatusNotFound)

	tokyo, err := time.LoadLocation("Asia/Tokyo")
	if err != nil {
		t.Fatal(err)
	}
	today := time.Now().In(tokyo)
	mustStatus(t, r, http.MethodPost, fmt.Sprintf("/sprints/%d/start", s1.ID),
		fmt.Sprintf(`{"name":"DEV 1","startDate":%q,"endDate":%q}`, today.Format("2006-01-02"), today.AddDate(0, 0, 4).Format("2006-01-02")), http.StatusOK)
	if started := findSprint(t, r, s1.ID); started.StartedAt == nil {
		t.Errorf("StartedAt is not recorded: %+v", started)
	}
	// 履歴の日時は秒単位なので、開始と同じ秒の変更にならないようにする
	time.Sleep(1100 * time.Millisecond)

	// 開始後: C を追加、B を完了、A を 3 → 8 pt
	setSprint(t, r, 3, s1.ID, http.StatusOK)
	mustStatus(t, r, http.MethodPut, "/todos/2", `{"title":"B","status":"Closed","spaceId":1,"storyPoints":5}`, http.StatusOK)
	mustStatus(t, r, http.MethodPut, "/todos/1", `{"title":"A","spaceId":1,"storyPoints":8}`, http.StatusOK)
	mustStatus(t, r, http.MethodGet, fmt.Sprintf("/sprints/%d/burndown?tz=Mars/Base", s1.ID), "", http.StatusBadRequest)

	check := func() {
		t.Helper()
		b := getJSON[domain.Burndown](t, r, burndownPath)
		if b.StartPoints != 8 {
			t.Errorf("StartPoints = %v, want 8 (A 3 + B 5)", b.StartPoints)
		}
		want := []struct {
			todoID    int
			kind      domain.BurndownEventKind
			remaining float64
		}{{3, domain.BurndownAdded, 10}, {2, domain.BurndownCompleted, 5}, {1, domain.BurndownEstimate, 10}}
		if len(b.Events) != len(want) {
			t.Fatalf("events = %+v", b.Events)
		}
		for i, w := range want {
			if e := b.Events[i]; e.TodoID != w.todoID || e.Kind != w.kind || e.Remaining != w.remaining || e.Key == "" {
				t.Errorf("event %d = %+v, want %+v", i, e, w)
			}
		}
		if len(b.Days) != 5 || b.Days[0].Date != today.Format("2006-01-02") || b.Days[0].Remaining == nil || *b.Days[0].Remaining != 10 || b.Days[4].Remaining != nil {
			t.Errorf("days = %+v", b.Days)
		}
	}
	check()

	// 完了: 未完了の A（とサブタスク）・C を次のスプリントへ。移動は履歴に残るが、完了したスプリントのバーンダウンは変わらない
	mustStatus(t, r, http.MethodPost, fmt.Sprintf("/sprints/%d/complete", s1.ID), fmt.Sprintf(`{"moveTo":%d}`, s2.ID), http.StatusOK)
	for _, todoID := range []int{1, 3, 4} {
		if e := lastSprintEntry(todoID); e.OldRefID == nil || *e.OldRefID != s1.ID || e.NewRefID == nil || *e.NewRefID != s2.ID || *e.NewValue != s2.Name {
			t.Errorf("todo %d sprint history after complete = %+v", todoID, e)
		}
	}
	check()

	// 未開始のスプリントを削除すると、バックログに戻ったことが履歴に残る
	mustStatus(t, r, http.MethodDelete, fmt.Sprintf("/sprints/%d", s2.ID), "", http.StatusOK)
	if e := lastSprintEntry(3); e.OldValue == nil || *e.OldValue != s2.Name || e.NewValue != nil || *e.NewRefID != 0 {
		t.Errorf("sprint history after delete = %+v", e)
	}
}

func TestSprint_Velocity(t *testing.T) {
	r := setupSprintRouter(t)
	mustStatus(t, r, http.MethodPost, "/spaces", `{"Title":"開発","Key":"DEV"}`, http.StatusCreated)
	mustStatus(t, r, http.MethodPost, "/spaces", `{"Title":"運用","Key":"OPS"}`, http.StatusCreated)
	mustStatus(t, r, http.MethodGet, "/spaces/99/velocity", "", http.StatusNotFound)
	mustStatus(t, r, http.MethodGet, "/spaces/1/velocity?tz=Mars/Base", "", http.StatusBadRequest)
	if v := getJSON[domain.Velocity](t, r, "/spaces/1/velocity"); len(v.Sprints) != 0 || v.Average != 0 {
		t.Errorf("velocity before sprints = %+v", v)
	}

	s1 := createSprint(t, r, 1, `{}`)
	s2 := createSprint(t, r, 1, `{}`)
	createSprint(t, r, 2, `{}`)
	mustStatus(t, r, http.MethodPost, "/todos", `{"title":"A","spaceId":1,"storyPoints":3}`, http.StatusCreated)
	mustStatus(t, r, http.MethodPost, "/todos", `{"title":"B","spaceId":1,"storyPoints":5}`, http.StatusCreated)
	setSprint(t, r, 1, s1.ID, http.StatusOK)
	setSprint(t, r, 2, s1.ID, http.StatusOK)
	today := time.Now().Format("2006-01-02")
	mustStatus(t, r, http.MethodPost, fmt.Sprintf("/sprints/%d/start", s1.ID), fmt.Sprintf(`{"name":"DEV 1","startDate":%q,"endDate":%q}`, today, today), http.StatusOK)
	time.Sleep(1100 * time.Millisecond)

	// 開始中のスプリントは並べない
	if v := getJSON[domain.Velocity](t, r, "/spaces/1/velocity"); len(v.Sprints) != 0 {
		t.Errorf("velocity with active sprint = %+v", v)
	}

	// B を完了して、A を次のスプリントへ
	mustStatus(t, r, http.MethodPut, "/todos/2", `{"title":"B","status":"Closed","spaceId":1,"storyPoints":5}`, http.StatusOK)
	mustStatus(t, r, http.MethodPost, fmt.Sprintf("/sprints/%d/complete", s1.ID), fmt.Sprintf(`{"moveTo":%d}`, s2.ID), http.StatusOK)

	v := getJSON[domain.Velocity](t, r, "/spaces/1/velocity?tz=Asia/Tokyo")
	if len(v.Sprints) != 1 || v.Sprints[0].SprintID != s1.ID || v.Sprints[0].Name != "DEV 1" || v.Sprints[0].Commitment != 8 || v.Sprints[0].Completed != 5 || v.Average != 5 {
		t.Errorf("velocity = %+v", v)
	}
	// 別のスペースには出ない
	if v := getJSON[domain.Velocity](t, r, "/spaces/2/velocity"); len(v.Sprints) != 0 {
		t.Errorf("velocity of other space = %+v", v)
	}
}
