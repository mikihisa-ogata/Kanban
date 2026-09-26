package main

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

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

	repo := repository.NewTodoRepository()
	epicRepo := repository.NewEpicRepository()
	spaceRepo := repository.NewSpaceRepository()
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

	r.GET("/epics", epicHandler.GetEpics)
	r.POST("/epics", epicHandler.CreateEpic)
	r.PUT("/epics/:id", epicHandler.UpdateEpic)
	r.DELETE("/epics/:id", epicHandler.DeleteEpic)

	r.GET("/spaces", spaceHandler.GetSpaces)
	r.POST("/spaces", spaceHandler.CreateSpace)
	r.DELETE("/spaces/:id", spaceHandler.DeleteSpace)

	r.Run(":8080")
}
