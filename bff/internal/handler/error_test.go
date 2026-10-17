package handler

import (
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRespondError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/internal", func(c *gin.Context) {
		respondError(c, http.StatusInternalServerError, errors.New("Error 1146 (42S02): Table 'kanban.todos' doesn't exist"))
	})
	r.GET("/bad", func(c *gin.Context) {
		respondError(c, http.StatusBadRequest, errors.New("タイトルは必須です"))
	})

	// 500 は内部のエラー文を返さない
	w := doRequest(t, r, http.MethodGet, "/internal", "")
	if w.Code != http.StatusInternalServerError || strings.Contains(w.Body.String(), "kanban.todos") || !strings.Contains(w.Body.String(), internalErrorMessage) {
		t.Errorf("500 response = %d %s", w.Code, w.Body)
	}
	// 4xx は利用者が直せるよう、エラー文をそのまま返す
	w = doRequest(t, r, http.MethodGet, "/bad", "")
	if w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), "タイトルは必須です") {
		t.Errorf("400 response = %d %s", w.Code, w.Body)
	}
}
