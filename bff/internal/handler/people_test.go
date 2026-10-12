package handler

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"todo-api/internal/domain"
	"todo-api/internal/service"
)

// newPeopleRouter はログインが必要な課題・コメント・履歴・ユーザー一覧の API を持つルーターと、リポジトリを返す
func newPeopleRouter(t *testing.T) (*gin.Engine, testRepos) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	repos := newTestRepos(t)
	auth := NewAuthHandler(service.NewAuthService(repos.user, repos.session))
	todo := NewTodoHandler(service.NewTodoService(repos.todo, repos.epic, repos.space, repos.link, repos.history, repos.attachment, repos.status, repos.user, repos.watcher, repos.notification))
	comment := NewCommentHandler(service.NewCommentService(repos.comment, repos.todo, repos.watcher, repos.notification, repos.user))
	users := NewUserHandler(service.NewUserService(repos.user))

	r := gin.New()
	r.POST("/auth/signup", auth.Signup)
	api := r.Group("/", auth.RequireLogin)
	api.GET("/users", users.GetUsers)
	api.GET("/todos", todo.GetTodos)
	api.POST("/todos", todo.CreateTodo)
	api.PUT("/todos/:id", todo.UpdateTodo)
	api.GET("/todos/:id/history", todo.GetHistory)
	api.GET("/todos/:id/comments", comment.GetComments)
	api.POST("/todos/:id/comments", comment.CreateComment)
	return r, repos
}

// signup はユーザーを登録し、そのユーザーとセッションのトークンを返す
func signup(t *testing.T, r *gin.Engine, displayName, email string) (domain.User, string) {
	t.Helper()
	w := doAuthRequest(t, r, http.MethodPost, "/auth/signup",
		`{"displayName":"`+displayName+`","email":"`+email+`","password":"password"}`, "")
	if w.Code != http.StatusCreated {
		t.Fatalf("POST /auth/signup status = %d, body = %s", w.Code, w.Body)
	}
	var res struct{ User domain.User }
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		t.Fatal(err)
	}
	return res.User, sessionCookie(t, w).Value
}

// sendAs は token のユーザーとしてリクエストを送り、ステータスが want でなければテストを失敗させる
func sendAs(t *testing.T, r *gin.Engine, token, method, path, body string, want int) []byte {
	t.Helper()
	w := doAuthRequest(t, r, method, path, body, token)
	if w.Code != want {
		t.Fatalf("%s %s %s status = %d, want %d, body = %s", method, path, body, w.Code, want, w.Body)
	}
	return w.Body.Bytes()
}

func decode[T any](t *testing.T, body []byte) T {
	t.Helper()
	var v T
	if err := json.Unmarshal(body, &v); err != nil {
		t.Fatalf("decode %s: %v", body, err)
	}
	return v
}

