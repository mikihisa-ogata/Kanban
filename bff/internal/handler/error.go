package handler

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

// internalErrorMessage は 500 のときに返す文言。DB などの内部のエラー文は返さない
const internalErrorMessage = "サーバーでエラーが発生しました。時間をおいてやり直してください"

// respondError はエラーを status で返す。500 のときは、テーブル名や SQL など内部の構成が分かる
// エラー文を返さず、決まった文言を返してエラーの中身はログにだけ出す
func respondError(c *gin.Context, status int, err error) {
	if status == http.StatusInternalServerError {
		log.Printf("%s %s: %v", c.Request.Method, c.Request.URL.Path, err)
		c.JSON(status, gin.H{
			"error": internalErrorMessage,
		})
		return
	}
	c.JSON(status, gin.H{
		"error": err.Error(),
	})
}
