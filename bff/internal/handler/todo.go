package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"todo-api/internal/domain"
	"todo-api/internal/service"

	"github.com/gin-gonic/gin"
)

type TodoHandler struct {
	service service.TodoService
}

func NewTodoHandler(s service.TodoService) *TodoHandler {
	return &TodoHandler{service: s}
}

func (h *TodoHandler) GetTodos(c *gin.Context) {
	todos, err := h.service.GetTodos()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, todos)
}

// GetIssue は課題キー（例: KAN-12）のタスクを返す。見つからなければ 404 を返す
func (h *TodoHandler) GetIssue(c *gin.Context) {
	todo, err := h.service.GetTodoByKey(c.Param("key"))
	if errors.Is(err, service.ErrIssueNotFound) {
		c.JSON(http.StatusNotFound, gin.H{
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

	c.JSON(http.StatusOK, todo)
}

// GetLabels はいずれかのタスクに付いているラベルを名前順に返す
func (h *TodoHandler) GetLabels(c *gin.Context) {
	labels, err := h.service.GetLabels()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, labels)
}

func (h *TodoHandler) CreateTodo(c *gin.Context) {
	var req struct {
		Title       string   `json:"title" binding:"required"`
		Deadline    string   `json:"deadline"`
		Status      string   `json:"status"`
		IssueType   string   `json:"issueType"`
		Priority    string   `json:"priority"`
		Labels      []string `json:"labels"`
		StoryPoints *float64 `json:"storyPoints"`
		ParentID    int      `json:"parentId"`
		EpicID      int      `json:"epicId"`
		Description string   `json:"description"`
		SpaceID     int      `json:"spaceId"`
	}

	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	todo, err := h.service.CreateTodo(req.Title, req.Deadline, req.Status, req.IssueType, req.Priority, req.Labels, req.StoryPoints, req.ParentID, req.EpicID, req.Description, req.SpaceID)
	if isTodoValidationError(err) {
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

	c.JSON(http.StatusCreated, gin.H{
		"message": "Todoの作成に成功しました",
		"todo":    todo,
	})
}

func (h *TodoHandler) UpdateTodo(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "不正なIDです",
		})
		return
	}

	var req struct {
		Title     string   `json:"title" binding:"required"`
		Done      bool     `json:"done"`
		Deadline  string   `json:"deadline"`
		Status    string   `json:"status" binding:"required"`
		IssueType string   `json:"issueType"`
		Priority  string   `json:"priority"`
		Labels    []string `json:"labels"`
		// 省略は「今のまま」、null は「未設定にする」なので、受け取った JSON のまま読む
		StoryPoints json.RawMessage `json:"storyPoints"`
		// 初期見積もり・残り時間は「1w 2d 3h 30m」形式。省略は今のまま、null か空文字は未設定にする
		OriginalEstimate  json.RawMessage `json:"originalEstimate"`
		RemainingEstimate json.RawMessage `json:"remainingEstimate"`
		EpicID            int             `json:"epicId"`
		Description       string          `json:"description"`
		SpaceID           int             `json:"spaceId"`
	}

	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	storyPoints, err := parseStoryPointsUpdate(req.StoryPoints)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	original, err := parseDurationUpdate(req.OriginalEstimate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}
	remaining, err := parseDurationUpdate(req.RemainingEstimate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	todo, err := h.service.UpdateTodo(id, req.Title, req.Done, req.Deadline, req.Status, req.IssueType, req.Priority, req.Labels, storyPoints,
		service.EstimatesUpdate{Original: original, Remaining: remaining}, req.EpicID, req.Description, req.SpaceID)
	if isTodoValidationError(err) {
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

	c.JSON(http.StatusOK, gin.H{
		"message": "Todoの更新に成功しました",
		"todo":    todo,
	})
}

func (h *TodoHandler) DeleteTodo(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "不正なIDです",
		})
		return
	}

	err = h.service.DeleteTodo(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Todoの削除に成功しました",
	})
}

