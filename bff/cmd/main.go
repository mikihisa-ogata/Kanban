package main

import (
	"log"
	"os"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"todo-api/internal/database"
	"todo-api/internal/handler"
	"todo-api/internal/repository"
	"todo-api/internal/service"
)

func main() {
	r := gin.Default()

	// CORS設定
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:5173", "http://localhost:3000"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))

	// docker compose で MySQL と同時に起動しても待てるよう、接続は 30 秒まで再試行する
	db, err := database.Open(database.DSNFromEnv(), 30*time.Second)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	if err := database.Migrate(db); err != nil {
		log.Fatal(err)
	}

	repo := repository.NewTodoMySQLRepository(db)
	epicRepo := repository.NewEpicMySQLRepository(db)
	spaceRepo := repository.NewSpaceMySQLRepository(db)
	linkRepo := repository.NewIssueLinkMySQLRepository(db)
	historyRepo := repository.NewHistoryMySQLRepository(db)
	attachmentRepo := repository.NewAttachmentMySQLRepository(db, attachmentsDir())
	svc := service.NewTodoService(repo, epicRepo, spaceRepo, linkRepo, historyRepo, attachmentRepo)
	h := handler.NewTodoHandler(svc)
	epicSvc := service.NewEpicService(epicRepo, repo, spaceRepo, historyRepo)
	epicHandler := handler.NewEpicHandler(epicSvc)
	commentRepo := repository.NewCommentMySQLRepository(db)
	commentSvc := service.NewCommentService(commentRepo, repo)
	commentHandler := handler.NewCommentHandler(commentSvc)
	attachmentSvc := service.NewAttachmentService(attachmentRepo, repo)
	attachmentHandler := handler.NewAttachmentHandler(attachmentSvc)
	spaceSvc := service.NewSpaceService(spaceRepo, epicRepo, repo, historyRepo)
	spaceHandler := handler.NewSpaceHandler(spaceSvc)

	r.GET("/todos", h.GetTodos)
	r.POST("/todos", h.CreateTodo)
	r.PUT("/todos/:id", h.UpdateTodo)
	r.DELETE("/todos/:id", h.DeleteTodo)
	r.POST("/todos/:id/move", h.MoveTodo)
	r.GET("/issues/:key", h.GetIssue)
	r.GET("/labels", h.GetLabels)
	r.POST("/todos/:id/links", h.AddLink)
	r.DELETE("/links/:id", h.DeleteLink)
	r.GET("/todos/:id/history", h.GetHistory)

	r.GET("/todos/:id/comments", commentHandler.GetComments)
	r.POST("/todos/:id/comments", commentHandler.CreateComment)
	r.PUT("/comments/:id", commentHandler.UpdateComment)
	r.DELETE("/comments/:id", commentHandler.DeleteComment)

	r.GET("/todos/:id/attachments", attachmentHandler.GetAttachments)
	r.POST("/todos/:id/attachments", attachmentHandler.AddAttachment)
	r.GET("/attachments/:id/file", attachmentHandler.DownloadAttachment)
	r.DELETE("/attachments/:id", attachmentHandler.DeleteAttachment)

	r.GET("/epics", epicHandler.GetEpics)
	r.POST("/epics", epicHandler.CreateEpic)
	r.PUT("/epics/:id", epicHandler.UpdateEpic)
	r.DELETE("/epics/:id", epicHandler.DeleteEpic)

	r.GET("/spaces", spaceHandler.GetSpaces)
	r.GET("/spaces/key-suggestion", spaceHandler.SuggestSpaceKey)
	r.POST("/spaces", spaceHandler.CreateSpace)
	r.PUT("/spaces/:id", spaceHandler.UpdateSpace)
	r.DELETE("/spaces/:id", spaceHandler.DeleteSpace)

	if err := r.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}

// attachmentsDir は添付ファイルの保存先のディレクトリを返す。環境変数 ATTACHMENTS_DIR が未設定なら、
// カレントディレクトリ（bff/）の attachments（git 管理外）にする
func attachmentsDir() string {
	if dir := os.Getenv("ATTACHMENTS_DIR"); dir != "" {
		return dir
	}
	return "attachments"
}
