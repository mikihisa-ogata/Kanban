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
	svc := service.NewTodoService(repo, epicRepo)
	h := handler.NewTodoHandler(svc)
	epicSvc := service.NewEpicService(epicRepo, repo)
	epicHandler := handler.NewEpicHandler(epicSvc)

	r.GET("/todos", h.GetTodos)
	r.POST("/todos", h.CreateTodo)
	r.PUT("/todos/:id", h.UpdateTodo)
	r.DELETE("/todos/:id", h.DeleteTodo)

	r.GET("/epics", epicHandler.GetEpics)
	r.POST("/epics", epicHandler.CreateEpic)
	r.DELETE("/epics/:id", epicHandler.DeleteEpic)

	r.Run(":8080")
}
