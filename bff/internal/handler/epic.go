package handler

import (
	"errors"
	"net/http"
	"strconv"

	"todo-api/internal/service"

	"github.com/gin-gonic/gin"
)

type EpicHandler struct {
	service service.EpicService
}

func NewEpicHandler(s service.EpicService) *EpicHandler {
	return &EpicHandler{service: s}
}

func (h *EpicHandler) GetEpics(c *gin.Context) {
	epics, err := h.service.GetEpics()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, epics)
}

func (h *EpicHandler) CreateEpic(c *gin.Context) {
	var req struct {
		Title   string `json:"title" binding:"required"`
		SpaceID int    `json:"spaceId"`
	}

	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	err := h.service.CreateEpic(req.Title, req.SpaceID)
	if errors.Is(err, service.ErrSpaceNotFound) {
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
		"message": "エピックの作成に成功しました",
	})
}

func (h *EpicHandler) UpdateEpic(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "不正なIDです",
		})
		return
	}

	var req struct {
		Title   string `json:"title" binding:"required"`
		SpaceID int    `json:"spaceId"`
	}

	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	err = h.service.UpdateEpic(id, req.Title, req.SpaceID)
	if errors.Is(err, service.ErrEpicNotFound) {
		c.JSON(http.StatusNotFound, gin.H{
			"error": err.Error(),
		})
		return
	}
	if errors.Is(err, service.ErrSpaceNotFound) {
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
		"message": "エピックの更新に成功しました",
	})
}

func (h *EpicHandler) DeleteEpic(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "不正なIDです",
		})
		return
	}

	err = h.service.DeleteEpic(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "エピックの削除に成功しました",
	})
}
