package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"todo-api/internal/domain"
	"todo-api/internal/service"

	"github.com/gin-gonic/gin"
)

type WorklogHandler struct {
	service service.WorklogService
}

func NewWorklogHandler(s service.WorklogService) *WorklogHandler {
	return &WorklogHandler{service: s}
}

// worklogRequest は作業ログの作成・更新のリクエスト
type worklogRequest struct {
	TimeSpent string `json:"timeSpent"` // 「1w 2d 3h 30m」形式
	StartedAt string `json:"startedAt"` // RFC 3339 形式（例: 2026-10-11T10:00:00+09:00）
	Note      string `json:"note"`
	// 残り時間。省略すると自動で調整し、文字列ならその値、null か空文字なら未設定にする
	RemainingEstimate json.RawMessage `json:"remainingEstimate"`
}

// GetWorklogs は課題（:id）の作業ログを開始日時の順に返す
func (h *WorklogHandler) GetWorklogs(c *gin.Context) {
	todoID, ok := parseID(c)
	if !ok {
		return
	}
	worklogs, err := h.service.GetWorklogs(todoID)
	if err != nil {
		respondWorklogError(c, err)
		return
	}

	c.JSON(http.StatusOK, worklogs)
}

// AddWorklog は課題（:id）に作業を記録する
func (h *WorklogHandler) AddWorklog(c *gin.Context) {
	todoID, ok := parseID(c)
	if !ok {
		return
	}
	startedAt, remaining, req, ok := bindWorklogRequest(c)
	if !ok {
		return
	}

	worklog, err := h.service.AddWorklog(todoID, req.TimeSpent, startedAt, req.Note, remaining, actorID(c))
	if err != nil {
		respondWorklogError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "作業の記録に成功しました",
		"worklog": worklog,
	})
}

// UpdateWorklog は作業ログ（:id）を変える
func (h *WorklogHandler) UpdateWorklog(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	startedAt, remaining, req, ok := bindWorklogRequest(c)
	if !ok {
		return
	}

	worklog, err := h.service.UpdateWorklog(id, req.TimeSpent, startedAt, req.Note, remaining, actorID(c))
	if err != nil {
		respondWorklogError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "作業ログの更新に成功しました",
		"worklog": worklog,
	})
}

// DeleteWorklog は作業ログ（:id）を削除する
func (h *WorklogHandler) DeleteWorklog(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if err := h.service.DeleteWorklog(id, actorID(c)); err != nil {
		respondWorklogError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "作業ログの削除に成功しました",
	})
}

// bindWorklogRequest はリクエストを読む。不正なら 400 を返して false を返す
func bindWorklogRequest(c *gin.Context) (time.Time, service.DurationUpdate, worklogRequest, bool) {
	var req worklogRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return time.Time{}, service.DurationUpdate{}, req, false
	}
	startedAt, err := time.Parse(time.RFC3339, req.StartedAt)
	if err != nil {
		respondWorklogError(c, service.ErrInvalidWorklogStart)
		return time.Time{}, service.DurationUpdate{}, req, false
	}
	remaining, err := parseDurationUpdate(req.RemainingEstimate)
	if err != nil {
		respondWorklogError(c, err)
		return time.Time{}, service.DurationUpdate{}, req, false
	}
	return startedAt, remaining, req, true
}

// respondWorklogError は作業ログの操作のエラーを、種類に合ったステータスで返す
func respondWorklogError(c *gin.Context, err error) {
	status := http.StatusInternalServerError
	switch {
	case errors.Is(err, service.ErrIssueNotFound), errors.Is(err, service.ErrWorklogNotFound):
		status = http.StatusNotFound
	case errors.Is(err, service.ErrInvalidWorklogTime),
		errors.Is(err, service.ErrInvalidWorklogStart),
		errors.Is(err, domain.ErrInvalidWorklogNote),
		errors.Is(err, domain.ErrInvalidDuration):
		status = http.StatusBadRequest
	}
	c.JSON(status, gin.H{
		"error": err.Error(),
	})
}
