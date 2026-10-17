package handler

import (
	"errors"
	"net/http"
	"strconv"

	"todo-api/internal/domain"
	"todo-api/internal/service"

	"github.com/gin-gonic/gin"
)

type CommentHandler struct {
	service service.CommentService
}

func NewCommentHandler(s service.CommentService) *CommentHandler {
	return &CommentHandler{service: s}
}

// GetComments は課題（:id）のコメントを投稿順（古い順）に返す
func (h *CommentHandler) GetComments(c *gin.Context) {
	todoID, ok := parseID(c)
	if !ok {
		return
	}
	comments, err := h.service.GetComments(todoID)
	if err != nil {
		respondCommentError(c, err)
		return
	}

	c.JSON(http.StatusOK, comments)
}

// CreateComment は課題（:id）にコメントを投稿する
func (h *CommentHandler) CreateComment(c *gin.Context) {
	todoID, ok := parseID(c)
	if !ok {
		return
	}
	var req struct {
		Body string `json:"body"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	comment, err := h.service.CreateComment(todoID, req.Body, actorID(c))
	if err != nil {
		respondCommentError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "コメントの投稿に成功しました",
		"comment": comment,
	})
}

// UpdateComment はコメント（:id）の本文を変える
func (h *CommentHandler) UpdateComment(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var req struct {
		Body string `json:"body"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	comment, err := h.service.UpdateComment(id, req.Body, actorID(c))
	if err != nil {
		respondCommentError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "コメントの更新に成功しました",
		"comment": comment,
	})
}

// DeleteComment はコメント（:id）を削除する
func (h *CommentHandler) DeleteComment(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if err := h.service.DeleteComment(id, actorID(c)); err != nil {
		respondCommentError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "コメントの削除に成功しました",
	})
}

// parseID はパスの :id を読む。数値でなければ 400 を返して false を返す
func parseID(c *gin.Context) (int, bool) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "不正なIDです",
		})
		return 0, false
	}
	return id, true
}

// respondCommentError はコメントの操作のエラーを、種類に合ったステータスで返す
func respondCommentError(c *gin.Context, err error) {
	status := http.StatusInternalServerError
	switch {
	case errors.Is(err, service.ErrIssueNotFound), errors.Is(err, service.ErrCommentNotFound):
		status = http.StatusNotFound
	case errors.Is(err, domain.ErrInvalidComment):
		status = http.StatusBadRequest
	case errors.Is(err, service.ErrCommentForbidden):
		status = http.StatusForbidden
	}
	c.JSON(status, gin.H{
		"error": err.Error(),
	})
}