// MoveTodo はタスクの並び順を変更する。beforeId のタスクの直前へ移動し、0 または省略時は末尾へ移動する
func (h *TodoHandler) MoveTodo(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "不正なIDです",
		})
		return
	}

	var req struct {
		BeforeID int `json:"beforeId"`
	}

	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	err = h.service.MoveTodo(id, req.BeforeID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Todoの並び替えに成功しました",
	})
}

// AddLink は課題リンクを作成する。relation は :id の課題から見たリンクの名前、targetId はリンク相手の課題の ID
func (h *TodoHandler) AddLink(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "不正なIDです",
		})
		return
	}

	var req struct {
		Relation string `json:"relation" binding:"required"`
		TargetID int    `json:"targetId" binding:"required"`
	}

	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	todo, err := h.service.AddLink(id, req.Relation, req.TargetID)
	switch {
	case errors.Is(err, service.ErrIssueNotFound):
		c.JSON(http.StatusNotFound, gin.H{
			"error": err.Error(),
		})
		return
	case errors.Is(err, service.ErrLinkDuplicate):
		c.JSON(http.StatusConflict, gin.H{
			"error": err.Error(),
		})
		return
	case errors.Is(err, service.ErrInvalidLinkRelation),
		errors.Is(err, service.ErrLinkTargetNotFound),
		errors.Is(err, service.ErrSelfLink):
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	case err != nil:
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "課題リンクの作成に成功しました",
		"todo":    todo,
	})
}

// DeleteLink は課題リンクを削除する。見つからなければ 404 を返す
func (h *TodoHandler) DeleteLink(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "不正なIDです",
		})
		return
	}

	err = h.service.DeleteLink(id)
	if errors.Is(err, service.ErrLinkNotFound) {
		c.JSON(http.StatusNotFound, gin.H{
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

	c.JSON(http.StatusOK, gin.H{
		"message": "課題リンクの削除に成功しました",
	})
}

// GetHistory は課題（:id）の変更履歴を記録順（古い順）に返す。課題がなければ 404 を返す
func (h *TodoHandler) GetHistory(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	entries, err := h.service.GetHistory(id)
	if errors.Is(err, service.ErrIssueNotFound) {
		c.JSON(http.StatusNotFound, gin.H{
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

	c.JSON(http.StatusOK, entries)
}

// parseDurationUpdate は更新リクエストの時間（初期見積もり・残り時間）を読む。省略は今のまま、null は未設定、文字列はその値
func parseDurationUpdate(raw json.RawMessage) (service.DurationUpdate, error) {
	if len(raw) == 0 {
		return service.DurationUpdate{}, nil
	}
	var value *string
	if err := json.Unmarshal(raw, &value); err != nil {
		return service.DurationUpdate{}, domain.ErrInvalidDuration
	}
	if value == nil {
		return service.DurationUpdate{Set: true}, nil
	}
	return service.DurationUpdate{Set: true, Value: *value}, nil
}

// parseStoryPointsUpdate は更新リクエストの storyPoints を読む。省略は今のまま、null は未設定、数値はその値
func parseStoryPointsUpdate(raw json.RawMessage) (service.StoryPointsUpdate, error) {
	if len(raw) == 0 {
		return service.StoryPointsUpdate{}, nil
	}
	var value *float64
	if err := json.Unmarshal(raw, &value); err != nil {
		return service.StoryPointsUpdate{}, service.ErrInvalidStoryPoints
	}
	return service.StoryPointsUpdate{Set: true, Value: value}, nil
}

// isTodoValidationError はリクエストの内容が不正なことによるエラーかを返す
func isTodoValidationError(err error) bool {
	return errors.Is(err, service.ErrEpicNotFound) ||
		errors.Is(err, service.ErrSpaceNotFound) ||
		errors.Is(err, service.ErrEpicSpaceMismatch) ||
		errors.Is(err, service.ErrInvalidIssueType) ||
		errors.Is(err, service.ErrInvalidPriority) ||
		errors.Is(err, domain.ErrInvalidLabel) ||
		errors.Is(err, service.ErrInvalidStoryPoints) ||
		errors.Is(err, service.ErrParentNotFound) ||
		errors.Is(err, service.ErrSubtaskParent) ||
		errors.Is(err, service.ErrSubtaskType) ||
		errors.Is(err, domain.ErrInvalidDuration)
}
