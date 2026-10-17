package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"todo-api/internal/domain"
	"todo-api/internal/service"
)

type ComponentHandler struct {
	service service.ComponentService
}

func NewComponentHandler(s service.ComponentService) *ComponentHandler {
	return &ComponentHandler{service: s}
}

// componentFields はコンポーネントの作成・編集で送る項目。defaultAssigneeId の 0 か省略は既定の担当者なし
type componentFields struct {
	Name              string `json:"name"`
	Description       string `json:"description"`
	DefaultAssigneeID int    `json:"defaultAssigneeId"`
}

// GetComponents はすべてのコンポーネントを、スペース・名前順に返す
func (h *ComponentHandler) GetComponents(c *gin.Context) {
	components, err := h.service.GetComponents()
	if err != nil {
		respondComponentError(c, err)
		return
	}
	c.JSON(http.StatusOK, components)
}

// CreateComponent はスペース（:id）にコンポーネントを作る
func (h *ComponentHandler) CreateComponent(c *gin.Context) {
	spaceID, ok := parseID(c)
	if !ok {
		return
	}
	var req componentFields
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}
	component, err := h.service.CreateComponent(spaceID, req.Name, req.Description, req.DefaultAssigneeID)
	if err != nil {
		respondComponentError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"message":   "コンポーネントの作成に成功しました",
		"component": component,
	})
}

// UpdateComponent はコンポーネント（:id）の名前・説明・既定の担当者を変える
func (h *ComponentHandler) UpdateComponent(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var req componentFields
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}
	component, err := h.service.UpdateComponent(id, req.Name, req.Description, req.DefaultAssigneeID)
	if err != nil {
		respondComponentError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message":   "コンポーネントの更新に成功しました",
		"component": component,
	})
}

// DeleteComponent はコンポーネント（:id）を削除する。課題からも外れる
func (h *ComponentHandler) DeleteComponent(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if err := h.service.DeleteComponent(id); err != nil {
		respondComponentError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message": "コンポーネントの削除に成功しました",
	})
}

// SetTodoComponents は課題（:id）のコンポーネントを componentIds に置き換え、変更後の課題を返す
// （担当者が空なら、コンポーネントの既定の担当者が入る）
func (h *ComponentHandler) SetTodoComponents(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var req struct {
		ComponentIDs []int `json:"componentIds"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}
	todo, err := h.service.SetTodoComponents(id, req.ComponentIDs, actorID(c))
	if err != nil {
		respondComponentError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message": "コンポーネントの変更に成功しました",
		"todo":    todo,
	})
}

// respondComponentError はコンポーネントの操作のエラーを、種類に合ったステータスで返す
func respondComponentError(c *gin.Context, err error) {
	status := http.StatusInternalServerError
	switch {
	case errors.Is(err, service.ErrComponentNotFound), errors.Is(err, service.ErrSpaceNotFound), errors.Is(err, service.ErrIssueNotFound):
		status = http.StatusNotFound
	case errors.Is(err, service.ErrComponentNameDuplicate):
		status = http.StatusConflict
	case errors.Is(err, domain.ErrInvalidComponentName),
		errors.Is(err, domain.ErrInvalidComponentDescription),
		errors.Is(err, service.ErrUnassignedComponent),
		errors.Is(err, service.ErrComponentOtherSpace),
		errors.Is(err, service.ErrUserNotFound):
		status = http.StatusBadRequest
	}
	respondError(c, status, err)
}