func TestPeople_AssigneeAndReporter(t *testing.T) {
	r, _ := newPeopleRouter(t)
	taro, taroToken := signup(t, r, "山田 太郎", "taro@example.com")
	hanako, hanakoToken := signup(t, r, "佐藤 花子", "hanako@example.com")

	// ユーザー一覧（担当者の候補）は表示名の順で、パスワードのハッシュを含まない
	body := sendAs(t, r, taroToken, http.MethodGet, "/users", "", http.StatusOK)
	if strings.Contains(string(body), "PasswordHash") || strings.Contains(string(body), "$2a$") {
		t.Errorf("GET /users contains password hash: %s", body)
	}
	users := decode[[]domain.User](t, body)
	if len(users) != 2 || users[0].ID != hanako.ID || users[1].ID != taro.ID {
		t.Fatalf("users = %+v", users)
	}

	// 作成したユーザーが報告者になり、担当者は未割り当て
	created := decode[struct{ Todo domain.Todo }](t, sendAs(t, r, taroToken, http.MethodPost, "/todos", `{"title":"a"}`, http.StatusCreated)).Todo
	if created.ReporterID != taro.ID || created.Reporter == nil || *created.Reporter != (domain.UserSummary{ID: taro.ID, DisplayName: "山田 太郎", AvatarColor: taro.AvatarColor}) {
		t.Errorf("reporter = %d, %+v", created.ReporterID, created.Reporter)
	}
	if created.AssigneeID != 0 || created.Assignee != nil {
		t.Errorf("assignee = %d, %+v", created.AssigneeID, created.Assignee)
	}

	// 担当者を設定すると、課題に担当者の概要が付き、履歴に変更者が残る
	updated := decode[struct{ Todo domain.Todo }](t, sendAs(t, r, taroToken, http.MethodPut, "/todos/1",
		`{"title":"a","assigneeId":`+itoa(hanako.ID)+`}`, http.StatusOK)).Todo
	if updated.AssigneeID != hanako.ID || updated.Assignee == nil || updated.Assignee.DisplayName != "佐藤 花子" || updated.Assignee.AvatarColor != hanako.AvatarColor {
		t.Errorf("assignee = %d, %+v", updated.AssigneeID, updated.Assignee)
	}
	if updated.ReporterID != taro.ID {
		t.Errorf("reporter changed to %d", updated.ReporterID)
	}
	todos := decode[[]domain.Todo](t, sendAs(t, r, taroToken, http.MethodGet, "/todos", "", http.StatusOK))
	if len(todos) != 1 || todos[0].Assignee == nil || todos[0].Assignee.ID != hanako.ID {
		t.Fatalf("GET /todos = %+v", todos)
	}

	// 担当者を送らない更新（ドラッグでの移動など）では担当者は変わらない
	kept := decode[struct{ Todo domain.Todo }](t, sendAs(t, r, hanakoToken, http.MethodPut, "/todos/1", `{"title":"b"}`, http.StatusOK)).Todo
	if kept.AssigneeID != hanako.ID || kept.ReporterID != taro.ID {
		t.Errorf("kept = assignee %d, reporter %d", kept.AssigneeID, kept.ReporterID)
	}

	// null で未割り当てに戻す。報告者も変えられる
	cleared := decode[struct{ Todo domain.Todo }](t, sendAs(t, r, hanakoToken, http.MethodPut, "/todos/1",
		`{"title":"b","assigneeId":null,"reporterId":`+itoa(hanako.ID)+`}`, http.StatusOK)).Todo
	if cleared.AssigneeID != 0 || cleared.Assignee != nil || cleared.ReporterID != hanako.ID {
		t.Errorf("cleared = assignee %d %+v, reporter %d", cleared.AssigneeID, cleared.Assignee, cleared.ReporterID)
	}

	// 存在しないユーザーや ID でない値は 400
	sendAs(t, r, taroToken, http.MethodPut, "/todos/1", `{"title":"b","assigneeId":999}`, http.StatusBadRequest)
	sendAs(t, r, taroToken, http.MethodPut, "/todos/1", `{"title":"b","assigneeId":"taro"}`, http.StatusBadRequest)
	sendAs(t, r, taroToken, http.MethodPut, "/todos/1", `{"title":"b","reporterId":-1}`, http.StatusBadRequest)
	sendAs(t, r, taroToken, http.MethodPost, "/todos", `{"title":"c","assigneeId":999}`, http.StatusBadRequest)

	// 作成時に担当者を指定できる
	withAssignee := decode[struct{ Todo domain.Todo }](t, sendAs(t, r, hanakoToken, http.MethodPost, "/todos",
		`{"title":"c","assigneeId":`+itoa(taro.ID)+`}`, http.StatusCreated)).Todo
	if withAssignee.AssigneeID != taro.ID || withAssignee.ReporterID != hanako.ID {
		t.Errorf("created = assignee %d, reporter %d", withAssignee.AssigneeID, withAssignee.ReporterID)
	}

	// 履歴の各記録に変更したユーザーが付き、担当者・報告者の変更は表示名で残る
	history := decode[[]domain.HistoryEntry](t, sendAs(t, r, taroToken, http.MethodGet, "/todos/1/history", "", http.StatusOK))
	type entry struct{ field, oldValue, newValue, user string }
	got := []entry{}
	for _, h := range history {
		e := entry{field: h.Field}
		if h.OldValue != nil {
			e.oldValue = *h.OldValue
		}
		if h.NewValue != nil {
			e.newValue = *h.NewValue
		}
		if h.User != nil {
			e.user = h.User.DisplayName
		}
		if h.User == nil || h.UserID != h.User.ID {
			t.Errorf("history user = %d, %+v", h.UserID, h.User)
		}
		got = append(got, e)
	}
	want := []entry{
		{"created", "", "", "山田 太郎"},
		{"assignee", "", "佐藤 花子", "山田 太郎"},
		{"title", "a", "b", "佐藤 花子"},
		{"assignee", "佐藤 花子", "", "佐藤 花子"},
		{"reporter", "山田 太郎", "佐藤 花子", "佐藤 花子"},
	}
	if len(got) != len(want) {
		t.Fatalf("history = %+v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("history[%d] = %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestPeople_CommentAuthor(t *testing.T) {
	r, repos := newPeopleRouter(t)
	_, taroToken := signup(t, r, "山田 太郎", "taro@example.com")
	hanako, hanakoToken := signup(t, r, "佐藤 花子", "hanako@example.com")
	sendAs(t, r, taroToken, http.MethodPost, "/todos", `{"title":"a"}`, http.StatusCreated)

	// ユーザー機能の追加前のコメント（投稿者なし）
	if _, err := repos.comment.Create(domain.Comment{TodoID: 1, Body: "前からあるコメント"}); err != nil {
		t.Fatal(err)
	}
	created := decode[struct{ Comment domain.Comment }](t, sendAs(t, r, hanakoToken, http.MethodPost, "/todos/1/comments", `{"body":"確認しました"}`, http.StatusCreated)).Comment
	if created.AuthorID != hanako.ID || created.Author == nil || created.Author.DisplayName != "佐藤 花子" {
		t.Errorf("created comment author = %d, %+v", created.AuthorID, created.Author)
	}

	comments := decode[[]domain.Comment](t, sendAs(t, r, taroToken, http.MethodGet, "/todos/1/comments", "", http.StatusOK))
	if len(comments) != 2 {
		t.Fatalf("comments = %+v", comments)
	}
	if comments[0].AuthorID != 0 || comments[0].Author != nil {
		t.Errorf("old comment author = %d, %+v", comments[0].AuthorID, comments[0].Author)
	}
	if comments[1].Author == nil || comments[1].Author.ID != hanako.ID || comments[1].Author.AvatarColor != hanako.AvatarColor {
		t.Errorf("new comment author = %+v", comments[1].Author)
	}
}

func itoa(n int) string {
	b, _ := json.Marshal(n)
	return string(b)
}
