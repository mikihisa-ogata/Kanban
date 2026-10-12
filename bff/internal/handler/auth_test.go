package handler

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"todo-api/internal/database/dbtest"
	"todo-api/internal/domain"
	"todo-api/internal/repository"
	"todo-api/internal/service"
)

// newAuthRouter は認証の API と、ログインが必要な課題の一覧の API を持つルーターを返す
func newAuthRouter(t *testing.T) (*gin.Engine, *sql.DB) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db := dbtest.Open(t, "handler_auth")
	h := NewAuthHandler(service.NewAuthService(repository.NewUserMySQLRepository(db), repository.NewSessionMySQLRepository(db)))
	todo := NewTodoHandler(service.NewTodoService(
		repository.NewTodoMySQLRepository(db), repository.NewEpicMySQLRepository(db), repository.NewSpaceMySQLRepository(db),
		repository.NewIssueLinkMySQLRepository(db), repository.NewHistoryMySQLRepository(db),
		repository.NewAttachmentMySQLRepository(db, t.TempDir()), repository.NewStatusMySQLRepository(db)))

	r := gin.New()
	r.POST("/auth/signup", h.Signup)
	r.POST("/auth/login", h.Login)
	r.POST("/auth/logout", h.Logout)
	r.GET("/auth/me", h.Me)
	api := r.Group("/", h.RequireLogin)
	api.GET("/todos", todo.GetTodos)
	api.GET("/whoami", func(c *gin.Context) {
		user, ok := CurrentUser(c)
		if !ok {
			c.Status(http.StatusInternalServerError)
			return
		}
		c.JSON(http.StatusOK, user)
	})
	return r, db
}

