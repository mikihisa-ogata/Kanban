package handler

import (
	"errors"
	"net/http"
	"strconv"

	"todo-api/internal/domain"
	"todo-api/internal/service"

	"github.com/gin-gonic/gin"
)

type SpaceHandler struct {
	service service.SpaceService
}

func NewSpaceHandler(s service.SpaceService) *SpaceHandler {
	return &SpaceHandler{service: s}
}

func (h *SpaceHandler) GetSpaces(c *gin.Context) {
	spaces, err := h.service.GetSpaces()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, spaces)
}

// GetStatuses はすべてのワークフローのステータスを返す（SpaceID 0 はスペース未割り当ての課題のワークフロー）
func (h *SpaceHandler) GetStatuses(c *gin.Context) {
	statuses, err := h.service.GetStatuses()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, statuses)
}

func (h *SpaceHandler) CreateSpace(c *gin.Context) {
	var req struct {
		Title string `json:"title" binding:"required"`
		Key   string `json:"key"`
	}

	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	err := h.service.CreateSpace(req.Title, req.Key)
	if errors.Is(err, service.ErrInvalidSpaceKey) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}
	if errors.Is(err, service.ErrSpaceKeyDuplicate) {
		c.JSON(http.StatusConflict, gin.H{
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
		"message": "スペースの作成に成功しました",
	})
}

// UpdateSpace はスペースの名前とキーを変える
func (h *SpaceHandler) UpdateSpace(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "不正なIDです",
		})
		return
	}

	var req struct {
		Title string `json:"title" binding:"required"`
		Key   string `json:"key" binding:"required"`
	}

	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	err = h.service.UpdateSpace(id, req.Title, req.Key)
	if errors.Is(err, service.ErrSpaceNotFound) {
		c.JSON(http.StatusNotFound, gin.H{
			"error": err.Error(),
		})
		return
	}
	if errors.Is(err, service.ErrInvalidSpaceKey) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}
	if errors.Is(err, service.ErrSpaceKeyDuplicate) {
		c.JSON(http.StatusConflict, gin.H{
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
		"message": "スペースの更新に成功しました",
	})
}

// SuggestSpaceKey はスペース名（クエリ title）から、既存と重複しないキーの候補を返す
func (h *SpaceHandler) SuggestSpaceKey(c *gin.Context) {
	key, err := h.service.SuggestSpaceKey(c.Query("title"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"key": key,
	})
}

func (h *SpaceHandler) DeleteSpace(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "不正なIDです",
		})
		return
	}

	err = h.service.DeleteSpace(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "スペースの削除に成功しました",
	})
}

// CreateStatus はスペース（:id）のワークフローの末尾にステータスを追加する
func (h *SpaceHandler) CreateStatus(c *gin.Context) {
	spaceID, ok := parseID(c)
	if !ok {
		return
	}
	var req struct {
		Name     string `json:"name"`
		Category string `json:"category"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}
	status, err := h.service.CreateStatus(spaceID, req.Name, req.Category)
	if err != nil {
		respondWorkflowError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "ステータスの追加に成功しました",
		"status":  status,
	})
}

// RenameStatus はステータス（:id）の名前を変える
func (h *SpaceHandler) RenameStatus(c *gin.Context) {
	id, ok := parseID(c)
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
	status, err := h.service.RenameStatus(id, req.Name)
	if err != nil {
		respondWorkflowError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "ステータスの名前の変更に成功しました",
		"status":  status,
	})
}

// ReorderStatuses はスペース（:id）のステータスを statusIds の順に並べ直す
func (h *SpaceHandler) ReorderStatuses(c *gin.Context) {
	spaceID, ok := parseID(c)
	if !ok {
		return
	}
	var req struct {
		StatusIDs []int `json:"statusIds"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}
	if err := h.service.ReorderStatuses(spaceID, req.StatusIDs); err != nil {
		respondWorkflowError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "ステータスの並べ替えに成功しました",
	})
}

// DeleteStatus はステータス（:id）を削除する。そのステータスの課題は ?moveTo=<ステータスの ID> へ移す
func (h *SpaceHandler) DeleteStatus(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	moveTo := 0
	if v := c.Query("moveTo"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			respondWorkflowError(c, service.ErrInvalidMoveTarget)
			return
		}
		moveTo = n
	}
	if err := h.service.DeleteStatus(id, moveTo); err != nil {
		respondWorkflowError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "ステータスの削除に成功しました",
	})
}

// respondWorkflowError はワークフローの編集のエラーを、種類に合ったステータスで返す
func respondWorkflowError(c *gin.Context, err error) {
	status := http.StatusInternalServerError
	switch {
	case errors.Is(err, service.ErrSpaceNotFound), errors.Is(err, service.ErrStatusNotFound):
		status = http.StatusNotFound
	case errors.Is(err, service.ErrStatusNameDuplicate), errors.Is(err, service.ErrStatusInUse):
		status = http.StatusConflict
	case errors.Is(err, domain.ErrInvalidStatusName),
		errors.Is(err, service.ErrInvalidStatusCategory),
		errors.Is(err, service.ErrUnassignedWorkflow),
		errors.Is(err, service.ErrInvalidMoveTarget),
		errors.Is(err, service.ErrLastStatus),
		errors.Is(err, service.ErrInvalidStatusOrder):
		status = http.StatusBadRequest
	}
	c.JSON(status, gin.H{
		"error": err.Error(),
	})
}
