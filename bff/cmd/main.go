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
	sprintRepo := repository.NewSprintMySQLRepository(db)
	versionRepo := repository.NewVersionMySQLRepository(db)
	sprintHandler := handler.NewSprintHandler(service.NewSprintService(sprintRepo, repo, spaceRepo, historyRepo))
	versionHandler := handler.NewVersionHandler(service.NewVersionService(versionRepo, repo, spaceRepo))
	jqlHandler := handler.NewJQLHandler(service.NewJQLService(repo, spaceRepo, statusRepo, userRepo, epicRepo, sprintRepo, versionRepo))
	filterHandler := handler.NewFilterHandler(service.NewFilterService(repository.NewFilterMySQLRepository(db)), service.NewIssueViewService(repository.NewIssueViewMySQLRepository(db), repo))
	bulkHandler := handler.NewBulkHandler(service.NewBulkService(repo, epicRepo, spaceRepo, sprintRepo, versionRepo, userRepo, attachmentRepo, statusRepo, watcherRepo, notificationRepo))
	componentHandler := handler.NewComponentHandler(service.NewComponentService(repository.NewComponentMySQLRepository(db), repo, epicRepo, spaceRepo, userRepo, watcherRepo, historyRepo, statusRepo))

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
	api.GET("/issues/search", h.SearchIssues)
	api.GET("/issues/jql", jqlHandler.SearchJQL)
	api.GET("/issues/:key", h.GetIssue)
	api.POST("/issues/bulk-update", bulkHandler.UpdateIssues)
	api.POST("/issues/bulk-delete", bulkHandler.DeleteIssues)
	api.POST("/todos/:id/view", filterHandler.RecordView)
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

	api.GET("/sprints", sprintHandler.GetSprints)
	api.POST("/spaces/:id/sprints", sprintHandler.CreateSprint)
	api.PUT("/sprints/:id", sprintHandler.UpdateSprint)
	api.POST("/sprints/:id/start", sprintHandler.StartSprint)
	api.POST("/sprints/:id/complete", sprintHandler.CompleteSprint)
	api.DELETE("/sprints/:id", sprintHandler.DeleteSprint)
	api.GET("/sprints/:id/issues", sprintHandler.GetSprintIssues)
	api.GET("/sprints/:id/burndown", sprintHandler.GetBurndown)
	api.PUT("/todos/:id/sprint", sprintHandler.SetTodoSprint)

	api.GET("/versions", versionHandler.GetVersions)
	api.POST("/spaces/:id/versions", versionHandler.CreateVersion)
	api.PUT("/versions/:id", versionHandler.UpdateVersion)
	api.POST("/versions/:id/release", versionHandler.ReleaseVersion)
	api.DELETE("/versions/:id", versionHandler.DeleteVersion)
	api.PUT("/todos/:id/fix-versions", versionHandler.SetTodoVersions)

	api.GET("/components", componentHandler.GetComponents)
	api.POST("/spaces/:id/components", componentHandler.CreateComponent)
	api.PUT("/components/:id", componentHandler.UpdateComponent)
	api.DELETE("/components/:id", componentHandler.DeleteComponent)
	api.PUT("/todos/:id/components", componentHandler.SetTodoComponents)

	api.GET("/filters", filterHandler.GetFilters)
	api.POST("/filters", filterHandler.CreateFilter)
	api.PUT("/filters/:id", filterHandler.UpdateFilter)
	api.PUT("/filters/:id/favourite", filterHandler.SetFavourite)
	api.DELETE("/filters/:id", filterHandler.DeleteFilter)

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
