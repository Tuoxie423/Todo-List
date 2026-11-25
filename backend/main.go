package main

import (
	"log"
	"os"
	"strings"

	"todo-list/backend/internal/config"
	"todo-list/backend/internal/database"
	"todo-list/backend/internal/router"
	"todo-list/backend/internal/wechat"

	"github.com/gin-gonic/gin"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}
	log.Println("config loaded: mode =", cfg.Server.Mode, "port =", cfg.Server.Port)

	if cfg.Server.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	db, err := database.Connect(cfg.DB)
	if err != nil {
		log.Fatalf("database connection failed: %v", err)
	}
	log.Println("database:", cfg.DB.DBName)

	if err := database.Migrate(db); err != nil {
		log.Fatalf("%v", err)
	}

	wx := wechat.NewClient(cfg.WeChat.AppID, cfg.WeChat.AppSecret)
	engine := router.New(db, wx, splitOrigins(os.Getenv("ALLOWED_ORIGINS")))

	port := cfg.Server.Port
	if envPort := os.Getenv("PORT"); envPort != "" {
		port = envPort
	}

	if err := engine.Run(":" + port); err != nil {
		panic(err)
	}
}

// splitOrigins 把逗号分隔的 CORS 白名单拆成切片。
func splitOrigins(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return strings.Split(value, ",")
}
