package handler

import (
	"errors"
	"net/http"

	"todo-api/internal/domain"
	"todo-api/internal/service"

	"github.com/gin-gonic/gin"
)

const (
	// SessionCookieName はログインのセッションのトークンを入れる Cookie の名前
	SessionCookieName = "kanban_session"
	// currentUserKey はログイン中のユーザーを gin.Context に入れるキー
	currentUserKey = "currentUser"
)

type AuthHandler struct {
	service service.AuthService
}

func NewAuthHandler(s service.AuthService) *AuthHandler {
	return &AuthHandler{service: s}
}

// Signup はユーザーを作成してログインする（セッションの Cookie を付ける）
func (h *AuthHandler) Signup(c *gin.Context) {
	var req struct {
		DisplayName string `json:"displayName"`
		Email       string `json:"email"`
		Password    string `json:"password"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	user, token, err := h.service.Signup(req.DisplayName, req.Email, req.Password)
	if err != nil {
		respondAuthError(c, err)
		return
	}
	setSessionCookie(c, token)
	c.JSON(http.StatusCreated, gin.H{
		"message": "ユーザーの登録に成功しました",
		"user":    user,
	})
}

// Login はメールアドレスとパスワードでログインする（セッションの Cookie を付ける）
func (h *AuthHandler) Login(c *gin.Context) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	user, token, err := h.service.Login(req.Email, req.Password)
	if err != nil {
		respondAuthError(c, err)
		return
	}
	setSessionCookie(c, token)
	c.JSON(http.StatusOK, gin.H{
		"message": "ログインに成功しました",
		"user":    user,
	})
}

// Logout はセッションを削除し、Cookie を消す。ログインしていなくても成功にする
func (h *AuthHandler) Logout(c *gin.Context) {
	token, _ := c.Cookie(SessionCookieName)
	if err := h.service.Logout(token); err != nil {
		respondAuthError(c, err)
		return
	}
	setSessionCookie(c, "")
	c.JSON(http.StatusOK, gin.H{
		"message": "ログアウトしました",
	})
}

// Me はログイン中のユーザーを返す。ログインしていなければ 401 を返す
func (h *AuthHandler) Me(c *gin.Context) {
	user, err := h.service.CurrentUser(sessionToken(c))
	if err != nil {
		respondAuthError(c, err)
		return
	}
	c.JSON(http.StatusOK, user)
}

// RequireLogin はログインしていないリクエストを 401 で止めるミドルウェア。
// ログイン中のユーザーは gin.Context に入れる（CurrentUser で取り出す）
func (h *AuthHandler) RequireLogin(c *gin.Context) {
	user, err := h.service.CurrentUser(sessionToken(c))
	if err != nil {
		respondAuthError(c, err)
		c.Abort()
		return
	}
	c.Set(currentUserKey, user)
	c.Next()
}

// CurrentUser は RequireLogin が入れたログイン中のユーザーを返す
func CurrentUser(c *gin.Context) (domain.User, bool) {
	user, ok := c.Get(currentUserKey)
	if !ok {
		return domain.User{}, false
	}
	u, ok := user.(domain.User)
	return u, ok
}

func sessionToken(c *gin.Context) string {
	token, _ := c.Cookie(SessionCookieName)
	return token
}

// setSessionCookie はセッションの Cookie を付ける。token が空なら Cookie を消す。
// JavaScript から読めないよう HttpOnly にし、別サイトからの送信を防ぐため SameSite=Lax にする。
// localhost の http で使うので Secure は付けない
func setSessionCookie(c *gin.Context, token string) {
	maxAge := int(service.SessionDuration.Seconds())
	if token == "" {
		maxAge = -1
	}
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(SessionCookieName, token, maxAge, "/", "", false, true)
}

// respondAuthError は認証の操作のエラーを、種類に合ったステータスで返す
func respondAuthError(c *gin.Context, err error) {
	status := http.StatusInternalServerError
	switch {
	case errors.Is(err, service.ErrInvalidCredentials), errors.Is(err, service.ErrUnauthenticated):
		status = http.StatusUnauthorized
	case errors.Is(err, service.ErrEmailTaken):
		status = http.StatusConflict
	case errors.Is(err, domain.ErrInvalidDisplayName), errors.Is(err, domain.ErrInvalidEmail), errors.Is(err, domain.ErrInvalidPassword):
		status = http.StatusBadRequest
	}
	c.JSON(status, gin.H{
		"error": err.Error(),
	})
}
