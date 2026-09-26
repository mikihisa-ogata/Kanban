package main

import (
	"log"
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
	svc := service.NewTodoService(repo, epicRepo, spaceRepo)
	h := handler.NewTodoHandler(svc)
	epicSvc := service.NewEpicService(epicRepo, repo, spaceRepo)
	epicHandler := handler.NewEpicHandler(epicSvc)
	spaceSvc := service.NewSpaceService(spaceRepo, epicRepo, repo)
	spaceHandler := handler.NewSpaceHandler(spaceSvc)

	r.GET("/todos", h.GetTodos)
	r.POST("/todos", h.CreateTodo)
	r.PUT("/todos/:id", h.UpdateTodo)
	r.DELETE("/todos/:id", h.DeleteTodo)
	r.POST("/todos/:id/move", h.MoveTodo)

	r.GET("/epics", epicHandler.GetEpics)
	r.POST("/epics", epicHandler.CreateEpic)
	r.PUT("/epics/:id", epicHandler.UpdateEpic)
	r.DELETE("/epics/:id", epicHandler.DeleteEpic)

	r.GET("/spaces", spaceHandler.GetSpaces)
	r.POST("/spaces", spaceHandler.CreateSpace)
	r.DELETE("/spaces/:id", spaceHandler.DeleteSpace)

	if err := r.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}
