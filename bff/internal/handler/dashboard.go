package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"todo-api/internal/domain"
	"todo-api/internal/service"
)

type DashboardHandler struct {
	service service.DashboardService
}

func NewDashboardHandler(s service.DashboardService) *DashboardHandler {
	return &DashboardHandler{service: s}
}

// GetDashboard はログイン中のユーザーのダッシュボード（ガジェットの並び）を返す。保存していなければ既定の並び
func (h *DashboardHandler) GetDashboard(c *gin.Context) {
	gadgets, err := h.service.GetDashboard(actorID(c))
	if err != nil {
		respondDashboardError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"gadgets": gadgets,
	})
}

// SaveDashboard はログイン中のユーザーのガジェットの並び（gadgets: [{id, type, filterId}]）を保存する
func (h *DashboardHandler) SaveDashboard(c *gin.Context) {
	var req struct {
		Gadgets []struct {
			ID       string `json:"id"`
			Type     string `json:"type"`
			FilterID int    `json:"filterId"`
		} `json:"gadgets"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}
	gadgets := make([]domain.DashboardGadget, len(req.Gadgets))
	for i, g := range req.Gadgets {
		gadgets[i] = domain.DashboardGadget{ID: g.ID, Type: domain.GadgetType(g.Type), FilterID: g.FilterID}
	}
	saved, err := h.service.SaveDashboard(actorID(c), gadgets)
	if err != nil {
		respondDashboardError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message": "ダッシュボードを保存しました",
		"gadgets": saved,
	})
}

// GetActivity はすべての課題の最近の変更履歴を新しい順に返す（limit で件数。既定 30、最大 100）
func (h *DashboardHandler) GetActivity(c *gin.Context) {
	limit, ok := queryInt(c, "limit", service.ActivityLimitDefault, service.ErrInvalidActivityLimit)
	if !ok {
		return
	}
	items, err := h.service.GetActivity(limit)
	if err != nil {
		respondDashboardError(c, err)
		return
	}
	c.JSON(http.StatusOK, items)
}

// GetStatusCounts は JQL（クエリ文字列 jql。空はすべての課題）に合う課題をステータスごとに数えて返す
func (h *DashboardHandler) GetStatusCounts(c *gin.Context) {
	counts, err := h.service.CountByStatus(c.Query("jql"), actorID(c))
	if err != nil {
		respondDashboardError(c, err)
		return
	}
	c.JSON(http.StatusOK, counts)
}

// GetCreatedVsResolved は JQL（クエリ文字列 jql）に合う課題の、今日までの days 日間（既定 30）の日ごとの作成数と解決数を返す
func (h *DashboardHandler) GetCreatedVsResolved(c *gin.Context) {
	days, ok := queryInt(c, "days", 30, domain.ErrInvalidTrendDays)
	if !ok {
		return
	}
	result, err := h.service.CreatedVsResolved(c.Query("jql"), days, actorID(c))
	if err != nil {
		respondDashboardError(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

// queryInt はクエリ文字列 name の整数を返す。なければ fallback、整数でなければ invalid のメッセージで 400 を返して ok を false にする
func queryInt(c *gin.Context, name string, fallback int, invalid error) (int, bool) {
	v := c.Query(name)
	if v == "" {
		return fallback, true
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": invalid.Error(),
		})
		return 0, false
	}
	return n, true
}

// respondDashboardError はエラーを返す。JQL の誤りは検索と同じく position（誤りの位置）も返す
func respondDashboardError(c *gin.Context, err error) {
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
	case errors.Is(err, domain.ErrTooManyGadgets), errors.Is(err, domain.ErrInvalidGadgetID),
		errors.Is(err, domain.ErrInvalidGadgetType), errors.Is(err, domain.ErrInvalidGadgetFilter),
		errors.Is(err, domain.ErrInvalidTrendDays), errors.Is(err, service.ErrInvalidActivityLimit):
		status = http.StatusBadRequest
	}
	c.JSON(status, gin.H{
		"error": err.Error(),
	})
}
