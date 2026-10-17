package handler

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"todo-api/internal/domain"
	"todo-api/internal/service"
)

type SprintHandler struct {
	service service.SprintService
}

func NewSprintHandler(s service.SprintService) *SprintHandler {
	return &SprintHandler{service: s}
}

// sprintFields はスプリントの開始・編集で送る名前・ゴール・期間
type sprintFields struct {
	Name      string `json:"name"`
	Goal      string `json:"goal"`
	StartDate string `json:"startDate"`
	EndDate   string `json:"endDate"`
}

// GetSprints はすべてのスプリントを、スペース・作成順に返す
func (h *SprintHandler) GetSprints(c *gin.Context) {
	sprints, err := h.service.GetSprints()
	if err != nil {
		respondSprintError(c, err)
		return
	}
	c.JSON(http.StatusOK, sprints)
}

// CreateSprint はスペース（:id）に未開始のスプリントを作る。name を省略すると「<キー> スプリント <番号>」
func (h *SprintHandler) CreateSprint(c *gin.Context) {
	spaceID, ok := parseID(c)
	if !ok {
		return
	}
	var req struct {
		Name string `json:"name"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}
	sprint, err := h.service.CreateSprint(spaceID, req.Name)
	if err != nil {
		respondSprintError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"message": "スプリントの作成に成功しました",
		"sprint":  sprint,
	})
}

// UpdateSprint はスプリント（:id）の名前・ゴール・期間を変える
func (h *SprintHandler) UpdateSprint(c *gin.Context) {
	h.withFields(c, h.service.UpdateSprint, "スプリントの更新に成功しました")
}

// StartSprint はスプリント（:id）を、名前・ゴール・期間を決めて開始する
func (h *SprintHandler) StartSprint(c *gin.Context) {
	h.withFields(c, h.service.StartSprint, "スプリントを開始しました")
}

func (h *SprintHandler) withFields(c *gin.Context, action func(id int, name, goal, startDate, endDate string) (domain.Sprint, error), message string) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var req sprintFields
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}
	sprint, err := action(id, req.Name, req.Goal, req.StartDate, req.EndDate)
	if err != nil {
		respondSprintError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message": message,
		"sprint":  sprint,
	})
}

// CompleteSprint はスプリント（:id）を完了する。完了していない課題は moveTo（未開始のスプリント。0 か省略はバックログ）へ移す
func (h *SprintHandler) CompleteSprint(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var req struct {
		MoveTo int `json:"moveTo"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}
	sprint, err := h.service.CompleteSprint(id, req.MoveTo, actorID(c))
	if err != nil {
		respondSprintError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message": "スプリントを完了しました",
		"sprint":  sprint,
	})
}

// DeleteSprint は未開始のスプリント（:id）を削除する。その課題はバックログに戻る
func (h *SprintHandler) DeleteSprint(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if err := h.service.DeleteSprint(id, actorID(c)); err != nil {
		respondSprintError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message": "スプリントの削除に成功しました",
	})
}

// GetSprintIssues は完了したときに記録したスプリント（:id）の課題を返す
func (h *SprintHandler) GetSprintIssues(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	issues, err := h.service.GetSprintIssues(id)
	if err != nil {
		respondSprintError(c, err)
		return
	}
	c.JSON(http.StatusOK, issues)
}

// GetBurndown はスプリント（:id）のバーンダウンを返す。日付はクエリ tz（IANA のタイムゾーン名。省略時はサーバーのタイムゾーン）で区切る
func (h *SprintHandler) GetBurndown(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	loc, ok := queryLocation(c)
	if !ok {
		return
	}
	burndown, err := h.service.GetBurndown(id, loc)
	if err != nil {
		respondSprintError(c, err)
		return
	}
	c.JSON(http.StatusOK, burndown)
}

// GetVelocity はスペース（:id）のベロシティ（完了した直近 7 スプリントのコミットメントと完了したポイント）を返す。
// 日付はクエリ tz（省略時はサーバーのタイムゾーン）で読む
func (h *SprintHandler) GetVelocity(c *gin.Context) {
	spaceID, ok := parseID(c)
	if !ok {
		return
	}
	loc, ok := queryLocation(c)
	if !ok {
		return
	}
	velocity, err := h.service.GetVelocity(spaceID, loc)
	if err != nil {
		respondSprintError(c, err)
		return
	}
	c.JSON(http.StatusOK, velocity)
}

// queryLocation はクエリ tz（IANA のタイムゾーン名）を読む。省略時はサーバーのタイムゾーン、正しくなければ 400 を返して false
func queryLocation(c *gin.Context) (*time.Location, bool) {
	tz := c.Query("tz")
	if tz == "" {
		return time.Local, true
	}
	loc, err := time.LoadLocation(tz)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "タイムゾーン（tz）が正しくありません",
		})
		return nil, false
	}
	return loc, true
}

// SetTodoSprint は課題（:id）とそのサブタスクを sprintId のスプリント（0 はバックログ）に入れる
func (h *SprintHandler) SetTodoSprint(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var req struct {
		SprintID int `json:"sprintId"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}
	if err := h.service.SetTodoSprint(id, req.SprintID, actorID(c)); err != nil {
		respondSprintError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message": "スプリントの変更に成功しました",
	})
}

// respondSprintError はスプリントの操作のエラーを、種類に合ったステータスで返す
func respondSprintError(c *gin.Context, err error) {
	status := http.StatusInternalServerError
	switch {
	case errors.Is(err, service.ErrSprintNotFound), errors.Is(err, service.ErrSpaceNotFound), errors.Is(err, service.ErrSprintTodoNotFound):
		status = http.StatusNotFound
	case errors.Is(err, service.ErrSprintAlreadyActive):
		status = http.StatusConflict
	case errors.Is(err, domain.ErrInvalidSprintName),
		errors.Is(err, domain.ErrInvalidSprintGoal),
		errors.Is(err, domain.ErrInvalidSprintDates),
		errors.Is(err, service.ErrUnassignedSprint),
		errors.Is(err, service.ErrSprintNotFuture),
		errors.Is(err, service.ErrSprintNotActive),
		errors.Is(err, service.ErrSprintClosed),
		errors.Is(err, service.ErrSprintNotDeletable),
		errors.Is(err, service.ErrInvalidSprintMoveTo),
		errors.Is(err, service.ErrSprintOtherSpace),
		errors.Is(err, service.ErrSubtaskSprint),
		errors.Is(err, domain.ErrSprintNotStarted):
		status = http.StatusBadRequest
	}
	c.JSON(status, gin.H{
		"error": err.Error(),
	})
}
