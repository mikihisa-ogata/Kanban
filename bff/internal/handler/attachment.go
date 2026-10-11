package handler

import (
	"errors"
	"mime"
	"net/http"

	"todo-api/internal/domain"
	"todo-api/internal/service"

	"github.com/gin-gonic/gin"
)

// attachmentRequestMax はアップロードのリクエスト全体の上限。ファイルの上限に multipart の区切りなどの分を足す
const attachmentRequestMax = domain.AttachmentMaxSize + 1<<20

type AttachmentHandler struct {
	service service.AttachmentService
}

func NewAttachmentHandler(s service.AttachmentService) *AttachmentHandler {
	return &AttachmentHandler{service: s}
}

// GetAttachments は課題（:id）の添付ファイルの情報を添付順に返す
func (h *AttachmentHandler) GetAttachments(c *gin.Context) {
	todoID, ok := parseID(c)
	if !ok {
		return
	}
	attachments, err := h.service.GetAttachments(todoID)
	if err != nil {
		respondAttachmentError(c, err)
		return
	}

	c.JSON(http.StatusOK, attachments)
}

// AddAttachment は課題（:id）にファイルを添付する。multipart/form-data の file フィールドで受け取る
func (h *AttachmentHandler) AddAttachment(c *gin.Context) {
	todoID, ok := parseID(c)
	if !ok {
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, attachmentRequestMax)
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			respondAttachmentError(c, service.ErrAttachmentTooLarge)
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "file フィールドでファイルを送ってください",
		})
		return
	}
	defer file.Close()

	attachment, err := h.service.AddAttachment(todoID, header.Filename, header.Size, file)
	if err != nil {
		respondAttachmentError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":    "ファイルの添付に成功しました",
		"attachment": attachment,
	})
}

// DownloadAttachment は添付ファイル（:id）の本体を返す。
// ?inline=1 のときは、画像（SVG を除く）に限りブラウザで表示できるようにする。それ以外はダウンロードさせる
func (h *AttachmentHandler) DownloadAttachment(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	attachment, file, err := h.service.OpenAttachment(id)
	if err != nil {
		respondAttachmentError(c, err)
		return
	}
	defer file.Close()

	disposition := "attachment"
	if c.Query("inline") == "1" && attachment.IsInlineImage() {
		disposition = "inline"
	}
	c.DataFromReader(http.StatusOK, attachment.Size, attachment.ContentType, file, map[string]string{
		// 日本語のファイル名も崩れないよう、filename*（UTF-8）で送る
		"Content-Disposition":    mime.FormatMediaType(disposition, map[string]string{"filename": attachment.FileName}),
		"X-Content-Type-Options": "nosniff",
	})
}

// DeleteAttachment は添付ファイル（:id）を削除する
func (h *AttachmentHandler) DeleteAttachment(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if err := h.service.DeleteAttachment(id); err != nil {
		respondAttachmentError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "添付ファイルの削除に成功しました",
	})
}

// respondAttachmentError は添付ファイルの操作のエラーを、種類に合ったステータスで返す
func respondAttachmentError(c *gin.Context, err error) {
	status := http.StatusInternalServerError
	switch {
	case errors.Is(err, service.ErrIssueNotFound), errors.Is(err, service.ErrAttachmentNotFound):
		status = http.StatusNotFound
	case errors.Is(err, service.ErrAttachmentTooLarge):
		status = http.StatusRequestEntityTooLarge
	case errors.Is(err, service.ErrAttachmentEmpty):
		status = http.StatusBadRequest
	}
	c.JSON(status, gin.H{
		"error": err.Error(),
	})
}
