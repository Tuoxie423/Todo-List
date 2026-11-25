// Package router 负责路由注册与中间件装配。
package router

import (
	"net/http"
	"time"

	"todo-list/backend/internal/handler"
	"todo-list/backend/internal/middleware"
	"todo-list/backend/internal/wechat"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// New 构建带全部中间件和路由的 gin 引擎。
func New(db *gorm.DB, wx wechat.Exchanger, extraOrigins []string) *gin.Engine {
	engine := gin.New()
	engine.Use(gin.Logger(), gin.Recovery(), middleware.CORS(extraOrigins))

	h := handler.New(db, wx)

	engine.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
			"time":   time.Now().Format(time.RFC3339),
		})
	})

	api := engine.Group("/api")
	api.POST("/auth/login", h.LoginWithPassword)
	api.POST("/auth/wechat-login", h.LoginWithWeChat)

	private := api.Group("")
	private.Use(middleware.RequireAuth(db))
	{
		private.POST("/rooms", h.CreateRoom)

		roomTasks := private.Group("/rooms/:roomID/tasks")
		roomTasks.GET("", h.ListTasks)
		roomTasks.POST("", h.CreateTask)
		roomTasks.PATCH("/:taskID/toggle", h.ToggleTask)
		roomTasks.DELETE("/:taskID", h.DeleteTask)
	}

	return engine
}
