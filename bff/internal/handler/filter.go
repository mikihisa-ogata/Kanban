package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"todo-api/internal/domain"
	"todo-api/internal/service"
)

type FilterHandler struct {
	service service.FilterService
	views   service.IssueViewService
}

func NewFilterHandler(s service.FilterService, v service.IssueViewService) *FilterHandler {
	return &FilterHandler{service: s, views: v}
}

// GetFilters はログイン中のユーザーの保存済みフィルターを名前順に返す
func (h *FilterHandler) GetFilters(c *gin.Context) {
	filters, err := h.service.GetFilters(actorID(c))
	if err != nil {
		respondFilterError(c, err)
		return
	}
	c.JSON(http.StatusOK, filters)
}

// CreateFilter は検索条件（jql）に名前を付けて保存する。favourite でお気に入りにもできる
func (h *FilterHandler) CreateFilter(c *gin.Context) {
	var req struct {
		Name      string `json:"name"`
		JQL       string `json:"jql"`
		Favourite bool   `json:"favourite"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}
	filter, err := h.service.CreateFilter(actorID(c), req.Name, req.JQL, req.Favourite)
	if err != nil {
		respondFilterError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"message": "フィルターを保存しました",
		"filter":  filter,
	})
}

// UpdateFilter はフィルター（:id）の名前と JQL を変える（上書き保存）
func (h *FilterHandler) UpdateFilter(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var req struct {
		Name string `json:"name"`
		JQL  string `json:"jql"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}
	filter, err := h.service.UpdateFilter(id, actorID(c), req.Name, req.JQL)
	if err != nil {
		respondFilterError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message": "フィルターを更新しました",
		"filter":  filter,
	})
}

// SetFavourite はフィルター（:id）のお気に入り（スター）を付ける・外す
func (h *FilterHandler) SetFavourite(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var req struct {
		Favourite bool `json:"favourite"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}
	filter, err := h.service.SetFavourite(id, actorID(c), req.Favourite)
	if err != nil {
		respondFilterError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message": "お気に入りを変更しました",
		"filter":  filter,
	})
}

// DeleteFilter はフィルター（:id）を削除する
func (h *FilterHandler) DeleteFilter(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if err := h.service.DeleteFilter(id, actorID(c)); err != nil {
		respondFilterError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message": "フィルターを削除しました",
	})
}

// RecordView はログイン中のユーザーが課題（:id）を表示したことを記録する（JQL の issueHistory() に出る）
func (h *FilterHandler) RecordView(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if err := h.views.RecordView(id, actorID(c)); err != nil {
		respondFilterError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message": "表示を記録しました",
	})
}

// respondFilterError はエラーを返す。JQL の構文の誤りは検索と同じく position（誤りの位置）も返す
func respondFilterError(c *gin.Context, err error) {
	var jqlErr *service.JQLError
	if errors.As(err, &jqlErr) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":    jqlErr.Error(),
			"position": jqlErr.Pos,
		})
		return
	}
	status := http.StatusInternalServerError
	switch {
	case errors.Is(err, service.ErrFilterNotFound), errors.Is(err, service.ErrIssueNotFound):
		status = http.StatusNotFound
	case errors.Is(err, service.ErrFilterNameDuplicate):
		status = http.StatusConflict
	case errors.Is(err, domain.ErrInvalidFilterName), errors.Is(err, domain.ErrInvalidFilterJQL):
		status = http.StatusBadRequest
	}
	respondError(c, status, err)
}
