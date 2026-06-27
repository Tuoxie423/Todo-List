package global

import (
	"fmt"
	"log"
	"todo-list/backend/config"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var DB *gorm.DB

func Connect() error {
	dsn := config.Global.DB.DSN()
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("数据库连接失败: %w", err)
	}
	DB = db
	log.Println("数据库连接成功")
	return nil
}
