package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"todo-api/internal/domain"
	"todo-api/internal/service"
)

var errInvalidBulkID = errors.New("スプリント・エピックは ID か null で指定してください")

type BulkHandler struct {
	service service.BulkService
}

func NewBulkHandler(s service.BulkService) *BulkHandler {
	return &BulkHandler{service: s}
}

// UpdateIssues は ids の課題をまとめて変更し、内容が変わった課題の数を count で返す。
// 指定しなかった項目は今のまま。assigneeId・sprintId・epicId は null か 0 で「なし」（スプリントはバックログ）にする
func (h *BulkHandler) UpdateIssues(c *gin.Context) {
	var req struct {
		IDs                 []int           `json:"ids"`
		StatusID            int             `json:"statusId"`
		AssigneeID          json.RawMessage `json:"assigneeId"`
		Priority            string          `json:"priority"`
		AddLabels           []string        `json:"addLabels"`
		RemoveLabels        []string        `json:"removeLabels"`
		SprintID            json.RawMessage `json:"sprintId"`
		EpicID              json.RawMessage `json:"epicId"`
		AddFixVersionIDs    []int           `json:"addFixVersionIds"`
		RemoveFixVersionIDs []int           `json:"removeFixVersionIds"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}
	assignee, err := parseUserUpdate(req.AssigneeID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}
	sprint, err := parseIDUpdate(req.SprintID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}
	epic, err := parseIDUpdate(req.EpicID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	count, err := h.service.UpdateIssues(req.IDs, service.BulkUpdate{
		StatusID:            req.StatusID,
		Assignee:            assignee,
		Priority:            req.Priority,
		AddLabels:           req.AddLabels,
		RemoveLabels:        req.RemoveLabels,
		Sprint:              sprint,
		Epic:                epic,
		AddFixVersionIDs:    req.AddFixVersionIDs,
		RemoveFixVersionIDs: req.RemoveFixVersionIDs,
	}, actorID(c))
	if err != nil {
		respondBulkError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message": "課題の一括変更に成功しました",
		"count":   count,
	})
}

// DeleteIssues は ids の課題（とそのサブタスク）をまとめて削除し、削除した課題の数を count で返す
func (h *BulkHandler) DeleteIssues(c *gin.Context) {
	var req struct {
		IDs []int `json:"ids"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}
	count, err := h.service.DeleteIssues(req.IDs)
	if err != nil {
		respondBulkError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message": "課題の一括削除に成功しました",
		"count":   count,
	})
}

// parseIDUpdate はスプリント・エピックの指定を読む。省略は「今のまま」、null か 0 は「なし」、正の整数はその ID
func parseIDUpdate(raw json.RawMessage) (service.IDUpdate, error) {
	if len(raw) == 0 {
		return service.IDUpdate{}, nil
	}
	var id *int
	if err := json.Unmarshal(raw, &id); err != nil || (id != nil && *id < 0) {
		return service.IDUpdate{}, errInvalidBulkID
	}
	if id == nil {
		return service.IDUpdate{Set: true}, nil
	}
	return service.IDUpdate{Set: true, ID: *id}, nil
}

// respondBulkError は一括変更・一括削除のエラーを、種類に合ったステータスで返す
func respondBulkError(c *gin.Context, err error) {
	status := http.StatusInternalServerError
	switch {
	case errors.Is(err, service.ErrIssueNotFound):
		status = http.StatusNotFound
	case errors.Is(err, service.ErrBulkNoIssues),
		errors.Is(err, service.ErrBulkNoChanges),
		errors.Is(err, service.ErrStatusNotFound),
		errors.Is(err, service.ErrInvalidPriority),
		errors.Is(err, service.ErrUserNotFound),
		errors.Is(err, domain.ErrInvalidLabel),
		errors.Is(err, service.ErrEpicNotFound),
		errors.Is(err, service.ErrEpicSpaceMismatch),
		errors.Is(err, service.ErrSprintNotFound),
		errors.Is(err, service.ErrSprintClosed),
		errors.Is(err, service.ErrSprintOtherSpace),
		errors.Is(err, service.ErrVersionNotFound),
		errors.Is(err, service.ErrVersionOtherSpace),
		errors.Is(err, service.ErrVersionArchived):
		status = http.StatusBadRequest
	}
	respondError(c, status, err)
}
