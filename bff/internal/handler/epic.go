package handler

import (
	"errors"
	"net/http"
	"strconv"

	"todo-api/internal/domain"
	"todo-api/internal/service"

	"github.com/gin-gonic/gin"
)

type EpicHandler struct {
	service service.EpicService
}

func NewEpicHandler(s service.EpicService) *EpicHandler {
	return &EpicHandler{service: s}
}

// epicRequest はエピックの作成・更新のリクエスト。startDate・dueDate・color は省略すると、作成では未設定、更新では今の値のまま
type epicRequest struct {
	Title     string  `json:"title" binding:"required"`
	SpaceID   int     `json:"spaceId"`
	StartDate *string `json:"startDate"`
	DueDate   *string `json:"dueDate"`
	Color     *string `json:"color"`
}

func (r epicRequest) input() service.EpicInput {
	return service.EpicInput{Title: r.Title, SpaceID: r.SpaceID, StartDate: r.StartDate, DueDate: r.DueDate, Color: r.Color}
}

// isEpicInputError はエピックの日付・色の誤り（400 にするもの）かを返す
func isEpicInputError(err error) bool {
	return errors.Is(err, domain.ErrInvalidEpicDates) || errors.Is(err, domain.ErrInvalidEpicColor)
}

func (h *EpicHandler) GetEpics(c *gin.Context) {
	epics, err := h.service.GetEpics()
	if err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, epics)
}

func (h *EpicHandler) CreateEpic(c *gin.Context) {
	var req epicRequest

	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	err := h.service.CreateEpic(req.input())
	if errors.Is(err, service.ErrSpaceNotFound) || isEpicInputError(err) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}
	if err != nil {
		respondError(c, http.StatusInternalServerError, err)
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

	var req epicRequest

	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	err = h.service.UpdateEpic(id, req.input(), actorID(c))
	if errors.Is(err, service.ErrEpicNotFound) {
		c.JSON(http.StatusNotFound, gin.H{
			"error": err.Error(),
		})
		return
	}
	if errors.Is(err, service.ErrSpaceNotFound) || isEpicInputError(err) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}
	if err != nil {
		respondError(c, http.StatusInternalServerError, err)
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

	err = h.service.DeleteEpic(id, actorID(c))
	if err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "エピックの削除に成功しました",
	})
}
