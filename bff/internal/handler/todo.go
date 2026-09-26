package handler

import (
	"errors"
	"net/http"
	"strconv"

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

func (h *TodoHandler) CreateTodo(c *gin.Context) {
	var req struct {
		Title       string `json:"title" binding:"required"`
		Deadline    string `json:"deadline"`
		Status      string `json:"status"`
		EpicID      int    `json:"epicId"`
		Description string `json:"description"`
		SpaceID     int    `json:"spaceId"`
	}

	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	err := h.service.CreateTodo(req.Title, req.Deadline, req.Status, req.EpicID, req.Description, req.SpaceID)
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
		Title       string `json:"title" binding:"required"`
		Done        bool   `json:"done"`
		Deadline    string `json:"deadline"`
		Status      string `json:"status" binding:"required"`
		EpicID      int    `json:"epicId"`
		Description string `json:"description"`
		SpaceID     int    `json:"spaceId"`
	}

	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	err = h.service.UpdateTodo(id, req.Title, req.Done, req.Deadline, req.Status, req.EpicID, req.Description, req.SpaceID)
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

// isTodoValidationError はリクエストの内容が不正なことによるエラーかを返す
func isTodoValidationError(err error) bool {
	return errors.Is(err, service.ErrEpicNotFound) ||
		errors.Is(err, service.ErrSpaceNotFound) ||
		errors.Is(err, service.ErrEpicSpaceMismatch)
}