// doAuthRequest はリクエストを送る。token が空でなければセッションの Cookie を付ける
func doAuthRequest(t *testing.T, r *gin.Engine, method, path, body, token string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.AddCookie(&http.Cookie{Name: SessionCookieName, Value: token})
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// sessionCookie はレスポンスが付けたセッションの Cookie を返す。付けていなければテストを失敗させる
func sessionCookie(t *testing.T, w *httptest.ResponseRecorder) *http.Cookie {
	t.Helper()
	for _, c := range w.Result().Cookies() {
		if c.Name == SessionCookieName {
			return c
		}
	}
	t.Fatalf("session cookie not set, headers = %v", w.Header())
	return nil
}

func TestAuth_SignupLoginLogout(t *testing.T) {
	r, db := newAuthRouter(t)

	// 未ログインでは API が 401 を返す
	if w := doAuthRequest(t, r, http.MethodGet, "/todos", "", ""); w.Code != http.StatusUnauthorized {
		t.Fatalf("GET /todos without login status = %d", w.Code)
	}
	if w := doAuthRequest(t, r, http.MethodGet, "/todos", "", "invalid-token"); w.Code != http.StatusUnauthorized {
		t.Fatalf("GET /todos with invalid token status = %d", w.Code)
	}
	if w := doAuthRequest(t, r, http.MethodGet, "/auth/me", "", ""); w.Code != http.StatusUnauthorized {
		t.Fatalf("GET /auth/me without login status = %d", w.Code)
	}

	// サインアップするとログインした状態になる
	w := doAuthRequest(t, r, http.MethodPost, "/auth/signup",
		`{"displayName":" 山田 太郎 ","email":" Taro@Example.com ","password":"correct-password"}`, "")
	if w.Code != http.StatusCreated {
		t.Fatalf("POST /auth/signup status = %d, body = %s", w.Code, w.Body)
	}
	if strings.Contains(w.Body.String(), "PasswordHash") || strings.Contains(w.Body.String(), "$2a$") {
		t.Errorf("signup response contains password hash: %s", w.Body)
	}
	var signedUp struct{ User domain.User }
	if err := json.Unmarshal(w.Body.Bytes(), &signedUp); err != nil {
		t.Fatal(err)
	}
	if signedUp.User.ID == 0 || signedUp.User.DisplayName != "山田 太郎" || signedUp.User.Email != "taro@example.com" ||
		signedUp.User.AvatarColor != domain.AvatarColorFor("taro@example.com") || signedUp.User.CreatedAt.IsZero() {
		t.Errorf("signed up user = %+v", signedUp.User)
	}
	cookie := sessionCookie(t, w)
	if !cookie.HttpOnly || cookie.Path != "/" || cookie.SameSite != http.SameSiteLaxMode || cookie.MaxAge <= 0 {
		t.Errorf("session cookie = %+v", cookie)
	}
	signupToken := cookie.Value

	if w := doAuthRequest(t, r, http.MethodGet, "/todos", "", signupToken); w.Code != http.StatusOK {
		t.Fatalf("GET /todos after signup status = %d, body = %s", w.Code, w.Body)
	}
	w = doAuthRequest(t, r, http.MethodGet, "/whoami", "", signupToken)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"DisplayName":"山田 太郎"`) {
		t.Fatalf("GET /whoami status = %d, body = %s", w.Code, w.Body)
	}

	// パスワードはハッシュにして保存する
	var hash string
	if err := db.QueryRow(`SELECT password_hash FROM users WHERE id = ?`, signedUp.User.ID).Scan(&hash); err != nil {
		t.Fatal(err)
	}
	if hash == "correct-password" || !strings.HasPrefix(hash, "$2a$") {
		t.Errorf("password_hash = %q", hash)
	}
	// セッションのトークンはそのまま保存しない
	var stored int
	if err := db.QueryRow(`SELECT COUNT(*) FROM sessions WHERE token_hash = ?`, signupToken).Scan(&stored); err != nil {
		t.Fatal(err)
	}
	if stored != 0 {
		t.Error("session token is stored as is")
	}

	// 大文字・小文字が違うだけのメールアドレスでもログインできる
	w = doAuthRequest(t, r, http.MethodPost, "/auth/login", `{"email":"TARO@example.com","password":"correct-password"}`, "")
	if w.Code != http.StatusOK {
		t.Fatalf("POST /auth/login status = %d, body = %s", w.Code, w.Body)
	}
	loginToken := sessionCookie(t, w).Value
	if loginToken == signupToken {
		t.Error("login reuses the signup session token")
	}
	w = doAuthRequest(t, r, http.MethodGet, "/auth/me", "", loginToken)
	if w.Code != http.StatusOK {
		t.Fatalf("GET /auth/me status = %d, body = %s", w.Code, w.Body)
	}
	var me domain.User
	if err := json.Unmarshal(w.Body.Bytes(), &me); err != nil {
		t.Fatal(err)
	}
	if me.ID != signedUp.User.ID || me.Email != "taro@example.com" {
		t.Errorf("me = %+v", me)
	}

	// ログアウトすると、そのセッションでは API を使えず、Cookie も消える
	w = doAuthRequest(t, r, http.MethodPost, "/auth/logout", "", loginToken)
	if w.Code != http.StatusOK {
		t.Fatalf("POST /auth/logout status = %d, body = %s", w.Code, w.Body)
	}
	if cleared := sessionCookie(t, w); cleared.Value != "" || cleared.MaxAge >= 0 {
		t.Errorf("cleared cookie = %+v", cleared)
	}
	if w := doAuthRequest(t, r, http.MethodGet, "/todos", "", loginToken); w.Code != http.StatusUnauthorized {
		t.Fatalf("GET /todos after logout status = %d", w.Code)
	}
	if w := doAuthRequest(t, r, http.MethodGet, "/auth/me", "", loginToken); w.Code != http.StatusUnauthorized {
		t.Fatalf("GET /auth/me after logout status = %d", w.Code)
	}
	// 別のセッション（サインアップのとき）はログアウトしていない
	if w := doAuthRequest(t, r, http.MethodGet, "/todos", "", signupToken); w.Code != http.StatusOK {
		t.Fatalf("GET /todos with another session status = %d", w.Code)
	}
	// ログインしていなくてもログアウトは成功にする
	if w := doAuthRequest(t, r, http.MethodPost, "/auth/logout", "", ""); w.Code != http.StatusOK {
		t.Fatalf("POST /auth/logout without login status = %d", w.Code)
	}
}

func TestAuth_LoginRejectsWrongPassword(t *testing.T) {
	r, _ := newAuthRouter(t)
	if w := doAuthRequest(t, r, http.MethodPost, "/auth/signup",
		`{"displayName":"taro","email":"taro@example.com","password":"correct-password"}`, ""); w.Code != http.StatusCreated {
		t.Fatalf("POST /auth/signup status = %d, body = %s", w.Code, w.Body)
	}

	for _, body := range []string{
		`{"email":"taro@example.com","password":"wrong-password"}`,
		`{"email":"taro@example.com","password":"correct-passwor"}`,
		`{"email":"taro@example.com","password":"CORRECT-PASSWORD"}`,
		`{"email":"taro@example.com","password":""}`,
		`{"email":"hanako@example.com","password":"correct-password"}`,
	} {
		w := doAuthRequest(t, r, http.MethodPost, "/auth/login", body, "")
		if w.Code != http.StatusUnauthorized {
			t.Errorf("POST /auth/login %s status = %d, want 401", body, w.Code)
		}
		for _, c := range w.Result().Cookies() {
			if c.Name == SessionCookieName {
				t.Errorf("POST /auth/login %s sets session cookie", body)
			}
		}
		// メールアドレスが登録されているかどうかを区別できないよう、同じメッセージを返す
		if !strings.Contains(w.Body.String(), service.ErrInvalidCredentials.Error()) {
			t.Errorf("POST /auth/login %s body = %s", body, w.Body)
		}
	}
}

func TestAuth_SignupValidation(t *testing.T) {
	r, _ := newAuthRouter(t)
	mustAuth := func(body string, want int) {
		t.Helper()
		if w := doAuthRequest(t, r, http.MethodPost, "/auth/signup", body, ""); w.Code != want {
			t.Errorf("POST /auth/signup %s status = %d, want %d, body = %s", body, w.Code, want, w.Body)
		}
	}
	mustAuth(`{"displayName":"taro","email":"taro@example.com","password":"password"}`, http.StatusCreated)
	// 大文字・小文字が違うだけのメールアドレスは登録済みとみなす
	mustAuth(`{"displayName":"taro2","email":"TARO@example.com","password":"password"}`, http.StatusConflict)
	mustAuth(`{"displayName":"  ","email":"a@example.com","password":"password"}`, http.StatusBadRequest)
	mustAuth(`{"displayName":"a","email":"not-an-email","password":"password"}`, http.StatusBadRequest)
	mustAuth(`{"displayName":"a","email":"a@example.com","password":"short"}`, http.StatusBadRequest)
	mustAuth(`not json`, http.StatusBadRequest)
}
