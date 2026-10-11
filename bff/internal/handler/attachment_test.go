package handler

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"todo-api/internal/domain"
	"todo-api/internal/service"
)

// pngHeader は PNG として判定される最小の先頭バイト
var pngHeader = []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR")

func newAttachmentRouter(t *testing.T) (*gin.Engine, testRepos) {
	t.Helper()
	repos := newTestRepos(t)
	r := newTodoRouter(repos)
	h := NewAttachmentHandler(service.NewAttachmentService(repos.attachment, repos.todo))
	r.GET("/todos/:id/attachments", h.GetAttachments)
	r.POST("/todos/:id/attachments", h.AddAttachment)
	r.GET("/attachments/:id/file", h.DownloadAttachment)
	r.DELETE("/attachments/:id", h.DeleteAttachment)
	return r, repos
}

// upload は multipart/form-data の file フィールドでファイルを送る
func upload(t *testing.T, r *gin.Engine, path, fileName string, content []byte) *httptest.ResponseRecorder {
	t.Helper()
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	part, err := w.CreateFormFile("file", fileName)
	if err != nil {
		t.Fatal(err)
	}
	part.Write(content)
	w.Close()
	req := httptest.NewRequest(http.MethodPost, path, &body)
	req.Header.Set("Content-Type", w.FormDataContentType())
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec
}

// storedFiles は保存先のディレクトリにあるファイルの数を返す
func storedFiles(t *testing.T, dir string) int {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	return len(entries)
}

func TestAttachmentHandler_UploadDownloadDelete(t *testing.T) {
	r, repos := newAttachmentRouter(t)
	mustStatus(t, r, http.MethodPost, "/todos", `{"title":"a"}`, http.StatusCreated)

	w := upload(t, r, "/todos/1/attachments", "画面.png", append(pngHeader, []byte("rest of image")...))
	if w.Code != http.StatusCreated {
		t.Fatalf("upload status = %d, body = %s", w.Code, w.Body)
	}
	var created struct{ Attachment domain.Attachment }
	if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	a := created.Attachment
	if a.ID == 0 || a.TodoID != 1 || a.FileName != "画面.png" || a.ContentType != "image/png" || a.Size != int64(len(pngHeader)+13) || a.CreatedAt.IsZero() {
		t.Errorf("attachment = %+v", a)
	}
	// 保存名は API に出さない
	if strings.Contains(w.Body.String(), "StorageName") {
		t.Errorf("response exposes StorageName: %s", w.Body)
	}
	if n := storedFiles(t, repos.attachmentsDir); n != 1 {
		t.Errorf("stored files = %d", n)
	}
	upload(t, r, "/todos/1/attachments", "../../memo.txt", []byte("<html><script>alert(1)</script></html>"))

	list := getJSON[[]domain.Attachment](t, r, "/todos/1/attachments")
	if len(list) != 2 || list[0].FileName != "画面.png" || list[1].FileName != "memo.txt" {
		t.Fatalf("attachments = %+v", list)
	}

	// 画像は inline=1 でブラウザに表示できる
	w = doRequest(t, r, http.MethodGet, "/attachments/1/file?inline=1", "")
	if w.Code != http.StatusOK || !bytes.HasPrefix(w.Body.Bytes(), pngHeader) || w.Header().Get("Content-Type") != "image/png" {
		t.Errorf("download image: status = %d, type = %s", w.Code, w.Header().Get("Content-Type"))
	}
	if got := w.Header().Get("Content-Disposition"); !strings.HasPrefix(got, "inline;") || !strings.Contains(got, "filename*=utf-8''%E7%94%BB%E9%9D%A2.png") {
		t.Errorf("image Content-Disposition = %q", got)
	}
	// 画像以外（HTML など）は inline=1 でもダウンロードさせる
	w = doRequest(t, r, http.MethodGet, "/attachments/2/file?inline=1", "")
	if got := w.Header().Get("Content-Disposition"); !strings.HasPrefix(got, "attachment;") {
		t.Errorf("html Content-Disposition = %q", got)
	}
	if w.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Error("missing nosniff")
	}

	mustStatus(t, r, http.MethodDelete, "/attachments/1", "", http.StatusOK)
	mustStatus(t, r, http.MethodDelete, "/attachments/1", "", http.StatusNotFound)
	mustStatus(t, r, http.MethodGet, "/attachments/1/file", "", http.StatusNotFound)
	if n := storedFiles(t, repos.attachmentsDir); n != 1 {
		t.Errorf("stored files after delete = %d", n)
	}
}

func TestAttachmentHandler_Errors(t *testing.T) {
	r, repos := newAttachmentRouter(t)
	mustStatus(t, r, http.MethodPost, "/todos", `{"title":"a"}`, http.StatusCreated)

	if w := upload(t, r, "/todos/99/attachments", "a.txt", []byte("x")); w.Code != http.StatusNotFound {
		t.Errorf("unknown todo: status = %d", w.Code)
	}
	if w := upload(t, r, "/todos/1/attachments", "empty.txt", nil); w.Code != http.StatusBadRequest {
		t.Errorf("empty file: status = %d", w.Code)
	}
	if w := upload(t, r, "/todos/1/attachments", "big.bin", make([]byte, domain.AttachmentMaxSize+1)); w.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("too large: status = %d, body = %s", w.Code, w.Body)
	}
	// ちょうど上限は添付できる
	if w := upload(t, r, "/todos/1/attachments", "max.bin", make([]byte, domain.AttachmentMaxSize)); w.Code != http.StatusCreated {
		t.Errorf("max size: status = %d, body = %s", w.Code, w.Body)
	}
	mustStatus(t, r, http.MethodPost, "/todos/1/attachments", `{"file":"x"}`, http.StatusBadRequest)
	mustStatus(t, r, http.MethodGet, "/todos/99/attachments", "", http.StatusNotFound)
	if n := storedFiles(t, repos.attachmentsDir); n != 1 {
		t.Errorf("stored files = %d, want 1", n)
	}
}

// 課題を削除すると、その課題とサブタスクの添付ファイル（本体も）が消える
func TestAttachmentHandler_DeleteTodoDeletesAttachments(t *testing.T) {
	r, repos := newAttachmentRouter(t)
	mustStatus(t, r, http.MethodPost, "/todos", `{"title":"parent"}`, http.StatusCreated)
	mustStatus(t, r, http.MethodPost, "/todos", `{"title":"sub","parentId":1}`, http.StatusCreated)
	mustStatus(t, r, http.MethodPost, "/todos", `{"title":"other"}`, http.StatusCreated)
	for _, id := range []string{"1", "2", "3"} {
		if w := upload(t, r, "/todos/"+id+"/attachments", "a.txt", []byte("hello")); w.Code != http.StatusCreated {
			t.Fatalf("upload status = %d", w.Code)
		}
	}

	mustStatus(t, r, http.MethodDelete, "/todos/1", "", http.StatusOK)
	if n := storedFiles(t, repos.attachmentsDir); n != 1 {
		t.Errorf("stored files = %d, want 1", n)
	}
	if list := getJSON[[]domain.Attachment](t, r, "/todos/3/attachments"); len(list) != 1 {
		t.Errorf("other todo attachments = %+v", list)
	}
}
