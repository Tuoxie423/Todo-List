// Package database 负责数据库连接与表结构迁移。
package database

import (
	"fmt"
	"log"

	"todo-list/backend/internal/config"
	"todo-list/backend/internal/model"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// Connect 按配置建立 MySQL 连接。
func Connect(cfg config.DB) (*gorm.DB, error) {
	db, err := gorm.Open(mysql.Open(cfg.DSN()), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("数据库连接失败: %w", err)
	}

	log.Println("数据库连接成功")
	return db, nil
}

// Migrate 创建/更新表结构，并清理旧版本遗留的索引。
func Migrate(db *gorm.DB) error {
	if err := db.AutoMigrate(&model.User{}, &model.Room{}, &model.Task{}); err != nil {
		return fmt.Errorf("auto migrate failed: %w", err)
	}

	// 旧版本给清单名建过全局唯一索引，现在改为按用户唯一。
	if db.Migrator().HasIndex(&model.Room{}, "idx_rooms_name") {
		if err := db.Migrator().DropIndex(&model.Room{}, "idx_rooms_name"); err != nil {
			log.Printf("drop old list name index failed: %v", err)
		}
	}

	return nil
}
