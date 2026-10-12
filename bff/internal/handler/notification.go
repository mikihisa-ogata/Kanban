package handler

import (
	"errors"
	"net/http"

	"todo-api/internal/domain"
	"todo-api/internal/service"

	"github.com/gin-gonic/gin"
)

type NotificationHandler struct {
	service service.NotificationService
}

func NewNotificationHandler(s service.NotificationService) *NotificationHandler {
	return &NotificationHandler{service: s}
}

// GetNotifications はログイン中のユーザーへの通知（新しい順に最大 50 件）と未読の数を返す
func (h *NotificationHandler) GetNotifications(c *gin.Context) {
	notifications, err := h.service.GetNotifications(actorID(c))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, notifications)
}

// MarkRead は通知（:id）を既読にする。ログイン中のユーザーへの通知でなければ 404 を返す
func (h *NotificationHandler) MarkRead(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	err := h.service.MarkRead(id, actorID(c))
	if errors.Is(err, service.ErrNotificationNotFound) {
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
		"message": "通知を既読にしました",
	})
}

// MarkAllRead はログイン中のユーザーへの通知をすべて既読にする
func (h *NotificationHandler) MarkAllRead(c *gin.Context) {
	if err := h.service.MarkAllRead(actorID(c)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "すべての通知を既読にしました",
	})
}

type WatcherHandler struct {
	service service.WatcherService
}

func NewWatcherHandler(s service.WatcherService) *WatcherHandler {
	return &WatcherHandler{service: s}
}

// GetWatchers は課題（:id）のウォッチャーを表示名の順に返す
func (h *WatcherHandler) GetWatchers(c *gin.Context) {
	todoID, ok := parseID(c)
	if !ok {
		return
	}
	watchers, err := h.service.GetWatchers(todoID)
	respondWatchers(c, watchers, err)
}

// Watch はログイン中のユーザーを課題（:id）のウォッチャーにし、ウォッチャーを返す
func (h *WatcherHandler) Watch(c *gin.Context) {
	todoID, ok := parseID(c)
	if !ok {
		return
	}
	watchers, err := h.service.Watch(todoID, actorID(c))
	respondWatchers(c, watchers, err)
}

// Unwatch はログイン中のユーザーを課題（:id）のウォッチャーから外し、ウォッチャーを返す
func (h *WatcherHandler) Unwatch(c *gin.Context) {
	todoID, ok := parseID(c)
	if !ok {
		return
	}
	watchers, err := h.service.Unwatch(todoID, actorID(c))
	respondWatchers(c, watchers, err)
}

// respondWatchers はウォッチャーを返す。課題がなければ 404 を返す
func respondWatchers(c *gin.Context, watchers []domain.UserSummary, err error) {
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

	c.JSON(http.StatusOK, watchers)
}
