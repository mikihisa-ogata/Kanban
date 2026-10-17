package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"todo-api/internal/domain"
	"todo-api/internal/service"
)

type VersionHandler struct {
	service service.VersionService
}

func NewVersionHandler(s service.VersionService) *VersionHandler {
	return &VersionHandler{service: s}
}

// versionFields はバージョンの作成・編集で送る項目
type versionFields struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	StartDate   string `json:"startDate"`
	ReleaseDate string `json:"releaseDate"`
	Status      string `json:"status"`
}

// GetVersions はすべてのバージョンを、スペース・作成順に返す
func (h *VersionHandler) GetVersions(c *gin.Context) {
	versions, err := h.service.GetVersions()
	if err != nil {
		respondVersionError(c, err)
		return
	}
	c.JSON(http.StatusOK, versions)
}

// CreateVersion はスペース（:id）に未リリースのバージョンを作る
func (h *VersionHandler) CreateVersion(c *gin.Context) {
	spaceID, ok := parseID(c)
	if !ok {
		return
	}
	var req versionFields
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}
	version, err := h.service.CreateVersion(spaceID, req.Name, req.Description, req.StartDate, req.ReleaseDate)
	if err != nil {
		respondVersionError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"message": "バージョンの作成に成功しました",
		"version": version,
	})
}

// UpdateVersion はバージョン（:id）の名前・説明・日付・状態を変える
func (h *VersionHandler) UpdateVersion(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var req versionFields
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}
	version, err := h.service.UpdateVersion(id, req.Name, req.Description, req.StartDate, req.ReleaseDate, req.Status)
	if err != nil {
		respondVersionError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message": "バージョンの更新に成功しました",
		"version": version,
	})
}

// ReleaseVersion はバージョン（:id）をリリース済みにする。完了していない課題は moveTo のバージョンへ移す（0 か省略なら移さない）
func (h *VersionHandler) ReleaseVersion(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var req struct {
		ReleaseDate string `json:"releaseDate"`
		MoveTo      int    `json:"moveTo"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}
	version, err := h.service.ReleaseVersion(id, req.ReleaseDate, req.MoveTo)
	if err != nil {
		respondVersionError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message": "バージョンをリリースしました",
		"version": version,
	})
}

// DeleteVersion はバージョン（:id）を削除する。課題からも外れる
func (h *VersionHandler) DeleteVersion(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if err := h.service.DeleteVersion(id); err != nil {
		respondVersionError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message": "バージョンの削除に成功しました",
	})
}

// SetTodoVersions は課題（:id）の修正バージョンを versionIds に置き換える
func (h *VersionHandler) SetTodoVersions(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var req struct {
		VersionIDs []int `json:"versionIds"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}
	if err := h.service.SetTodoVersions(id, req.VersionIDs); err != nil {
		respondVersionError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message": "修正バージョンの変更に成功しました",
	})
}

// respondVersionError はバージョンの操作のエラーを、種類に合ったステータスで返す
func respondVersionError(c *gin.Context, err error) {
	status := http.StatusInternalServerError
	switch {
	case errors.Is(err, service.ErrVersionNotFound), errors.Is(err, service.ErrSpaceNotFound), errors.Is(err, service.ErrVersionTodoNotFound):
		status = http.StatusNotFound
	case errors.Is(err, service.ErrVersionNameDuplicate):
		status = http.StatusConflict
	case errors.Is(err, domain.ErrInvalidVersionName),
		errors.Is(err, domain.ErrInvalidVersionDescription),
		errors.Is(err, domain.ErrInvalidVersionDates),
		errors.Is(err, domain.ErrInvalidVersionStatus),
		errors.Is(err, service.ErrUnassignedVersion),
		errors.Is(err, service.ErrVersionAlreadyReleased),
		errors.Is(err, service.ErrInvalidVersionMoveTo),
		errors.Is(err, service.ErrVersionOtherSpace),
		errors.Is(err, service.ErrVersionArchived):
		status = http.StatusBadRequest
	}
	respondError(c, status, err)
}
