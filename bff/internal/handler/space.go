package handler

import (
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
	}

	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	err := h.service.CreateSpace(req.Title)
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
