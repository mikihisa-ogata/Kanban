package handler

import (
	"errors"
	"net/http"
	"strconv"

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
