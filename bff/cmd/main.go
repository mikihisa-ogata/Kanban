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
	watcherRepo := repository.NewWatcherMySQLRepository(db)
	notificationRepo := repository.NewNotificationMySQLRepository(db)
	// 課題の変更履歴を記録すると、ウォッチャーに通知を作る
	historyRepo := service.WithNotifications(repository.NewHistoryMySQLRepository(db), watcherRepo, notificationRepo)
	attachmentRepo := repository.NewAttachmentMySQLRepository(db, attachmentsDir())
	statusRepo := repository.NewStatusMySQLRepository(db)
	userRepo := repository.NewUserMySQLRepository(db)
	svc := service.NewTodoService(repo, epicRepo, spaceRepo, linkRepo, historyRepo, attachmentRepo, statusRepo, userRepo, watcherRepo, notificationRepo)
	h := handler.NewTodoHandler(svc)
	epicSvc := service.NewEpicService(epicRepo, repo, spaceRepo, historyRepo, statusRepo)
	epicHandler := handler.NewEpicHandler(epicSvc)
	commentRepo := repository.NewCommentMySQLRepository(db)
	commentSvc := service.NewCommentService(commentRepo, repo, watcherRepo, notificationRepo, userRepo)
	commentHandler := handler.NewCommentHandler(commentSvc)
	attachmentSvc := service.NewAttachmentService(attachmentRepo, repo)
	attachmentHandler := handler.NewAttachmentHandler(attachmentSvc)
	worklogRepo := repository.NewWorklogMySQLRepository(db)
	worklogSvc := service.NewWorklogService(worklogRepo, repo, epicRepo, spaceRepo, historyRepo, statusRepo)
	worklogHandler := handler.NewWorklogHandler(worklogSvc)
	spaceSvc := service.NewSpaceService(spaceRepo, epicRepo, repo, historyRepo, statusRepo)
	spaceHandler := handler.NewSpaceHandler(spaceSvc)
	authSvc := service.NewAuthService(userRepo, repository.NewSessionMySQLRepository(db))
	authHandler := handler.NewAuthHandler(authSvc)
	userHandler := handler.NewUserHandler(service.NewUserService(userRepo))
	notificationHandler := handler.NewNotificationHandler(service.NewNotificationService(notificationRepo))
	watcherHandler := handler.NewWatcherHandler(service.NewWatcherService(watcherRepo, repo))

	r.POST("/auth/signup", authHandler.Signup)
	r.POST("/auth/login", authHandler.Login)
	r.POST("/auth/logout", authHandler.Logout)
	r.GET("/auth/me", authHandler.Me)

	// ここから下の API はログインが必要（未ログインなら 401）
	api := r.Group("/", authHandler.RequireLogin)
	api.GET("/users", userHandler.GetUsers)

	api.GET("/notifications", notificationHandler.GetNotifications)
	api.POST("/notifications/read-all", notificationHandler.MarkAllRead)
	api.POST("/notifications/:id/read", notificationHandler.MarkRead)
	api.GET("/todos/:id/watchers", watcherHandler.GetWatchers)
	api.POST("/todos/:id/watch", watcherHandler.Watch)
	api.DELETE("/todos/:id/watch", watcherHandler.Unwatch)

	api.GET("/todos", h.GetTodos)
	api.POST("/todos", h.CreateTodo)
	api.PUT("/todos/:id", h.UpdateTodo)
	api.DELETE("/todos/:id", h.DeleteTodo)
	api.POST("/todos/:id/move", h.MoveTodo)
	api.GET("/issues/:key", h.GetIssue)
	api.GET("/labels", h.GetLabels)
	api.POST("/todos/:id/links", h.AddLink)
	api.DELETE("/links/:id", h.DeleteLink)
	api.GET("/todos/:id/history", h.GetHistory)

	api.GET("/todos/:id/comments", commentHandler.GetComments)
	api.POST("/todos/:id/comments", commentHandler.CreateComment)
	api.PUT("/comments/:id", commentHandler.UpdateComment)
	api.DELETE("/comments/:id", commentHandler.DeleteComment)

	api.GET("/todos/:id/attachments", attachmentHandler.GetAttachments)
	api.POST("/todos/:id/attachments", attachmentHandler.AddAttachment)
	api.GET("/attachments/:id/file", attachmentHandler.DownloadAttachment)
	api.DELETE("/attachments/:id", attachmentHandler.DeleteAttachment)

	api.GET("/todos/:id/worklogs", worklogHandler.GetWorklogs)
	api.POST("/todos/:id/worklogs", worklogHandler.AddWorklog)
	api.PUT("/worklogs/:id", worklogHandler.UpdateWorklog)
	api.DELETE("/worklogs/:id", worklogHandler.DeleteWorklog)

	api.GET("/epics", epicHandler.GetEpics)
	api.POST("/epics", epicHandler.CreateEpic)
	api.PUT("/epics/:id", epicHandler.UpdateEpic)
	api.DELETE("/epics/:id", epicHandler.DeleteEpic)

	api.GET("/spaces", spaceHandler.GetSpaces)
	api.GET("/spaces/key-suggestion", spaceHandler.SuggestSpaceKey)
	api.POST("/spaces", spaceHandler.CreateSpace)
	api.PUT("/spaces/:id", spaceHandler.UpdateSpace)
	api.DELETE("/spaces/:id", spaceHandler.DeleteSpace)
	api.GET("/statuses", spaceHandler.GetStatuses)
	api.POST("/spaces/:id/statuses", spaceHandler.CreateStatus)
	api.PUT("/spaces/:id/statuses/order", spaceHandler.ReorderStatuses)
	api.PUT("/statuses/:id", spaceHandler.RenameStatus)
	api.PUT("/statuses/:id/wip-limit", spaceHandler.SetStatusWIPLimit)
	api.DELETE("/statuses/:id", spaceHandler.DeleteStatus)

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
