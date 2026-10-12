package handler

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"

	"todo-api/internal/domain"
	"todo-api/internal/service"
)

// newNotificationRouter は newPeopleRouter に、通知・ウォッチ・課題の削除の API を足したルーターを返す
func newNotificationRouter(t *testing.T) *gin.Engine {
	t.Helper()
	r, repos := newPeopleRouter(t)
	auth := NewAuthHandler(service.NewAuthService(repos.user, repos.session))
	todo := NewTodoHandler(service.NewTodoService(repos.todo, repos.epic, repos.space, repos.link, repos.history, repos.attachment, repos.status, repos.user, repos.watcher, repos.notification))
	notifications := NewNotificationHandler(service.NewNotificationService(repos.notification))
	watchers := NewWatcherHandler(service.NewWatcherService(repos.watcher, repos.todo))
	api := r.Group("/", auth.RequireLogin)
	api.DELETE("/todos/:id", todo.DeleteTodo)
	api.PUT("/comments/:id", NewCommentHandler(service.NewCommentService(repos.comment, repos.todo, repos.watcher, repos.notification, repos.user)).UpdateComment)
	api.GET("/notifications", notifications.GetNotifications)
	api.POST("/notifications/read-all", notifications.MarkAllRead)
	api.POST("/notifications/:id/read", notifications.MarkRead)
	api.GET("/todos/:id/watchers", watchers.GetWatchers)
	api.POST("/todos/:id/watch", watchers.Watch)
	api.DELETE("/todos/:id/watch", watchers.Unwatch)
	return r
}

// notificationsOf は token のユーザーへの通知を返す
func notificationsOf(t *testing.T, r *gin.Engine, token string) service.Notifications {
	t.Helper()
	return decode[service.Notifications](t, sendAs(t, r, token, http.MethodGet, "/notifications", "", http.StatusOK))
}

