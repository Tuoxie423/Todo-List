package main

import (
	"errors"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"gin-practice/backend/config"
	"gin-practice/backend/global"
	"gin-practice/backend/model"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type createRoomRequest struct {
	Name string `json:"name"`
}

type createTaskRequest struct {
	Title string `json:"title"`
	Level string `json:"level"`
	Kind  string `json:"kind"`
}

func migrate(db *gorm.DB) {
	if err := db.AutoMigrate(&model.Room{}, &model.Task{}); err != nil {
		log.Fatalf("自动迁移失败: %v", err)
	}
}

func main() {
	config.LoadConfig()

	if err := global.Connect(); err != nil {
		log.Fatalf("数据库连接失败: %v", err)
	}

	if config.Global.Server.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}
	log.Println("数据库：", config.Global.DB.DBName)

	migrate(global.DB)
	router := setupRouter(global.DB)

	port := os.Getenv("PORT")
	if port == "" {
		port = config.Global.Server.Port
	}

	if err := router.Run(":" + port); err != nil {
		panic(err)
	}
}

func setupRouter(db *gorm.DB) *gin.Engine {
	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery(), corsMiddleware())

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
			"time":   time.Now().Format(time.RFC3339),
		})
	})

	api := router.Group("/api")
	{
		api.POST("/rooms", func(c *gin.Context) {
			var payload createRoomRequest
			if err := c.ShouldBindJSON(&payload); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"message": "请求体必须是 JSON"})
				return
			}

			name := strings.TrimSpace(payload.Name)
			if name == "" {
				c.JSON(http.StatusBadRequest, gin.H{"message": "房间名称不能为空"})
				return
			}

			room := model.Room{Name: name}
			result := db.Where("name = ?", name).FirstOrCreate(&room)
			if result.Error != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"message": "创建房间失败"})
				return
			}

			status := http.StatusCreated
			if result.RowsAffected == 0 {
				status = http.StatusOK
			}
			c.JSON(status, room)
		})

		roomTasks := api.Group("/rooms/:roomID/tasks")
		{
			roomTasks.GET("", func(c *gin.Context) {
				roomID, ok := parseParamID(c, "roomID", "房间 ID 必须是正整数")
				if !ok {
					return
				}
				if !roomExists(c, db, roomID) {
					return
				}

				var tasks []model.Task
				if err := db.Where("room_id = ?", roomID).
					Order("created_at DESC, id DESC").
					Find(&tasks).Error; err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"message": "查询任务失败"})
					return
				}

				c.JSON(http.StatusOK, gin.H{"items": tasks})
			})

			roomTasks.POST("", func(c *gin.Context) {
				roomID, ok := parseParamID(c, "roomID", "房间 ID 必须是正整数")
				if !ok {
					return
				}
				if !roomExists(c, db, roomID) {
					return
				}

				var payload createTaskRequest
				if err := c.ShouldBindJSON(&payload); err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"message": "请求体必须是 JSON"})
					return
				}

				title := strings.TrimSpace(payload.Title)
				if title == "" {
					c.JSON(http.StatusBadRequest, gin.H{"message": "任务标题不能为空"})
					return
				}

				task := model.Task{
					RoomID: roomID,
					Title:  title,
					Level:  normalizeLevel(payload.Level),
					Kind:   normalizeKind(payload.Kind),
				}

				if err := db.Create(&task).Error; err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"message": "创建任务失败"})
					return
				}
				c.JSON(http.StatusCreated, task)
			})

			roomTasks.PATCH("/:taskID/toggle", func(c *gin.Context) {
				roomID, ok := parseParamID(c, "roomID", "房间 ID 必须是正整数")
				if !ok {
					return
				}
				taskID, ok := parseParamID(c, "taskID", "任务 ID 必须是正整数")
				if !ok {
					return
				}

				var task model.Task
				if err := db.Where("id = ? AND room_id = ?", taskID, roomID).First(&task).Error; err != nil {
					if errors.Is(err, gorm.ErrRecordNotFound) {
						c.JSON(http.StatusNotFound, gin.H{"message": "任务不存在"})
					} else {
						c.JSON(http.StatusInternalServerError, gin.H{"message": "查询任务失败"})
					}
					return
				}

				task.Done = !task.Done
				if err := db.Save(&task).Error; err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"message": "更新任务失败"})
					return
				}

				c.JSON(http.StatusOK, task)
			})

			roomTasks.DELETE("/:taskID", func(c *gin.Context) {
				roomID, ok := parseParamID(c, "roomID", "房间 ID 必须是正整数")
				if !ok {
					return
				}
				taskID, ok := parseParamID(c, "taskID", "任务 ID 必须是正整数")
				if !ok {
					return
				}

				result := db.Where("id = ? AND room_id = ?", taskID, roomID).Delete(&model.Task{})
				if result.Error != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"message": "删除任务失败"})
					return
				}
				if result.RowsAffected == 0 {
					c.JSON(http.StatusNotFound, gin.H{"message": "任务不存在"})
					return
				}
				c.Status(http.StatusNoContent)
			})
		}
	}

	return router
}

func parseParamID(c *gin.Context, name string, message string) (int, bool) {
	id, err := strconv.Atoi(c.Param(name))
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"message": message})
		return 0, false
	}

	return id, true
}

func roomExists(c *gin.Context, db *gorm.DB, roomID int) bool {
	var count int64
	if err := db.Model(&model.Room{}).Where("id = ?", roomID).Count(&count).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "查询房间失败"})
		return false
	}
	if count == 0 {
		c.JSON(http.StatusNotFound, gin.H{"message": "房间不存在"})
		return false
	}

	return true
}

func normalizeLevel(level string) string {
	level = strings.TrimSpace(level)
	switch level {
	case "基础", "进阶", "挑战":
		return level
	default:
		return "基础"
	}
}

func normalizeKind(kind string) string {
	kind = strings.TrimSpace(kind)
	if kind == "optimization" {
		return kind
	}

	return "learning"
}

func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET,POST,PATCH,DELETE,OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
