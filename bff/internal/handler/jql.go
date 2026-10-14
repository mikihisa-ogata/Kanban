package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"todo-api/internal/domain"
	"todo-api/internal/service"
)

type JQLHandler struct {
	service service.JQLService
}

func NewJQLHandler(s service.JQLService) *JQLHandler {
	return &JQLHandler{service: s}
}

// SearchJQL は JQL（クエリ文字列 jql）で課題を検索し、1 ページ分を返す（page・perPage は課題一覧と同じ）。
// JQL の誤りは 400 で、error にメッセージ、position に誤りの位置（先頭からの文字数。0 から）を返す
func (h *JQLHandler) SearchJQL(c *gin.Context) {
	page, perPage := 0, 0
	for name, dest := range map[string]*int{"page": &page, "perPage": &perPage} {
		v := c.Query(name)
		if v == "" {
			continue
		}
		n, err := strconv.Atoi(v)
		if err != nil || n <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": domain.ErrInvalidIssuePage.Error(),
			})
			return
		}
		*dest = n
	}

	result, err := h.service.SearchJQL(c.Query("jql"), page, perPage, actorID(c))
	var jqlErr *service.JQLError
	if errors.As(err, &jqlErr) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":    jqlErr.Error(),
			"position": jqlErr.Pos,
		})
		return
	}
	if errors.Is(err, domain.ErrInvalidIssuePage) || errors.Is(err, domain.ErrInvalidIssueSort) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, result)
}