// watcherNames はウォッチャーの表示名を返す
func watcherNames(watchers []domain.UserSummary) []string {
	names := []string{}
	for _, w := range watchers {
		names = append(names, w.DisplayName)
	}
	return names
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestNotifications_WatchersGetNotified(t *testing.T) {
	r := newNotificationRouter(t)
	taro, taroToken := signup(t, r, "A 太郎", "taro@example.com")
	hanako, hanakoToken := signup(t, r, "B 花子", "hanako@example.com")
	_, jiroToken := signup(t, r, "C 次郎", "jiro@example.com")

	// 作成者は自動でウォッチャーになる。自分の変更では通知しない
	sendAs(t, r, taroToken, http.MethodPost, "/todos", `{"title":"a"}`, http.StatusCreated)
	if got := watcherNames(decode[[]domain.UserSummary](t, sendAs(t, r, jiroToken, http.MethodGet, "/todos/1/watchers", "", http.StatusOK))); !equalStrings(got, []string{"A 太郎"}) {
		t.Fatalf("watchers = %v", got)
	}
	sendAs(t, r, taroToken, http.MethodPut, "/todos/1", `{"title":"b"}`, http.StatusOK)
	if n := notificationsOf(t, r, taroToken); n.Unread != 0 || len(n.Notifications) != 0 {
		t.Fatalf("taro notifications after own change = %+v", n)
	}

	// 担当者は自動でウォッチャーになり、割り当てられたことが通知される
	sendAs(t, r, taroToken, http.MethodPut, "/todos/1", `{"title":"b","assigneeId":`+itoa(hanako.ID)+`}`, http.StatusOK)
	n := notificationsOf(t, r, hanakoToken)
	if n.Unread != 1 || len(n.Notifications) != 1 {
		t.Fatalf("hanako notifications = %+v", n)
	}
	got := n.Notifications[0]
	if got.Kind != domain.NotificationUpdated || got.Detail != "assignee" || got.IssueKey != "TASK-1" || got.IssueTitle != "b" ||
		got.ActorID != taro.ID || got.Actor == nil || got.Actor.DisplayName != "A 太郎" || got.Read || got.TodoID != 1 || got.CreatedAt.IsZero() {
		t.Errorf("notification = %+v", got)
	}

	// コメントした人は自動でウォッチャーになり、ほかのウォッチャーに通知される
	sendAs(t, r, jiroToken, http.MethodPost, "/todos/1/comments", `{"body":"確認しました\n\nよろしく"}`, http.StatusCreated)
	if got := watcherNames(decode[[]domain.UserSummary](t, sendAs(t, r, taroToken, http.MethodGet, "/todos/1/watchers", "", http.StatusOK))); !equalStrings(got, []string{"A 太郎", "B 花子", "C 次郎"}) {
		t.Fatalf("watchers = %v", got)
	}
	for _, token := range []string{taroToken, hanakoToken} {
		latest := notificationsOf(t, r, token).Notifications[0]
		if latest.Kind != domain.NotificationCommented || latest.Detail != "確認しました よろしく" || latest.Actor == nil || latest.Actor.DisplayName != "C 次郎" {
			t.Errorf("comment notification = %+v", latest)
		}
	}
	if n := notificationsOf(t, r, jiroToken); len(n.Notifications) != 0 {
		t.Errorf("jiro got notifications for own comment: %+v", n)
	}

	// ウォッチを外すと通知されない。ステータスとカテゴリが同時に変わっても 1 件にまとめる
	watchers := decode[[]domain.UserSummary](t, sendAs(t, r, hanakoToken, http.MethodDelete, "/todos/1/watch", "", http.StatusOK))
	if got := watcherNames(watchers); !equalStrings(got, []string{"A 太郎", "C 次郎"}) {
		t.Fatalf("watchers after unwatch = %v", got)
	}
	sendAs(t, r, jiroToken, http.MethodPut, "/todos/1", `{"title":"b","priority":"High","status":"InProgress"}`, http.StatusOK)
	if n := notificationsOf(t, r, hanakoToken); len(n.Notifications) != 2 {
		t.Errorf("hanako got notified after unwatch: %+v", n)
	}
	n = notificationsOf(t, r, taroToken)
	if n.Unread != 2 || len(n.Notifications) != 2 || n.Notifications[0].Detail != "status,priority" {
		t.Fatalf("taro notifications = %+v", n)
	}

	// もう一度ウォッチできる（重ねてもウォッチャーは 1 人）
	sendAs(t, r, hanakoToken, http.MethodPost, "/todos/1/watch", "", http.StatusOK)
	watchers = decode[[]domain.UserSummary](t, sendAs(t, r, hanakoToken, http.MethodPost, "/todos/1/watch", "", http.StatusOK))
	if got := watcherNames(watchers); !equalStrings(got, []string{"A 太郎", "B 花子", "C 次郎"}) {
		t.Fatalf("watchers after watch = %v", got)
	}
	sendAs(t, r, hanakoToken, http.MethodPost, "/todos/99/watch", "", http.StatusNotFound)
	sendAs(t, r, hanakoToken, http.MethodGet, "/todos/99/watchers", "", http.StatusNotFound)

	// 既読にする。他人の通知は既読にできない
	first := n.Notifications[1].ID
	sendAs(t, r, hanakoToken, http.MethodPost, "/notifications/"+itoa(first)+"/read", "", http.StatusNotFound)
	sendAs(t, r, taroToken, http.MethodPost, "/notifications/"+itoa(first)+"/read", "", http.StatusOK)
	sendAs(t, r, taroToken, http.MethodPost, "/notifications/"+itoa(first)+"/read", "", http.StatusOK)
	n = notificationsOf(t, r, taroToken)
	if n.Unread != 1 || n.Notifications[0].Read || !n.Notifications[1].Read {
		t.Fatalf("after mark read = %+v", n)
	}
	sendAs(t, r, taroToken, http.MethodPost, "/notifications/999/read", "", http.StatusNotFound)
	sendAs(t, r, taroToken, http.MethodPost, "/notifications/read-all", "", http.StatusOK)
	if n := notificationsOf(t, r, taroToken); n.Unread != 0 || len(n.Notifications) != 2 || !n.Notifications[0].Read {
		t.Fatalf("after mark all read = %+v", n)
	}
	// 他人の通知は既読にならない
	if n := notificationsOf(t, r, hanakoToken); n.Unread != 2 {
		t.Errorf("hanako unread = %d", n.Unread)
	}

	// 課題を削除すると、その課題の通知とウォッチャーも消える
	sendAs(t, r, taroToken, http.MethodDelete, "/todos/1", "", http.StatusOK)
	if n := notificationsOf(t, r, hanakoToken); n.Unread != 0 || len(n.Notifications) != 0 {
		t.Errorf("notifications after delete = %+v", n)
	}
}

// kindsOf は通知の種類と内容を「kind:detail」の形で新しい順に返す
func kindsOf(n service.Notifications) []string {
	kinds := []string{}
	for _, x := range n.Notifications {
		kinds = append(kinds, string(x.Kind)+":"+x.Detail)
	}
	return kinds
}

func TestNotifications_Mentions(t *testing.T) {
	r := newNotificationRouter(t)
	_, taroToken := signup(t, r, "A 太郎", "taro@example.com")
	hanako, hanakoToken := signup(t, r, "B 花子", "hanako@example.com")
	jiro, jiroToken := signup(t, r, "C 次郎", "jiro@example.com")
	mention := func(name string, id int) string { return "@[" + name + "](user:" + itoa(id) + ")" }
	body := func(text string) string { b, _ := json.Marshal(text); return string(b) }

	// 説明でメンションされたユーザーに通知する（メンションだけではウォッチャーにならない）
	description := mention("B 花子", hanako.ID) + " 見てください"
	sendAs(t, r, taroToken, http.MethodPost, "/todos", `{"title":"a","description":`+body(description)+`}`, http.StatusCreated)
	if got := kindsOf(notificationsOf(t, r, hanakoToken)); !equalStrings(got, []string{"mentioned:@B 花子 見てください"}) {
		t.Fatalf("hanako notifications = %v", got)
	}
	if got := notificationsOf(t, r, hanakoToken).Notifications[0]; got.Actor == nil || got.Actor.DisplayName != "A 太郎" || got.IssueKey != "TASK-1" {
		t.Errorf("mention notification = %+v", got)
	}
	if got := kindsOf(notificationsOf(t, r, jiroToken)); len(got) != 0 {
		t.Errorf("jiro notifications = %v", got)
	}

	// コメントでメンションされたウォッチャーには「コメントしました」を重ねない。自分と存在しないユーザーへのメンションは通知しない
	sendAs(t, r, hanakoToken, http.MethodPost, "/todos/1/watch", "", http.StatusOK)
	comment := mention("B 花子", hanako.ID) + " " + mention("C 次郎", jiro.ID) + " " + mention("A 太郎", 1) + " " + mention("誰か", 999) + " よろしく"
	created := decode[struct{ Comment domain.Comment }](t, sendAs(t, r, taroToken, http.MethodPost, "/todos/1/comments", `{"body":`+body(comment)+`}`, http.StatusCreated)).Comment
	if created.Body != comment {
		t.Errorf("comment body = %q", created.Body)
	}
	plain := "@B 花子 @C 次郎 @A 太郎 @誰か よろしく"
	if got := kindsOf(notificationsOf(t, r, hanakoToken)); !equalStrings(got, []string{"mentioned:" + plain, "mentioned:@B 花子 見てください"}) {
		t.Errorf("hanako notifications = %v", got)
	}
	if got := kindsOf(notificationsOf(t, r, jiroToken)); !equalStrings(got, []string{"mentioned:" + plain}) {
		t.Errorf("jiro notifications = %v", got)
	}
	if got := kindsOf(notificationsOf(t, r, taroToken)); len(got) != 0 {
		t.Errorf("taro notifications = %v", got)
	}

	// 編集では、新しくメンションされたユーザーだけに通知する
	sendAs(t, r, taroToken, http.MethodPut, "/comments/"+itoa(created.ID), `{"body":`+body(comment+"（追記）")+`}`, http.StatusOK)
	if n := notificationsOf(t, r, jiroToken); len(n.Notifications) != 1 {
		t.Errorf("jiro notified again on edit: %v", kindsOf(n))
	}
	sendAs(t, r, hanakoToken, http.MethodPut, "/todos/1", `{"title":"a","description":`+body(description+" "+mention("C 次郎", jiro.ID))+`}`, http.StatusOK)
	if got := kindsOf(notificationsOf(t, r, jiroToken)); len(got) != 2 || got[0] != "mentioned:@B 花子 見てください @C 次郎" {
		t.Errorf("jiro notifications after description edit = %v", got)
	}
	if got := notificationsOf(t, r, jiroToken).Notifications[0]; got.Actor == nil || got.Actor.ID != hanako.ID {
		t.Errorf("description mention actor = %+v", got.Actor)
	}
	// ウォッチ中の作成者には、説明の変更が通知される
	if got := kindsOf(notificationsOf(t, r, taroToken)); !equalStrings(got, []string{"updated:description"}) {
		t.Errorf("taro notifications = %v", got)
	}
	// 同じ説明のまま保存しても、もう一度は通知しない
	sendAs(t, r, hanakoToken, http.MethodPut, "/todos/1", `{"title":"b","description":`+body(description+" "+mention("C 次郎", jiro.ID))+`}`, http.StatusOK)
	if n := notificationsOf(t, r, jiroToken); len(n.Notifications) != 2 {
		t.Errorf("jiro notified again: %v", kindsOf(n))
	}
}
